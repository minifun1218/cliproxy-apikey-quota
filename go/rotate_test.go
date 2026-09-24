package main

import (
	"errors"
	"testing"
	"time"
)

func newRotateStore(t *testing.T) *store {
	t.Helper()
	cfg, errParse := parseConfig([]byte(`
state_file: ` + t.TempDir() + `/state.json
keys:
  - key: "sk-configured-old"
    label: "billing"
    note: "from yaml"
    blocked_models: ["o1*"]
    limits:
      daily:
        tokens: 1000
`))
	if errParse != nil {
		t.Fatal(errParse)
	}
	s := newStore(nil)
	if errConfigure := s.configure(cfg); errConfigure != nil {
		t.Fatal(errConfigure)
	}
	return s
}

func TestRotateKeyMovesUsageAndOverrides(t *testing.T) {
	s := newRotateStore(t)
	oldID := keyID("sk-plain-old")
	s.record(oldID, maskKey("sk-plain-old"), "gpt-5", counters{Tokens: 42, Requests: 1}, time.Now())
	note := "team a"
	if errApply := s.applyLimits(limitsRequest{ID: oldID, Note: &note}); errApply != nil {
		t.Fatal(errApply)
	}

	result, errRotate := s.rotateKey(rotateRequest{ID: oldID, NewKey: "sk-plain-new"})
	if errRotate != nil {
		t.Fatal(errRotate)
	}
	if result.ID != keyID("sk-plain-new") || result.Configured || result.Hint != "sk-p...-new" {
		t.Fatalf("unexpected result %+v", result)
	}
	if _, stillThere := s.state.Keys[oldID]; stillThere {
		t.Fatal("old entry must be removed")
	}
	moved := s.state.Keys[result.ID]
	if moved == nil || moved.Total.Counters.Tokens != 42 || moved.Note == nil || *moved.Note != note {
		t.Fatalf("usage and note must move: %+v", moved)
	}
}

func TestRotateKeyCarriesConfiguredSettings(t *testing.T) {
	s := newRotateStore(t)
	oldID := keyID("sk-configured-old")

	result, errRotate := s.rotateKey(rotateRequest{ID: oldID, NewKey: "sk-configured-new"})
	if errRotate != nil {
		t.Fatal(errRotate)
	}
	if !result.Configured {
		t.Fatal("rotation of a configured key must be reported")
	}
	moved := s.state.Keys[result.ID]
	if moved.Overrides == nil || moved.Overrides.Daily.Tokens != 1000 {
		t.Fatalf("configured limits must become overrides: %+v", moved.Overrides)
	}
	if moved.Note == nil || *moved.Note != "from yaml" || moved.Label != "billing" {
		t.Fatalf("configured note and label must carry over: %+v", moved)
	}
	if moved.BlockedModels == nil || len(*moved.BlockedModels) != 1 {
		t.Fatalf("configured deny list must carry over: %+v", moved.BlockedModels)
	}
	if got := s.blockedModelsLocked(result.ID, moved); len(got) != 1 || got[0] != "o1*" {
		t.Fatalf("new key must be denied the configured models: %v", got)
	}
}

func TestRotateKeyRejectsBadTargets(t *testing.T) {
	s := newRotateStore(t)
	oldID := keyID("sk-a")
	s.record(oldID, "", "", counters{Requests: 1}, time.Now())
	s.record(keyID("sk-b"), "", "", counters{Requests: 1}, time.Now())

	if _, err := s.rotateKey(rotateRequest{ID: oldID, NewKey: "sk-a"}); !errors.Is(err, errRotateSameKey) {
		t.Fatalf("same key: %v", err)
	}
	if _, err := s.rotateKey(rotateRequest{ID: oldID, NewKey: "sk-b"}); !errors.Is(err, errRotateTaken) {
		t.Fatalf("taken key: %v", err)
	}
	if _, err := s.rotateKey(rotateRequest{ID: "missing", NewKey: "sk-c"}); !errors.Is(err, errRotateUnknown) {
		t.Fatalf("unknown id: %v", err)
	}
}
