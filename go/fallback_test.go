package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const fallbackConfig = `
model_mappings:
  - from: "fast"
    to: "claude-haiku-4-5"
    provider: "claude"
fallback_models:
  - "gpt-5-mini"
  - "gemini-2.5-flash"
keys:
  - key: "sk-own"
    fallback_models:
      - "claude-haiku-4-5"
  - key: "sk-plain"
`

func fallbackStore(t *testing.T, extra string) *store {
	t.Helper()
	cfg, errParse := parseConfig([]byte("state_file: " + t.TempDir() + "/state.json\n" + fallbackConfig + extra))
	if errParse != nil {
		t.Fatal(errParse)
	}
	s := newStore(nil)
	if errConfigure := s.configure(cfg); errConfigure != nil {
		t.Fatal(errConfigure)
	}
	return s
}

// withFakeHost marks host callbacks available and answers them with handle.
func withFakeHost(t *testing.T, handle func(method string, request []byte) (any, error)) {
	t.Helper()
	previousAPI := hostAPI.Load()
	previousCall := hostCall
	var marker byte
	setHostAPI(unsafe.Pointer(&marker))
	hostCall = func(method string, request any, out any) error {
		raw, errMarshal := json.Marshal(request)
		if errMarshal != nil {
			return errMarshal
		}
		result, errHandle := handle(method, raw)
		if errHandle != nil {
			return errHandle
		}
		if out == nil || result == nil {
			return nil
		}
		encoded, errEncode := json.Marshal(result)
		if errEncode != nil {
			return errEncode
		}
		return json.Unmarshal(encoded, out)
	}
	t.Cleanup(func() {
		hostAPI.Store(previousAPI)
		hostCall = previousCall
	})
}

func executorPayload(t *testing.T, apiKey, model, streamID string) []byte {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+apiKey)
	raw, errMarshal := json.Marshal(executorRequest{
		ExecutorRequest: pluginapi.ExecutorRequest{
			Model:        model,
			Format:       "openai",
			SourceFormat: "openai",
			Stream:       streamID != "",
			Headers:      headers,
			Payload:      []byte(`{"model":"` + model + `"}`),
			Metadata:     map[string]any{"requested_model": model},
		},
		StreamID:       streamID,
		HostCallbackID: "cb-1",
	})
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}
	return raw
}

func TestFallbackConfigValidation(t *testing.T) {
	dir := t.TempDir()
	if _, errParse := parseConfig([]byte("state_file: " + dir + "/s.json\nfallback_models: [\"gpt-*\"]\n")); errParse == nil {
		t.Fatal("a glob fallback model must be rejected")
	}
	if _, errParse := parseConfig([]byte("state_file: " + dir + "/s.json\nfallback_status_codes: [200]\n")); errParse == nil {
		t.Fatal("a non-error fallback status must be rejected")
	}
}

func TestPlanRouteFallbacks(t *testing.T) {
	s := fallbackStore(t, "")
	headers := func(key string) http.Header {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+key)
		return h
	}

	plan, _ := s.planRoute(headers("sk-plain"), "claude-sonnet-5(high)", nil)
	if plan.model != "claude-sonnet-5(high)" || plan.provider != "" {
		t.Fatalf("unmapped primary: %+v", plan)
	}
	if strings.Join(plan.fallbacks, ",") != "gpt-5-mini(high),gemini-2.5-flash(high)" {
		t.Fatalf("global chain with suffix: %v", plan.fallbacks)
	}

	plan, _ = s.planRoute(headers("sk-own"), "claude-sonnet-5", nil)
	if strings.Join(plan.fallbacks, ",") != "claude-haiku-4-5" {
		t.Fatalf("own chain should replace the global one: %v", plan.fallbacks)
	}

	// The mapped target is the primary and is not tried twice.
	plan, _ = s.planRoute(headers("sk-own"), "fast", nil)
	if plan.model != "claude-haiku-4-5" || plan.provider != "claude" || len(plan.fallbacks) != 0 {
		t.Fatalf("mapped primary equal to the only fallback: %+v", plan)
	}
}

