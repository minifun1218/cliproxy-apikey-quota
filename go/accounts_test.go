package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func candidates(ids ...string) []pluginapi.SchedulerAuthCandidate {
	out := make([]pluginapi.SchedulerAuthCandidate, 0, len(ids))
	for _, id := range ids {
		out = append(out, pluginapi.SchedulerAuthCandidate{ID: id, Provider: "codex"})
	}
	return out
}

// pick runs one scheduler pick for a key and returns the chosen account, or
// the HTTP status and code of the failure. An empty account with status zero
// means the pick was left to the host.
func pick(t *testing.T, s *store, key string, list []pluginapi.SchedulerAuthCandidate) (string, int, string) {
	t.Helper()
	headers := http.Header{}
	if key != "" {
		headers.Set("Authorization", "Bearer "+key)
	}
	raw, _ := json.Marshal(pluginapi.SchedulerPickRequest{Model: "gpt-5", Options: pluginapi.SchedulerOptions{Headers: headers}, Candidates: list})
	out, errPick := schedulerPick(s, raw)
	if errPick != nil {
		t.Fatal(errPick)
	}
	var envelope pluginabi.Envelope
	if errUnmarshal := json.Unmarshal(out, &envelope); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !envelope.OK {
		return "", envelope.Error.HTTPStatus, envelope.Error.Code
	}
	var resp pluginapi.SchedulerPickResponse
	if errUnmarshal := json.Unmarshal(envelope.Result, &resp); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !resp.Handled {
		return "", 0, ""
	}
	return resp.AuthID, 0, ""
}

// runOn reports to the store that a request runs on an account, as the host
// does through the after-auth interceptor.
func runOn(t *testing.T, s *store, requestID, authID string) {
	t.Helper()
	raw, _ := json.Marshal(pluginapi.RequestInterceptRequest{RequestID: requestID, Metadata: map[string]any{selectedAuthKey: authID}})
	if _, errIntercept := interceptAfterAuth(s, raw); errIntercept != nil {
		t.Fatal(errIntercept)
	}
}

func complete(t *testing.T, s *store, requestID string) {
	t.Helper()
	raw, _ := json.Marshal(pluginapi.RequestCompletion{RequestID: requestID})
	if errComplete := handleRequestComplete(s, raw); errComplete != nil {
		t.Fatal(errComplete)
	}
}

func TestPickLeftToHostWhenNothingRestricts(t *testing.T) {
	clock := monday
	s := newClockStore(t, "keys:\n  - key: sk-a\n", &clock)
	if authID, status, _ := pick(t, s, "sk-a", candidates("a.json", "b.json")); authID != "" || status != 0 {
		t.Fatalf("an unbound key without account rules must keep the host scheduler: %q %d", authID, status)
	}
	if authID, status, _ := pick(t, s, "", candidates("a.json")); authID != "" || status != 0 {
		t.Fatalf("a request without a key must keep the host scheduler: %q %d", authID, status)
	}
}

func TestBoundKeyUsesItsAccounts(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
keys:
  - key: sk-a
    accounts: ["team-*"]
  - key: sk-strict
    accounts: ["team-*"]
    strict_accounts: true
`, &clock)
	for i := 0; i < 4; i++ {
		authID, _, _ := pick(t, s, "sk-a", candidates("pool.json", "team-1.json", "team-2.json"))
		if authID != "team-1.json" && authID != "team-2.json" {
			t.Fatalf("a bound key must be scheduled on its accounts, got %q", authID)
		}
	}
	if authID, _, _ := pick(t, s, "sk-a", candidates("pool.json")); authID != "pool.json" {
		t.Fatalf("a non-strict key must spill over to the pool, got %q", authID)
	}
	if _, status, code := pick(t, s, "sk-strict", candidates("pool.json")); status != http.StatusServiceUnavailable || code != "account_unavailable" {
		t.Fatalf("a strict key without a usable bound account must fail: %d %s", status, code)
	}
}

func TestExclusiveAccountsServeOnlyTheirKeys(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
accounts:
  - match: "vip.json"
    exclusive: true
keys:
  - key: sk-vip
    accounts: ["vip.json"]
`, &clock)
	for i := 0; i < 3; i++ {
		if authID, _, _ := pick(t, s, "sk-other", candidates("vip.json", "pool.json")); authID != "pool.json" {
			t.Fatalf("an exclusive account must not serve other keys, got %q", authID)
		}
	}
	if authID, _, _ := pick(t, s, "sk-vip", candidates("vip.json", "pool.json")); authID != "vip.json" {
		t.Fatalf("the bound key must get its exclusive account, got %q", authID)
	}
	if _, status, code := pick(t, s, "sk-other", candidates("vip.json")); status != http.StatusServiceUnavailable || code != "account_unavailable" {
		t.Fatalf("only an exclusive account left must fail for other keys: %d %s", status, code)
	}
}

