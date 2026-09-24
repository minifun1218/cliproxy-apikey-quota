package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestRPMLimitSlidesWithTheMinute(t *testing.T) {
	clock := monday
	s := newClockStore(t, "default_rate_limits:\n  rpm: 2\n", &clock)
	if status, _, _ := intercept(t, s, "sk-a", "r1"); status != 0 {
		t.Fatal("first request must pass")
	}
	clock = clock.Add(20 * time.Second)
	if status, _, _ := intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatal("second request must pass")
	}
	clock = clock.Add(10 * time.Second)
	status, code, retry := intercept(t, s, "sk-a", "r3")
	if status != 429 || code != "rate_limit_exceeded" || retry != "30" {
		t.Fatalf("third request must wait for the first to leave the window: %d %s %s", status, code, retry)
	}
	// Retrying an admitted request ID is not counted again.
	if status, _, _ = intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatalf("a host retry of an admitted request must pass, got %d", status)
	}
	if status, _, _ = intercept(t, s, "sk-b", "r4"); status != 0 {
		t.Fatal("other keys have their own window")
	}
	clock = clock.Add(31 * time.Second)
	if status, _, _ = intercept(t, s, "sk-a", "r5"); status != 0 {
		t.Fatal("the window must have moved on")
	}
}

func TestTPMLimitCountsCompletedTokens(t *testing.T) {
	clock := monday
	s := newClockStore(t, "default_rate_limits:\n  tpm: 1000\n", &clock)
	id := keyID("sk-a")
	s.record(id, "", "gpt-5", counters{Tokens: 600, Requests: 1}, clock)
	clock = clock.Add(15 * time.Second)
	s.record(id, "", "gpt-5", counters{Tokens: 500, Requests: 1}, clock)
	status, code, retry := intercept(t, s, "sk-a", "r1")
	if status != 429 || code != "rate_limit_exceeded" || retry != "45" {
		t.Fatalf("TPM over the ceiling must refuse until the first sample expires: %d %s %s", status, code, retry)
	}
	clock = clock.Add(46 * time.Second)
	if status, _, _ = intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatal("expired tokens must free the window")
	}
}

func TestConcurrencyLimitReleasesOnCompletion(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
default_rate_limits:
  concurrency: 1
keys:
  - key: sk-a
    rate_limits:
      concurrency: 2
`, &clock)
	if status, _, _ := intercept(t, s, "sk-a", "r1"); status != 0 {
		t.Fatal("first in-flight request must pass")
	}
	if status, _, _ := intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatal("the key's own ceiling of 2 must override the default")
	}
	if status, code, _ := intercept(t, s, "sk-a", "r3"); status != 429 || code != "rate_limit_exceeded" {
		t.Fatalf("third concurrent request must be refused: %d %s", status, code)
	}
	raw, _ := json.Marshal(pluginapi.RequestCompletion{RequestID: "r1"})
	if err := handleRequestComplete(s, raw); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := intercept(t, s, "sk-a", "r3"); status != 0 {
		t.Fatal("a completed request must free its slot")
	}
}

func TestRuntimeRateOverrideAndUnlimited(t *testing.T) {
	clock := monday
	s := newClockStore(t, "default_rate_limits:\n  rpm: 1\n", &clock)
	id := keyID("sk-a")
	if err := s.applyLimits(limitsRequest{IDs: []string{id}, RateLimits: &rateLimits{RPM: -1}}); err != nil {
		t.Fatal(err)
	}
	for _, request := range []string{"r1", "r2", "r3"} {
		if status, _, _ := intercept(t, s, "sk-a", request); status != 0 {
			t.Fatalf("-1 must lift the default ceiling, %s got %d", request, status)
		}
	}
	view := s.usageSnapshot(clock)
	if view.Keys[0].Rate.RPM.Used != 3 || view.Keys[0].Rate.RPM.Limit != nil || view.Keys[0].BaseRateLimits.RPM != 0 {
		t.Fatalf("usage view must report 3 unlimited requests: %+v", view.Keys[0].Rate)
	}
}
