package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// apiKeyFromHeaders mirrors the header credentials accepted by the built-in
// config access provider. Query-string credentials are not visible to request
// interceptors, so those requests are accounted for but never blocked.
func apiKeyFromHeaders(headers http.Header) string {
	if headers == nil {
		return ""
	}
	if authorization := strings.TrimSpace(headers.Get("Authorization")); authorization != "" {
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "bearer") {
			if token := strings.TrimSpace(parts[1]); token != "" {
				return token
			}
		} else {
			return authorization
		}
	}
	if googleKey := strings.TrimSpace(headers.Get("X-Goog-Api-Key")); googleKey != "" {
		return googleKey
	}
	return strings.TrimSpace(headers.Get("X-Api-Key"))
}

// violation describes the first quota rule that a request would exceed.
type violation struct {
	Period    string
	Dimension string
	Used      float64
	Limit     float64
}

func (v violation) message() string {
	if v.Dimension == "cost_usd" {
		return fmt.Sprintf("api key quota exceeded: %s cost %.4f/%.4f USD", v.Period, v.Used, v.Limit)
	}
	return fmt.Sprintf("api key quota exceeded: %s %s %d/%d", v.Period, v.Dimension, int64(v.Used), int64(v.Limit))
}

// checkBucket reports the first exceeded dimension of one period.
func checkBucket(period string, used counters, limit limits) (violation, bool) {
	if limit.Tokens > 0 && used.Tokens >= limit.Tokens {
		return violation{Period: period, Dimension: "tokens", Used: float64(used.Tokens), Limit: float64(limit.Tokens)}, true
	}
	if limit.Requests > 0 && used.Requests >= limit.Requests {
		return violation{Period: period, Dimension: "requests", Used: float64(used.Requests), Limit: float64(limit.Requests)}, true
	}
	if limit.CostUSD > 0 && used.CostUSD >= limit.CostUSD {
		return violation{Period: period, Dimension: "cost_usd", Used: used.CostUSD, Limit: limit.CostUSD}, true
	}
	return violation{}, false
}

// decision is the outcome of a quota evaluation for one request.
type decision struct {
	Allowed  bool
	Disabled bool
	// Outside reports that the schedule refuses the key right now; Window
	// names the blocking window, or is empty outside every window.
	Outside   bool
	Window    string
	Violation violation
	RetryIn   time.Duration
}

// evaluate resolves whether a key may issue another request right now.
func (s *store) evaluate(id string, at time.Time) decision {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := s.entryLocked(id, "", false)
	effective, disabled := s.effectiveLimitsLocked(id, entry)
	if disabled {
		return decision{Disabled: true}
	}
	sched, _ := s.scheduleLocked(id, entry)
	active, inWindow, blocked := sched.blockedAt(at, s.cfg.location)
	if blocked {
		result := decision{Outside: true, Window: active.Window.Name}
		if next, ok := sched.nextAllowed(at, s.cfg.location); ok {
			result.RetryIn = next.Sub(at)
		}
		return result
	}
	if entry == nil {
		return decision{Allowed: true}
	}
	s.rollLocked(entry, at)
	if inWindow {
		if found, exceeded := checkBucket("window "+active.Window.Name, windowUsed(entry, active), active.Window.Limits); exceeded {
			return decision{Violation: found, RetryIn: active.End.Sub(at)}
		}
	}

	checks := []struct {
		period string
		used   counters
		limit  limits
	}{
		{periodDaily, entry.Daily.Counters, effective.Daily},
		{periodMonthly, entry.Monthly.Counters, effective.Monthly},
		{periodTotal, entry.Total.Counters, effective.Total},
	}
	for _, check := range checks {
		found, exceeded := checkBucket(check.period, check.used, check.limit)
		if !exceeded {
			continue
		}
		result := decision{Violation: found}
		if next, ok := resetAt(check.period, at, s.cfg.location); ok {
			if remaining := next.Sub(at); remaining > 0 {
				result.RetryIn = remaining
			}
		}
		return result
	}
	return decision{Allowed: true}
}

