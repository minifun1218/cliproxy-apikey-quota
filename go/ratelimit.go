package main

import (
	"fmt"
	"time"
)

// rateWindow is the sliding window the per-minute rate limits are measured over.
const rateWindow = time.Minute

// rateLimits caps how fast one key may be used. Like limits, zero inherits,
// a negative value means unlimited, and a positive value is the ceiling.
type rateLimits struct {
	// RPM caps the requests admitted during the last minute.
	RPM int64 `yaml:"rpm" json:"rpm"`
	// TPM caps the tokens reported during the last minute. Tokens are only
	// known once a request completes, so the ceiling rejects the requests that
	// follow a burst rather than the request that crosses it.
	TPM int64 `yaml:"tpm" json:"tpm"`
	// Concurrency caps the requests in flight at the same time.
	Concurrency int64 `yaml:"concurrency" json:"concurrency"`
	// QueueSeconds is how long a request waits for a free slot when the key,
	// or every account it may use, is at its concurrency limit, before it is
	// rejected. Zero inherits and a negative value rejects at once.
	QueueSeconds int64 `yaml:"queue_seconds" json:"queue_seconds"`
}

// merge overlays non-zero fields of other on top of the receiver.
func (r rateLimits) merge(other rateLimits) rateLimits {
	if other.RPM != 0 {
		r.RPM = other.RPM
	}
	if other.TPM != 0 {
		r.TPM = other.TPM
	}
	if other.Concurrency != 0 {
		r.Concurrency = other.Concurrency
	}
	if other.QueueSeconds != 0 {
		r.QueueSeconds = other.QueueSeconds
	}
	return r
}

// queueWait is how long a request of a key with these limits may wait for a
// concurrency slot.
func queueWait(rate rateLimits) time.Duration {
	if rate.QueueSeconds <= 0 {
		return 0
	}
	return time.Duration(rate.QueueSeconds) * time.Second
}

type tokenSample struct {
	At     time.Time
	Tokens int64
}

// rateTrack holds the recent activity of one key. It lives in memory only:
// a minute-long window is meaningless after a restart.
type rateTrack struct {
	requests []time.Time
	tokens   []tokenSample
}

func (t *rateTrack) prune(at time.Time) {
	cutoff := at.Add(-rateWindow)
	drop := 0
	for drop < len(t.requests) && !t.requests[drop].After(cutoff) {
		drop++
	}
	t.requests = t.requests[drop:]
	drop = 0
	for drop < len(t.tokens) && !t.tokens[drop].At.After(cutoff) {
		drop++
	}
	t.tokens = t.tokens[drop:]
}

func (t *rateTrack) tokenSum() int64 {
	var sum int64
	for _, sample := range t.tokens {
		sum += sample.Tokens
	}
	return sum
}

// rateLocked returns the pruned activity of a key, creating it when asked.
func (s *store) rateLocked(id string, at time.Time, create bool) *rateTrack {
	track := s.rates[id]
	if track == nil {
		if !create {
			return nil
		}
		if s.rates == nil {
			s.rates = map[string]*rateTrack{}
		}
		track = &rateTrack{}
		s.rates[id] = track
	}
	track.prune(at)
	return track
}

// rateUsageLocked reports the requests and tokens of the last minute.
func (s *store) rateUsageLocked(id string, at time.Time) (int64, int64) {
	track := s.rateLocked(id, at, false)
	if track == nil {
		return 0, 0
	}
	return int64(len(track.requests)), track.tokenSum()
}

// recordTokensLocked adds a completed request's tokens to the TPM window.
func (s *store) recordTokensLocked(id string, tokens int64, at time.Time) {
	if tokens <= 0 {
		return
	}
	track := s.rateLocked(id, at, true)
	track.tokens = append(track.tokens, tokenSample{At: at, Tokens: tokens})
}

// baseRateLocked resolves the configured rate limits of a key overlaid with
// its runtime override, before any schedule window applies.
func (s *store) baseRateLocked(id string, entry *keyState) rateLimits {
	effective := s.cfg.DefaultRateLimits
	if configured, ok := s.cfg.byID[id]; ok {
		effective = effective.merge(configured.RateLimits)
	}
	if entry != nil && entry.RateOverrides != nil {
		effective = effective.merge(*entry.RateOverrides)
	}
	return effective
}

