package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func routeFor(t *testing.T, s *store, apiKey, model string, available []string) pluginapi.ModelRouteResponse {
	t.Helper()
	headers := http.Header{}
	if apiKey != "" {
		headers.Set("Authorization", "Bearer "+apiKey)
	}
	raw, errMarshal := json.Marshal(pluginapi.ModelRouteRequest{
		RequestedModel:     model,
		Headers:            headers,
		AvailableProviders: available,
	})
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	out, errRoute := routeModel(s, raw)
	if errRoute != nil {
		t.Fatal(errRoute)
	}
	var env struct {
		OK     bool                         `json:"ok"`
		Result pluginapi.ModelRouteResponse `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(out, &env); errUnmarshal != nil || !env.OK {
		t.Fatalf("bad envelope %s: %v", out, errUnmarshal)
	}
	return env.Result
}

func TestRouteModelMappings(t *testing.T) {
	cfg, errParse := parseConfig([]byte(`
state_file: ` + t.TempDir() + `/state.json
model_mappings:
  - from: "gpt-4o*"
    to: "gpt-5-mini"
  - from: "fast"
    to: "claude-haiku-4-5"
    provider: "Claude"
keys:
  - key: "sk-special"
    model_mappings:
      - from: "gpt-4o*"
        to: "gpt-5"
        provider: "codex"
`))
	if errParse != nil {
		t.Fatal(errParse)
	}
	s := newStore(nil)
	if errConfigure := s.configure(cfg); errConfigure != nil {
		t.Fatal(errConfigure)
	}
	available := []string{"claude", "codex", "gemini-cli"}

	got := routeFor(t, s, "sk-other", "GPT-4o-mini", available)
	if !got.Handled || got.Target != "codex" || got.TargetModel != "gpt-5-mini" {
		t.Fatalf("global rule with inferred provider: %+v", got)
	}

	got = routeFor(t, s, "sk-special", "gpt-4o", available)
	if got.Target != "codex" || got.TargetModel != "gpt-5" {
		t.Fatalf("per-key rule should win: %+v", got)
	}

	got = routeFor(t, s, "", "fast(high)", available)
	if got.Target != "claude" || got.TargetModel != "claude-haiku-4-5(high)" {
		t.Fatalf("thinking suffix should carry over: %+v", got)
	}

	if got = routeFor(t, s, "sk-other", "claude-sonnet-4-5", available); got.Handled {
		t.Fatalf("unmatched model must not be handled: %+v", got)
	}

	if got = routeFor(t, s, "sk-other", "gpt-4o", []string{"claude", "vertex"}); got.Handled {
		t.Fatalf("uninferable provider must not be handled: %+v", got)
	}

	usage := s.usageSnapshot(s.now())
	for _, item := range usage.Keys {
		if item.ID == keyID("sk-special") && item.Mapped != 1 {
			t.Fatalf("mapped counter = %d, want 1", item.Mapped)
		}
	}
	if len(usage.Providers) != 2 {
		t.Fatalf("providers should reflect last route call: %v", usage.Providers)
	}
}

func TestMappingValidation(t *testing.T) {
	for _, doc := range []string{
		"model_mappings:\n  - from: \"a\"\n",
		"model_mappings:\n  - to: \"b\"\n",
		"model_mappings:\n  - from: \"[\"\n    to: \"b\"\n",
		"model_mappings:\n  - from: \"a\"\n    to: \"b*\"\n",
	} {
		if _, errParse := parseConfig([]byte(doc)); errParse == nil {
			t.Fatalf("expected error for %q", doc)
		}
	}
}