func TestRouteModelSendsFallbackKeysToExecutor(t *testing.T) {
	s := fallbackStore(t, "")

	// Without host callbacks the executor cannot forward, so mapping still applies.
	if got := routeFor(t, s, "sk-plain", "fast", []string{"claude"}); got.TargetKind != pluginapi.ModelRouteTargetProvider {
		t.Fatalf("no host: %+v", got)
	}

	withFakeHost(t, func(string, []byte) (any, error) { return nil, nil })
	got := routeFor(t, s, "sk-plain", "fast", []string{"claude"})
	if !got.Handled || got.TargetKind != pluginapi.ModelRouteTargetSelf {
		t.Fatalf("fallback key should route to the executor: %+v", got)
	}

	// An empty own list turns fallback off, and the mapping answers again.
	if errApply := s.applyLimits(limitsRequest{ID: keyID("sk-plain"), FallbackModels: &[]string{}}); errApply != nil {
		t.Fatal(errApply)
	}
	if got = routeFor(t, s, "sk-plain", "fast", []string{"claude"}); got.TargetKind != pluginapi.ModelRouteTargetProvider || got.TargetModel != "claude-haiku-4-5" {
		t.Fatalf("fallback turned off: %+v", got)
	}
	if got = routeFor(t, s, "sk-plain", "claude-sonnet-5", nil); got.Handled {
		t.Fatalf("no mapping and no fallback: %+v", got)
	}

	if errApply := s.applyLimits(limitsRequest{ID: keyID("sk-plain"), ClearFallbacks: true}); errApply != nil {
		t.Fatal(errApply)
	}
	if got = routeFor(t, s, "sk-plain", "claude-sonnet-5", nil); got.TargetKind != pluginapi.ModelRouteTargetSelf {
		t.Fatalf("cleared override should restore the global chain: %+v", got)
	}
}

func TestFallbackSkipsTokenCounting(t *testing.T) {
	withFakeHost(t, func(string, []byte) (any, error) { return nil, nil })
	req := pluginapi.ModelRouteRequest{SourceFormat: "claude", Metadata: map[string]any{"request_path": "/v1/messages/count_tokens"}}
	if fallbackEligible(req) {
		t.Fatal("count_tokens must not reach the executor")
	}
	req.Metadata["request_path"] = "/v1/messages"
	if !fallbackEligible(req) {
		t.Fatal("messages should reach the executor")
	}
}