// effectiveRateLocked adds the rate limits of the window active at the given
// time on top of the base rate limits.
func (s *store) effectiveRateLocked(id string, entry *keyState, at time.Time) rateLimits {
	effective := s.baseRateLocked(id, entry)
	if active, ok := s.activeWindowLocked(id, entry, at); ok {
		effective = effective.merge(active.Window.RateLimits)
	}
	return effective
}

// rateViolation describes the first rate limit a request would exceed.
type rateViolation struct {
	Dimension string
	Used      int64
	Limit     int64
	RetryIn   time.Duration
}

func (v rateViolation) message() string {
	return fmt.Sprintf("api key rate limit exceeded: %s %d/%d", v.Dimension, v.Used, v.Limit)
}

// admit checks the rate limits of a key and, when they allow it, counts the
// request as admitted and in flight in the same critical section, so parallel
// requests cannot slip past a ceiling together. A request ID that is already
// in flight, as on a host retry, is admitted again without being counted twice.
func (s *store) admit(requestID, id, hint string, at time.Time, enforce bool) (rateViolation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.admitLocked(requestID, id, hint, at, enforce)
}

// admitQueued admits a request like admit, but when only the key's
// concurrency limit stands in the way it waits for a slot to free up, for up
// to the key's queue time, before giving up.
func (s *store) admitQueued(requestID, id, hint string, enforce bool) (rateViolation, bool) {
	var deadline time.Time
	for {
		s.mu.Lock()
		now := s.now()
		violation, admitted := s.admitLocked(requestID, id, hint, now, enforce)
		if admitted || violation.Dimension != "concurrency" {
			s.mu.Unlock()
			return violation, admitted
		}
		if deadline.IsZero() {
			wait := queueWait(s.effectiveRateLocked(id, s.state.Keys[id], now))
			deadline = time.Now().Add(wait)
		}
		signal := s.releaseSignalLocked()
		s.mu.Unlock()

		remaining := time.Until(deadline)
		if remaining <= 0 || !waitForRelease(signal, remaining) {
			return violation, false
		}
	}
}

func (s *store) admitLocked(requestID, id, hint string, at time.Time, enforce bool) (rateViolation, bool) {
	if requestID != "" {
		if _, known := s.inflight[requestID]; known {
			return rateViolation{}, true
		}
	}
	entry := s.entryLocked(id, hint, true)
	track := s.rateLocked(id, at, true)
	if enforce {
		limit := s.effectiveRateLocked(id, entry, at)
		if violation, exceeded := checkRate(track, limit, s.concurrencyLocked(at)[id], at); exceeded {
			return violation, false
		}
	}
	track.requests = append(track.requests, at)
	s.beginRequestLocked(requestID, id, at)
	return rateViolation{}, true
}

// checkRate reports the first exceeded rate limit with the wait until it frees.
func checkRate(track *rateTrack, limit rateLimits, inflight int, at time.Time) (rateViolation, bool) {
	if limit.Concurrency > 0 && int64(inflight) >= limit.Concurrency {
		return rateViolation{Dimension: "concurrency", Used: int64(inflight), Limit: limit.Concurrency, RetryIn: time.Second}, true
	}
	if limit.RPM > 0 && int64(len(track.requests)) >= limit.RPM {
		// The oldest admissions leave the window first; the one that brings
		// the count back under the ceiling decides the wait.
		index := int64(len(track.requests)) - limit.RPM
		wait := track.requests[index].Add(rateWindow).Sub(at)
		return rateViolation{Dimension: "rpm", Used: int64(len(track.requests)), Limit: limit.RPM, RetryIn: wait}, true
	}
	if limit.TPM > 0 {
		sum := track.tokenSum()
		if sum >= limit.TPM {
			wait := time.Duration(0)
			remaining := sum
			for _, sample := range track.tokens {
				remaining -= sample.Tokens
				wait = sample.At.Add(rateWindow).Sub(at)
				if remaining < limit.TPM {
					break
				}
			}
			return rateViolation{Dimension: "tpm", Used: sum, Limit: limit.TPM, RetryIn: wait}, true
		}
	}
	return rateViolation{}, false
}
