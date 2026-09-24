package main

import (
	"testing"
	"time"
)

func TestDeleteKeysRemovesTrackedAndReportsConfigured(t *testing.T) {
	s := newRotateStore(t)
	plain := keyID("sk-plain")
	configured := keyID("sk-configured-old")
	s.record(plain, maskKey("sk-plain"), "gpt-5", counters{Tokens: 7, Requests: 1}, time.Now())
	s.record(configured, "", "gpt-5", counters{Requests: 1}, time.Now())

	result := s.deleteKeys(targetIDs(plain, []string{configured, plain, "missing"}))
	if len(result.Deleted) != 2 || len(result.Configured) != 1 || result.Configured[0] != configured {
		t.Fatalf("unexpected result %+v", result)
	}
	if len(result.Missing) != 1 || result.Missing[0] != "missing" {
		t.Fatalf("unknown id must be reported: %+v", result.Missing)
	}
	if _, ok := s.state.Keys[plain]; ok {
		t.Fatal("tracked key must be removed")
	}
	if _, ok := s.state.Keys[configured]; ok {
		t.Fatal("configured key's usage must be removed")
	}
	view := s.usageSnapshot(time.Now())
	if len(view.Keys) != 1 || view.Keys[0].ID != configured || view.Keys[0].Periods[periodTotal].Used.Requests != 0 {
		t.Fatalf("only the configured key may remain, with zeroed usage: %+v", view.Keys)
	}
}

func TestBatchLimitsAndReset(t *testing.T) {
	s := newRotateStore(t)
	a, b := keyID("sk-a"), keyID("sk-b")
	s.record(a, "", "", counters{Requests: 3}, time.Now())
	s.record(b, "", "", counters{Requests: 4}, time.Now())

	disabled := true
	if errApply := s.applyLimits(limitsRequest{IDs: []string{a, b}, Disabled: &disabled}); errApply != nil {
		t.Fatal(errApply)
	}
	for _, id := range []string{a, b} {
		if entry := s.state.Keys[id]; entry.Disabled == nil || !*entry.Disabled {
			t.Fatalf("%s must be paused", id)
		}
	}

	if errReset := s.resetUsage(resetRequest{IDs: []string{a, b}, Period: "total"}, time.Now()); errReset != nil {
		t.Fatal(errReset)
	}
	for _, id := range []string{a, b} {
		if used := s.state.Keys[id].Total.Counters.Requests; used != 0 {
			t.Fatalf("%s total must be reset, got %d", id, used)
		}
	}
	if errReset := s.resetUsage(resetRequest{IDs: []string{a}, Period: "weekly"}, time.Now()); errReset == nil {
		t.Fatal("invalid period must be rejected")
	}
}