func TestExecuteFallsBackOnRetryableStatus(t *testing.T) {
	s := fallbackStore(t, "")
	var tried []string
	withFakeHost(t, func(method string, raw []byte) (any, error) {
		if method != pluginabi.MethodHostModelExecute {
			t.Fatalf("unexpected method %s", method)
		}
		var req hostModelRequest
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			t.Fatal(errUnmarshal)
		}
		if req.HostCallbackID != "cb-1" || req.EntryProtocol != "openai" {
			t.Fatalf("callback context not forwarded: %+v", req)
		}
		tried = append(tried, req.Model+"@"+req.ForcedProvider)
		switch req.Model {
		case "claude-haiku-4-5":
			return nil, &hostError{Message: "overloaded", Status: 529}
		case "gpt-5-mini":
			return nil, &hostError{Message: "no auth available"}
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: 200, Body: []byte(`{"ok":true}`)}, nil
	})

	out, errExecute := executeWithFallback(s, executorPayload(t, "sk-plain", "fast", ""))
	if errExecute != nil {
		t.Fatal(errExecute)
	}
	var env struct {
		OK     bool                       `json:"ok"`
		Result pluginapi.ExecutorResponse `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(out, &env); errUnmarshal != nil || !env.OK {
		t.Fatalf("bad envelope %s", out)
	}
	if string(env.Result.Payload) != `{"ok":true}` || env.Result.Headers.Get(fallbackHeader) != "gemini-2.5-flash" {
		t.Fatalf("unexpected result: %+v", env.Result)
	}
	if strings.Join(tried, ",") != "claude-haiku-4-5@claude,gpt-5-mini@,gemini-2.5-flash@" {
		t.Fatalf("attempt order: %v", tried)
	}
	if got := s.state.Keys[keyID("sk-plain")].Fallbacks; got != 1 {
		t.Fatalf("fallback counter = %d", got)
	}
}

func TestExecuteStopsOnClientError(t *testing.T) {
	s := fallbackStore(t, "")
	calls := 0
	withFakeHost(t, func(string, []byte) (any, error) {
		calls++
		return nil, &hostError{Code: "invalid_request", Message: "bad request", Status: 400}
	})
	out, errExecute := executeWithFallback(s, executorPayload(t, "sk-plain", "claude-sonnet-5", ""))
	if errExecute != nil {
		t.Fatal(errExecute)
	}
	var env pluginabi.Envelope
	if errUnmarshal := json.Unmarshal(out, &env); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if env.OK || env.Error == nil || env.Error.HTTPStatus != 400 || calls != 1 {
		t.Fatalf("a 400 must be returned without fallback: %s (calls %d)", out, calls)
	}
}

func TestExecuteStreamFallsBackAndPumps(t *testing.T) {
	s := fallbackStore(t, "")
	var mu sync.Mutex
	var emitted []string
	closed := make(chan string, 1)
	reads := 0
	withFakeHost(t, func(method string, raw []byte) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		switch method {
		case pluginabi.MethodHostModelExecuteStream:
			var req hostModelRequest
			_ = json.Unmarshal(raw, &req)
			if !req.Stream {
				t.Errorf("stream flag lost")
			}
			if req.Model == "claude-sonnet-5" {
				return nil, &hostError{Message: "rate limited", Status: 429}
			}
			return pluginapi.HostModelStreamResponse{StatusCode: 200, StreamID: "up-" + req.Model}, nil
		case pluginabi.MethodHostModelStreamRead:
			reads++
			if reads == 1 {
				return pluginapi.HostModelStreamReadResponse{Payload: []byte("data: a\n\n")}, nil
			}
			return pluginapi.HostModelStreamReadResponse{Done: true}, nil
		case pluginabi.MethodHostStreamEmit:
			var req streamEmitRequest
			_ = json.Unmarshal(raw, &req)
			emitted = append(emitted, req.StreamID+":"+string(req.Payload))
		case pluginabi.MethodHostStreamClose:
			var req streamCloseRequest
			_ = json.Unmarshal(raw, &req)
			closed <- req.StreamID + ":" + req.Error
		case pluginabi.MethodHostModelStreamClose:
		default:
			t.Errorf("unexpected method %s", method)
		}
		return nil, nil
	})

	out, errExecute := executeStreamWithFallback(s, executorPayload(t, "sk-plain", "claude-sonnet-5", "down-1"))
	if errExecute != nil {
		t.Fatal(errExecute)
	}
	var env struct {
		OK     bool                 `json:"ok"`
		Result executorStreamResult `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(out, &env); errUnmarshal != nil || !env.OK {
		t.Fatalf("bad envelope %s", out)
	}
	if env.Result.Headers.Get(fallbackHeader) != "gpt-5-mini" {
		t.Fatalf("fallback header missing: %+v", env.Result)
	}
	select {
	case got := <-closed:
		if got != "down-1:" {
			t.Fatalf("stream closed with %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream was never closed")
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(emitted, "|") != "down-1:data: a\n\n" {
		t.Fatalf("emitted %q", emitted)
	}
}

func TestGlobalFallbacksOverride(t *testing.T) {
	s := fallbackStore(t, "")
	if errApply := s.applyGlobalFallbacks(fallbacksRequest{FallbackModels: []string{" gpt-5 ", "GPT-5", ""}}); errApply != nil {
		t.Fatal(errApply)
	}
	view := s.usageSnapshot(s.now())
	if strings.Join(view.FallbackModels, ",") != "gpt-5" || !view.FallbacksCustom {
		t.Fatalf("override not applied: %v %v", view.FallbackModels, view.FallbacksCustom)
	}
	if errApply := s.applyGlobalFallbacks(fallbacksRequest{Clear: true}); errApply != nil {
		t.Fatal(errApply)
	}
	view = s.usageSnapshot(s.now())
	if strings.Join(view.FallbackModels, ",") != "gpt-5-mini,gemini-2.5-flash" || view.FallbacksCustom {
		t.Fatalf("clear should restore the configured chain: %v", view.FallbackModels)
	}
	if errApply := s.applyGlobalFallbacks(fallbacksRequest{FallbackModels: []string{"gpt-*"}}); errApply == nil {
		t.Fatal("a glob must be rejected")
	}
}
