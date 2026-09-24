package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const tokensPerPriceUnit = 1_000_000

// tokensOf reports the token count charged against the token quota. Providers
// that report a total use it directly; the rest fall back to input + output.
func tokensOf(detail pluginapi.UsageDetail) int64 {
	if detail.TotalTokens > 0 {
		return detail.TotalTokens
	}
	total := detail.InputTokens + detail.OutputTokens
	if total < 0 {
		return 0
	}
	return total
}

// costOf converts a usage detail into USD using the price entry of the model.
// Cache and reasoning prices default to zero so a plain input/output price
// table never double counts tokens that providers already fold into those two
// counters.
func costOf(price modelPrice, detail pluginapi.UsageDetail) float64 {
	units := float64(detail.InputTokens)*price.Input +
		float64(detail.OutputTokens)*price.Output +
		float64(detail.ReasoningTokens)*price.Reasoning +
		float64(detail.CacheReadTokens)*price.CacheRead +
		float64(detail.CacheCreationTokens)*price.CacheWrite
	if units == 0 {
		return 0
	}
	return units / tokensPerPriceUnit
}

// handleUsage accumulates one completed usage record against its API key.
func handleUsage(s *store, raw []byte) error {
	var record pluginapi.UsageRecord
	if errUnmarshal := json.Unmarshal(raw, &record); errUnmarshal != nil {
		return errUnmarshal
	}

	apiKey := strings.TrimSpace(record.APIKey)
	if apiKey == "" {
		return nil
	}

	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	id := keyID(apiKey)
	if _, configured := cfg.byID[id]; !configured && !cfg.TrackUnknownKeys {
		return nil
	}
	if record.Failed && !cfg.CountFailedRequests {
		return nil
	}

	at := record.RequestedAt
	if at.IsZero() {
		at = s.now()
	}
	price, _ := s.priceForModel(record.Model)
	delta := counters{
		Tokens:   tokensOf(record.Detail),
		Requests: 1,
		CostUSD:  costOf(price, record.Detail),
	}
	s.record(id, maskKey(apiKey), record.Model, delta, at)
	return nil
}

// resetAt reports when the given period rolls over, for Retry-After hints.
func resetAt(period string, at time.Time, location *time.Location) (time.Time, bool) {
	local := at.In(location)
	switch period {
	case periodDaily:
		next := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location).AddDate(0, 0, 1)
		return next, true
	case periodMonthly:
		next := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location).AddDate(0, 1, 0)
		return next, true
	default:
		return time.Time{}, false
	}
}