func TestAccountConcurrencyCapsAndReleases(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
accounts:
  - match: "*.json"
    concurrency: 1
`, &clock)
	first, _, _ := pick(t, s, "sk-a", candidates("a.json", "b.json"))
	runOn(t, s, "r1", first)
	second, _, _ := pick(t, s, "sk-b", candidates("a.json", "b.json"))
	if second == first || second == "" {
		t.Fatalf("the second request must go to the free account, got %q after %q", second, first)
	}
	runOn(t, s, "r2", second)
	if _, status, code := pick(t, s, "sk-c", candidates("a.json", "b.json")); status != http.StatusTooManyRequests || code != "account_busy" {
		t.Fatalf("every account at its limit must refuse without a queue: %d %s", status, code)
	}
	// A host retry of the same request on another account moves its slot.
	runOn(t, s, "r1", second)
	got := s.accountsSnapshot(nil, nil, clock)
	if len(got.Accounts) != 1 || got.Accounts[0].ID != second || got.Accounts[0].Concurrent != 2 {
		t.Fatalf("a retry must move the slot, not add one: %+v", got.Accounts)
	}
	complete(t, s, "r1")
	complete(t, s, "r2")
	if authID, _, _ := pick(t, s, "sk-c", candidates("a.json", "b.json")); authID == "" {
		t.Fatal("completed requests must free their accounts")
	}
}

func TestPickPrefersHigherPriority(t *testing.T) {
	clock := monday
	s := newClockStore(t, "accounts:\n  - match: \"*\"\n    concurrency: 1\n", &clock)
	list := []pluginapi.SchedulerAuthCandidate{{ID: "low", Priority: 0}, {ID: "high", Priority: 5}}
	if authID, _, _ := pick(t, s, "sk-a", list); authID != "high" {
		t.Fatalf("the highest priority account must win, got %q", authID)
	}
	if authID, _, _ := pick(t, s, "sk-a", list); authID != "low" {
		t.Fatalf("a full high priority account must give way, got %q", authID)
	}
}

func TestBusyPickWaitsForRelease(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
default_rate_limits:
  queue_seconds: 5
accounts:
  - match: "a.json"
    concurrency: 1
`, &clock)
	first, _, _ := pick(t, s, "sk-a", candidates("a.json"))
	runOn(t, s, "r1", first)
	done := make(chan string, 1)
	go func() {
		authID, _, _ := pick(t, s, "sk-a", candidates("a.json"))
		done <- authID
	}()
	select {
	case <-done:
		t.Fatal("the pick must wait while the account is busy")
	case <-time.After(50 * time.Millisecond):
	}
	complete(t, s, "r1")
	select {
	case authID := <-done:
		if authID != "a.json" {
			t.Fatalf("the queued pick must get the freed account, got %q", authID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a release must wake the queued pick")
	}
}

func TestKeyConcurrencyQueues(t *testing.T) {
	clock := monday
	s := newClockStore(t, "default_rate_limits:\n  concurrency: 1\n  queue_seconds: 5\n", &clock)
	if status, _, _ := intercept(t, s, "sk-a", "r1"); status != 0 {
		t.Fatal("first request must pass")
	}
	done := make(chan int, 1)
	go func() {
		status, _, _ := intercept(t, s, "sk-a", "r2")
		done <- status
	}()
	select {
	case <-done:
		t.Fatal("the second request must queue for the slot")
	case <-time.After(50 * time.Millisecond):
	}
	complete(t, s, "r1")
	select {
	case status := <-done:
		if status != 0 {
			t.Fatalf("the queued request must be admitted, got %d", status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a completion must wake the queued request")
	}
}

func TestKeyConcurrencyQueueTimesOut(t *testing.T) {
	clock := monday
	s := newClockStore(t, "default_rate_limits:\n  concurrency: 1\n  queue_seconds: 1\n", &clock)
	intercept(t, s, "sk-a", "r1")
	started := time.Now()
	status, code, _ := intercept(t, s, "sk-a", "r2")
	if status != http.StatusTooManyRequests || code != "rate_limit_exceeded" {
		t.Fatalf("an expired wait must reject: %d %s", status, code)
	}
	if waited := time.Since(started); waited < 900*time.Millisecond {
		t.Fatalf("the request must wait for its queue time, waited %v", waited)
	}
}

func TestRuntimeBindingAndRules(t *testing.T) {
	clock := monday
	s := newClockStore(t, "keys:\n  - key: sk-a\n    accounts: [\"a.json\"]\n", &clock)
	id := keyID("sk-a")
	accounts := []string{"b.json"}
	strict := true
	if err := s.applyLimits(limitsRequest{ID: id, Accounts: &accounts, StrictAccounts: &strict}); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := pick(t, s, "sk-a", candidates("a.json")); status != http.StatusServiceUnavailable {
		t.Fatal("the runtime binding must replace the configured one")
	}
	view := s.usageSnapshot(clock)
	if len(view.Keys[0].Accounts) != 1 || view.Keys[0].Accounts[0] != "b.json" || !view.Keys[0].StrictAccounts || !view.Keys[0].Overridden {
		t.Fatalf("usage view must report the runtime binding: %+v", view.Keys[0])
	}
	if err := s.applyLimits(limitsRequest{ID: id, ClearAccounts: true}); err != nil {
		t.Fatal(err)
	}
	if authID, _, _ := pick(t, s, "sk-a", candidates("a.json", "c.json")); authID != "a.json" {
		t.Fatalf("clearing must restore the configured binding, got %q", authID)
	}

	if err := s.applyAccountRules(accountRulesRequest{Rules: []accountRule{{Match: "c.json", Exclusive: true}}}); err != nil {
		t.Fatal(err)
	}
	if authID, _, _ := pick(t, s, "sk-b", candidates("c.json", "d.json")); authID != "d.json" {
		t.Fatalf("runtime rules must apply, got %q", authID)
	}
	if err := s.applyAccountRules(accountRulesRequest{Rules: []accountRule{{Match: "["}}}); err == nil {
		t.Fatal("an invalid pattern must be rejected")
	}
	if err := s.applyLimits(limitsRequest{ID: id, Accounts: &[]string{"["}}); err == nil {
		t.Fatal("an invalid binding pattern must be rejected")
	}
}

func TestAccountsSnapshotListsHostAccounts(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
accounts:
  - match: "team-*"
    concurrency: 2
    exclusive: true
keys:
  - key: sk-a
    label: Team A
    accounts: ["team-*"]
`, &clock)
	runOn(t, s, "r1", "team-1.json")
	runOn(t, s, "r2", "gone.json")
	entries := []pluginapi.HostAuthFileEntry{
		{ID: "team-1.json", Name: "team-1.json", Provider: "codex", Email: "a@example.com"},
		{ID: "pool.json", Name: "pool.json", Type: "claude"},
	}
	view := s.accountsSnapshot(entries, nil, clock)
	if !view.Listed || len(view.Accounts) != 3 {
		t.Fatalf("host accounts plus busy unknown ones must be listed: %+v", view)
	}
	team := view.Accounts[0]
	if team.Concurrent != 1 || team.Limit == nil || *team.Limit != 2 || !team.Exclusive || len(team.BoundKeys) != 1 || team.BoundKeys[0] != "Team A" {
		t.Fatalf("team account view is wrong: %+v", team)
	}
	if view.Accounts[1].Provider != "claude" || view.Accounts[1].Limit != nil {
		t.Fatalf("pool account view is wrong: %+v", view.Accounts[1])
	}
	if view.Accounts[2].ID != "gone.json" || view.Accounts[2].Concurrent != 1 {
		t.Fatalf("an account serving requests must be listed even when the host omits it: %+v", view.Accounts[2])
	}

	failed := s.accountsSnapshot(nil, errors.New("host callbacks are unavailable"), clock)
	if failed.Listed || failed.ListError == "" {
		t.Fatalf("a failed host listing must be reported: %+v", failed)
	}
}

func TestExecutorAttemptHoldsAccountUntilItEnds(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
fallback_models: ["gpt-5-mini"]
accounts:
  - match: "a.json"
    concurrency: 1
`, &clock)
	var tagged string
	var busyDuringCall int
	withFakeHost(t, func(method string, request []byte) (any, error) {
		var forwarded hostModelRequest
		if errUnmarshal := json.Unmarshal(request, &forwarded); errUnmarshal != nil {
			return nil, errUnmarshal
		}
		tagged = forwarded.Headers.Get(attemptHeader)
		// The host picks an account for the forwarded request, which carries
		// the attempt tag, and never confirms it through the interceptors.
		raw, _ := json.Marshal(pluginapi.SchedulerPickRequest{Options: pluginapi.SchedulerOptions{Headers: forwarded.Headers}, Candidates: candidates("a.json")})
		if _, errPick := schedulerPick(s, raw); errPick != nil {
			return nil, errPick
		}
		if _, status, _ := pick(t, s, "sk-other", candidates("a.json")); status != 0 {
			busyDuringCall = status
		}
		return pluginapi.HostModelExecutionResponse{StatusCode: 200, Body: []byte(`{}`)}, nil
	})
	if _, errExec := executeWithFallback(s, executorPayload(t, "sk-a", "gpt-5", "")); errExec != nil {
		t.Fatal(errExec)
	}
	if tagged == "" {
		t.Fatal("the forwarded request must carry the attempt tag")
	}
	if busyDuringCall != http.StatusTooManyRequests {
		t.Fatalf("the attempt must hold the account while it runs, got %d", busyDuringCall)
	}
	if authID, status, _ := pick(t, s, "sk-other", candidates("a.json")); authID != "a.json" || status != 0 {
		t.Fatalf("the account must be free once the attempt ended: %q %d", authID, status)
	}
}

func TestConfigRejectsInvalidAccountPatterns(t *testing.T) {
	if _, err := parseConfig([]byte("accounts:\n  - match: \"[\"\n")); err == nil {
		t.Fatal("an invalid account rule pattern must be rejected")
	}
	if _, err := parseConfig([]byte("keys:\n  - key: sk-a\n    accounts: [\"[\"]\n")); err == nil {
		t.Fatal("an invalid binding pattern must be rejected")
	}
}