// modelBlocked reports the deny rule that forbids a model for one key.
func (s *store) modelBlocked(id, model string) (string, bool) {
	if strings.TrimSpace(model) == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entryLocked(id, "", false)
	return matchPattern(s.blockedModelsLocked(id, entry), model)
}

// requestedModel prefers the model the client asked for over the upstream model
// chosen later, so a deny rule matches what the caller actually wrote.
func requestedModel(req pluginapi.RequestInterceptRequest) string {
	if model := strings.TrimSpace(req.RequestedModel); model != "" {
		return model
	}
	return strings.TrimSpace(req.Model)
}

// interceptBeforeAuth enforces quota and model rules before the request reaches
// any upstream, and counts every admitted request as in flight until the host
// reports its completion.
func interceptBeforeAuth(s *store, raw []byte) ([]byte, error) {
	var req pluginapi.RequestInterceptRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	passThrough := pluginapi.RequestInterceptResponse{Headers: req.Headers, Body: req.Body}
	apiKey := apiKeyFromHeaders(req.Headers)
	if apiKey == "" {
		return okEnvelope(passThrough)
	}

	id := keyID(apiKey)
	if _, configured := cfg.byID[id]; !configured && !cfg.TrackUnknownKeys {
		return okEnvelope(passThrough)
	}

	now := s.now()
	hint := maskKey(apiKey)
	if !cfg.Enforce {
		s.admit(req.RequestID, id, hint, now, false)
		return okEnvelope(passThrough)
	}
	result := s.evaluate(id, now)
	if result.Disabled {
		s.countBlocked(id, hint, now)
		return rejectionEnvelope(http.StatusForbidden, "insufficient_quota", "api_key_disabled", "api key is disabled by quota policy", 0)
	}
	if result.Outside {
		s.countBlocked(id, hint, now)
		message := "api key may not be used at this time (outside every schedule window)"
		if result.Window != "" {
			message = fmt.Sprintf("api key may not be used at this time (schedule window %s)", result.Window)
		}
		return rejectionEnvelope(http.StatusForbidden, "insufficient_quota", "outside_schedule", message, result.RetryIn)
	}

	model := requestedModel(req)
	if pattern, blocked := s.modelBlocked(id, model); blocked {
		s.countBlocked(id, hint, now)
		return rejectionEnvelope(http.StatusForbidden, "insufficient_quota", "model_blocked",
			fmt.Sprintf("model %s is not allowed for this api key (rule %s)", model, pattern), 0)
	}

	if !result.Allowed {
		s.countBlocked(id, hint, now)
		return rejectionEnvelope(cfg.RejectStatus, "insufficient_quota", "quota_exceeded", result.Violation.message(), result.RetryIn)
	}
	if violation, admitted := s.admitQueued(req.RequestID, id, hint, true); !admitted {
		s.countBlocked(id, hint, now)
		return rejectionEnvelope(http.StatusTooManyRequests, "rate_limit_error", "rate_limit_exceeded", violation.message(), violation.RetryIn)
	}
	return okEnvelope(passThrough)
}

// rejectionEnvelope builds a terminated interceptor response.
func rejectionEnvelope(status int, errorType, code, message string, retryIn time.Duration) ([]byte, error) {
	body, errMarshal := json.Marshal(map[string]any{
		"error": map[string]any{
			"type":    errorType,
			"code":    code,
			"message": message,
			"param":   nil,
		},
	})
	if errMarshal != nil {
		return nil, errMarshal
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	if retryIn > 0 {
		seconds := int64(math.Ceil(retryIn.Seconds()))
		headers.Set("Retry-After", strconv.FormatInt(seconds, 10))
	}
	return okEnvelope(pluginapi.RequestInterceptResponse{
		Terminate:       true,
		StatusCode:      status,
		ResponseHeaders: headers,
		ResponseBody:    body,
	})
}

// passThroughRequest returns the request unchanged.
func passThroughRequest(raw []byte) ([]byte, error) {
	var req pluginapi.RequestInterceptRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	return okEnvelope(pluginapi.RequestInterceptResponse{Headers: req.Headers, Body: req.Body})
}
