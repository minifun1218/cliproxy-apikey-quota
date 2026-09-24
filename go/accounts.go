package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const routeAccounts = routePrefix + "/accounts"

// selectedAuthKey is the metadata key the host fills with the credential it
// picked, before the after-auth interceptor runs.
const selectedAuthKey = "selected_auth_id"

// accountReserveTTL bounds how long a pick holds a slot on an account before
// the after-auth interceptor confirms it. The two normally follow each other
// within milliseconds; a pick whose request never reaches the upstream frees
// the slot on expiry.
const accountReserveTTL = 30 * time.Second

// attemptHeader tags each request the fallback executor forwards through the
// host. The host skips this plugin's interceptors on those requests, so no
// after-auth confirmation or completion ever arrives for them; the scheduler
// holds their account slot under the tag instead, and the executor releases it
// when the forwarded call ends.
const attemptHeader = "X-Apikey-Quota-Attempt"

// newAttemptID returns a random tag for one forwarded request.
func newAttemptID() string {
	var raw [12]byte
	if _, errRead := rand.Read(raw[:]); errRead != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw[:])
}

// attemptSlot is the in-flight key an attempt's account slot is held under,
// kept apart from host request IDs.
func attemptSlot(attemptID string) string {
	return "attempt:" + attemptID
}

// accountRule applies to the host credentials ("accounts") whose ID matches
// Match. Concurrency caps the requests an account serves at the same time,
// zero or less meaning unlimited. An exclusive account only serves the API
// keys bound to it. The first matching rule wins.
type accountRule struct {
	Match       string `yaml:"match" json:"match"`
	Concurrency int64  `yaml:"concurrency" json:"concurrency"`
	Exclusive   bool   `yaml:"exclusive" json:"exclusive"`
}

// cleanAccountRules trims every rule, drops rules without a pattern, and keeps
// only the first rule for each pattern so rule order stays meaningful.
func cleanAccountRules(rules []accountRule) []accountRule {
	if len(rules) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rules))
	out := make([]accountRule, 0, len(rules))
	for _, rule := range rules {
		rule.Match = strings.TrimSpace(rule.Match)
		if rule.Match == "" {
			continue
		}
		if rule.Concurrency < 0 {
			rule.Concurrency = 0
		}
		lowered := strings.ToLower(rule.Match)
		if _, ok := seen[lowered]; ok {
			continue
		}
		seen[lowered] = struct{}{}
		out = append(out, rule)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validateAccountRules rejects rules that cannot be applied, naming the offender.
func validateAccountRules(field string, rules []accountRule) error {
	for index, rule := range rules {
		match := strings.TrimSpace(rule.Match)
		if match == "" {
			if rule.Concurrency != 0 || rule.Exclusive {
				return fmt.Errorf("%s[%d].match must not be empty", field, index)
			}
			continue
		}
		if _, errMatch := path.Match(strings.ToLower(match), ""); errMatch != nil {
			return fmt.Errorf("%s[%d].match %q is not a valid pattern", field, index, match)
		}
	}
	return nil
}

// validateAccountPatterns rejects binding patterns that are not valid globs.
func validateAccountPatterns(field string, patterns []string) error {
	for index, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		if _, errMatch := path.Match(strings.ToLower(trimmed), ""); errMatch != nil {
			return fmt.Errorf("%s[%d] %q is not a valid pattern", field, index, trimmed)
		}
	}
	return nil
}

// accountSlot is one request an account is serving right now.
type accountSlot struct {
	AuthID  string
	Started time.Time
}

// effectiveAccountRulesLocked resolves the account rules in force and whether
// they are a runtime override.
func (s *store) effectiveAccountRulesLocked() ([]accountRule, bool) {
	if s.state.Settings != nil && s.state.Settings.AccountRules != nil {
		return *s.state.Settings.AccountRules, true
	}
	return s.cfg.AccountRules, false
}

// ruleFor returns the first rule matching an account ID.
func ruleFor(rules []accountRule, authID string) (accountRule, bool) {
	for _, rule := range rules {
		if _, ok := matchPattern([]string{rule.Match}, authID); ok {
			return rule, true
		}
	}
	return accountRule{}, false
}

// ownAccountsLocked resolves the account patterns bound to a key and whether
// the binding is strict: the runtime override when one is set, otherwise the
// configured values.
func (s *store) ownAccountsLocked(id string, entry *keyState) ([]string, bool) {
	configured, hasConfig := s.cfg.byID[id]
	var accounts []string
	strict := false
	if hasConfig {
		accounts = configured.Accounts
		strict = configured.StrictAccounts
	}
	if entry != nil && entry.Accounts != nil {
		accounts = *entry.Accounts
	}
	if entry != nil && entry.StrictAccounts != nil {
		strict = *entry.StrictAccounts
	}
	return accounts, strict
}

// releaseSignalLocked returns the channel closed on the next slot release.
func (s *store) releaseSignalLocked() <-chan struct{} {
	if s.released == nil {
		s.released = make(chan struct{})
	}
	return s.released
}

// notifyReleaseLocked wakes every request waiting for a free slot.
func (s *store) notifyReleaseLocked() {
	if s.released != nil {
		close(s.released)
	}
	s.released = make(chan struct{})
}

// pruneAccountsLocked drops expired reservations and stale in-flight slots.
func (s *store) pruneAccountsLocked(at time.Time) {
	for requestID, slot := range s.accountInflight {
		if at.Sub(slot.Started) > inflightTTL {
			delete(s.accountInflight, requestID)
		}
	}
	for authID, reservations := range s.accountPending {
		keep := reservations[:0]
		for _, reserved := range reservations {
			if at.Sub(reserved) <= accountReserveTTL {
				keep = append(keep, reserved)
			}
		}
		if len(keep) == 0 {
			delete(s.accountPending, authID)
		} else {
			s.accountPending[authID] = keep
		}
	}
}

// accountLoadLocked counts the requests each account is serving, including
// picks that have not been confirmed yet.
func (s *store) accountLoadLocked(at time.Time) map[string]int {
	s.pruneAccountsLocked(at)
	load := make(map[string]int, len(s.accountInflight)+len(s.accountPending))
	for _, slot := range s.accountInflight {
		load[slot.AuthID]++
	}
	for authID, reservations := range s.accountPending {
		load[authID] += len(reservations)
	}
	return load
}

// reserveAccountLocked holds a slot on an account until the request confirms it.
func (s *store) reserveAccountLocked(authID string, at time.Time) {
	if s.accountPending == nil {
		s.accountPending = map[string][]time.Time{}
	}
	s.accountPending[authID] = append(s.accountPending[authID], at)
}

// confirmAccount turns a reservation into an in-flight slot once the host
// reports which account a request runs on.
func (s *store) confirmAccount(requestID, authID string, at time.Time) {
	requestID = strings.TrimSpace(requestID)
	authID = strings.TrimSpace(authID)
	if requestID == "" || authID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if previous, known := s.accountInflight[requestID]; known && previous.AuthID == authID {
		return
	}
	if reservations := s.accountPending[authID]; len(reservations) > 0 {
		if len(reservations) == 1 {
			delete(s.accountPending, authID)
		} else {
			s.accountPending[authID] = reservations[1:]
		}
	}
	s.holdAccountLocked(requestID, authID, at)
}

// holdAccountLocked counts a request as served by an account. A retry of the
// same request on another account moves its slot there.
func (s *store) holdAccountLocked(slotID, authID string, at time.Time) {
	if s.accountInflight == nil {
		s.accountInflight = map[string]accountSlot{}
	}
	previous, known := s.accountInflight[slotID]
	s.accountInflight[slotID] = accountSlot{AuthID: authID, Started: at}
	if known && previous.AuthID != authID {
		s.notifyReleaseLocked()
	}
}

// pickPlan is what a scheduler pick needs to know about the calling key.
type pickPlan struct {
	id       string
	bindings []string
	strict   bool
	wait     time.Duration
	// attempt tags a request forwarded by the fallback executor, whose slot
	// is held directly instead of waiting for a host confirmation.
	attempt string
}

// planPick resolves the account binding and queue wait of the calling key.
func (s *store) planPick(headers map[string][]string, at time.Time) pickPlan {
	attempt := strings.TrimSpace(http.Header(headers).Get(attemptHeader))
	apiKey := apiKeyFromHeaders(http.Header(headers))
	if apiKey == "" {
		return pickPlan{attempt: attempt}
	}
	id := keyID(apiKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, configured := s.cfg.byID[id]; !configured && !s.cfg.TrackUnknownKeys {
		return pickPlan{attempt: attempt}
	}
	entry := s.entryLocked(id, "", false)
	bindings, strict := s.ownAccountsLocked(id, entry)
	plan := pickPlan{id: id, bindings: cleanPatterns(bindings), strict: strict, attempt: attempt}
	plan.wait = queueWait(s.effectiveRateLocked(id, entry, at))
	return plan
}

// pickOutcome classifies one attempt at choosing an account.
type pickOutcome int

const (
	pickDelegate pickOutcome = iota
	pickChosen
	pickBusy
	pickUnavailable
)

// chooseAccountLocked picks an account among the candidates for a key. Keys
// bound to accounts use them first; a strict binding stops there, otherwise
// the request spills over to the shared pool, which leaves out exclusive
// accounts. Within a group the highest priority account with a free slot wins,
// then the least loaded one, rotating among ties. When nothing restricts the
// choice, the host's own scheduler keeps deciding.
func (s *store) chooseAccountLocked(plan pickPlan, candidates []pluginapi.SchedulerAuthCandidate, at time.Time) (string, pickOutcome) {
	rules, _ := s.effectiveAccountRulesLocked()
	load := s.accountLoadLocked(at)

	restricted := len(plan.bindings) > 0
	var bound, pool, boundFree, poolFree []pluginapi.SchedulerAuthCandidate
	for _, candidate := range candidates {
		rule, _ := ruleFor(rules, candidate.ID)
		if rule.Exclusive || rule.Concurrency > 0 {
			restricted = true
		}
		free := rule.Concurrency <= 0 || int64(load[candidate.ID]) < rule.Concurrency
		_, isBound := matchPattern(plan.bindings, candidate.ID)
		switch {
		case isBound:
			bound = append(bound, candidate)
			if free {
				boundFree = append(boundFree, candidate)
			}
		case !rule.Exclusive:
			pool = append(pool, candidate)
			if free {
				poolFree = append(poolFree, candidate)
			}
		}
	}
	if !restricted {
		return "", pickDelegate
	}

	groups := [][]pluginapi.SchedulerAuthCandidate{poolFree}
	exists := len(pool) > 0
	if len(plan.bindings) > 0 {
		groups = [][]pluginapi.SchedulerAuthCandidate{boundFree}
		exists = len(bound) > 0
		if !plan.strict {
			groups = append(groups, poolFree)
			exists = exists || len(pool) > 0
		}
	}
	for _, group := range groups {
		if len(group) == 0 {
			continue
		}
		chosen := s.bestAccountLocked(group, load)
		if plan.attempt != "" {
			s.holdAccountLocked(attemptSlot(plan.attempt), chosen, at)
		} else {
			s.reserveAccountLocked(chosen, at)
		}
		return chosen, pickChosen
	}
	if exists {
		return "", pickBusy
	}
	return "", pickUnavailable
}

// bestAccountLocked prefers the highest priority, then the lightest load, and
// rotates among accounts that tie on both.
func (s *store) bestAccountLocked(group []pluginapi.SchedulerAuthCandidate, load map[string]int) string {
	sorted := append([]pluginapi.SchedulerAuthCandidate(nil), group...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		if load[sorted[i].ID] != load[sorted[j].ID] {
			return load[sorted[i].ID] < load[sorted[j].ID]
		}
		return sorted[i].ID < sorted[j].ID
	})
	ties := 1
	for ties < len(sorted) && sorted[ties].Priority == sorted[0].Priority && load[sorted[ties].ID] == load[sorted[0].ID] {
		ties++
	}
	chosen := sorted[s.pickCursor%uint64(ties)].ID
	s.pickCursor++
	return chosen
}

// schedulerPick answers scheduler.pick. When every account the key may use is
// busy, the pick waits for a slot for up to the key's queue time.
func schedulerPick(s *store, raw []byte) ([]byte, error) {
	var req pluginapi.SchedulerPickRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	plan := s.planPick(req.Options.Headers, s.now())
	deadline := time.Now().Add(plan.wait)
	for {
		s.mu.Lock()
		authID, outcome := s.chooseAccountLocked(plan, req.Candidates, s.now())
		signal := s.releaseSignalLocked()
		s.mu.Unlock()

		switch outcome {
		case pickChosen:
			return okEnvelope(pluginapi.SchedulerPickResponse{AuthID: authID, Handled: true})
		case pickDelegate:
			return okEnvelope(pluginapi.SchedulerPickResponse{Handled: false})
		case pickUnavailable:
			message := "no account available for this api key"
			if len(plan.bindings) > 0 {
				message = "no account bound to this api key is available for model " + req.Model
			}
			return statusErrorEnvelope(http.StatusServiceUnavailable, "account_unavailable", message), nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return statusErrorEnvelope(http.StatusTooManyRequests, "account_busy",
				"every account this api key may use is at its concurrency limit"), nil
		}
		if !waitForRelease(signal, remaining) {
			return statusErrorEnvelope(http.StatusTooManyRequests, "account_busy",
				"every account this api key may use is at its concurrency limit"), nil
		}
	}
}

// waitForRelease blocks until a slot is released or the wait runs out, and
// reports whether it was woken by a release.
func waitForRelease(signal <-chan struct{}, wait time.Duration) bool {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-signal:
		return true
	case <-timer.C:
		return false
	}
}

// statusErrorEnvelope builds a failed envelope that asks the host to answer
// the client with the given HTTP status.
func statusErrorEnvelope(status int, code, message string) []byte {
	raw, errMarshal := pluginabi.NewErrorEnvelope(code, message, status)
	if errMarshal != nil {
		return errorEnvelope(code, message)
	}
	return raw
}

// interceptAfterAuth records which account a request runs on so the account's
// concurrency can be enforced, and leaves the request unchanged.
func interceptAfterAuth(s *store, raw []byte) ([]byte, error) {
	var req pluginapi.RequestInterceptRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	if authID, ok := req.Metadata[selectedAuthKey].(string); ok {
		s.confirmAccount(req.RequestID, authID, s.now())
	}
	return okEnvelope(pluginapi.RequestInterceptResponse{Headers: req.Headers, Body: req.Body})
}

// accountView describes one host credential with its live load and the rule
// and keys that apply to it.
type accountView struct {
	ID          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Label       string   `json:"label,omitempty"`
	Email       string   `json:"email,omitempty"`
	Provider    string   `json:"provider,omitempty"`
	Status      string   `json:"status,omitempty"`
	Disabled    bool     `json:"disabled"`
	Unavailable bool     `json:"unavailable"`
	Priority    int      `json:"priority"`
	Concurrent  int      `json:"concurrent"`
	Limit       *int64   `json:"limit"`
	Exclusive   bool     `json:"exclusive"`
	Rule        string   `json:"rule"`
	BoundKeys   []string `json:"bound_keys"`
}

type accountsView struct {
	// Listed reports whether the host answered host.auth.list; when it did
	// not, Accounts only holds the accounts seen serving requests.
	Listed     bool          `json:"listed"`
	ListError  string        `json:"list_error,omitempty"`
	Rules      []accountRule `json:"rules"`
	Overridden bool          `json:"rules_overridden"`
	Accounts   []accountView `json:"accounts"`
}

// listHostAuths asks the host for its credentials; tests replace it.
var listHostAuths = func() ([]pluginapi.HostAuthFileEntry, error) {
	var resp struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if errCall := hostCall(pluginabi.MethodHostAuthList, map[string]any{}, &resp); errCall != nil {
		return nil, errCall
	}
	return resp.Files, nil
}

// accountsSnapshot renders every known account with its load, rule, and the
// keys bound to it.
func (s *store) accountsSnapshot(entries []pluginapi.HostAuthFileEntry, listErr error, at time.Time) accountsView {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules, overridden := s.effectiveAccountRulesLocked()
	view := accountsView{Listed: listErr == nil, Rules: append([]accountRule{}, rules...), Overridden: overridden, Accounts: []accountView{}}
	if listErr != nil {
		view.ListError = listErr.Error()
	}
	load := s.accountLoadLocked(at)

	type binding struct {
		name     string
		patterns []string
	}
	var bindings []binding
	for _, id := range s.sortedKeyIDs() {
		patterns, _ := s.ownAccountsLocked(id, s.state.Keys[id])
		if len(patterns) == 0 {
			continue
		}
		name := s.cfg.labelForID(id)
		if entry := s.state.Keys[id]; entry != nil {
			if entry.Label != "" {
				name = entry.Label
			} else if name == "" {
				name = entry.Hint
			}
		}
		if name == "" {
			name = id
		}
		bindings = append(bindings, binding{name: name, patterns: patterns})
	}

	seen := map[string]struct{}{}
	add := func(item accountView) {
		seen[item.ID] = struct{}{}
		item.Concurrent = load[item.ID]
		if rule, ok := ruleFor(rules, item.ID); ok {
			item.Rule = rule.Match
			item.Exclusive = rule.Exclusive
			if rule.Concurrency > 0 {
				value := rule.Concurrency
				item.Limit = &value
			}
		}
		item.BoundKeys = []string{}
		for _, bound := range bindings {
			if _, ok := matchPattern(bound.patterns, item.ID); ok {
				item.BoundKeys = append(item.BoundKeys, bound.name)
			}
		}
		view.Accounts = append(view.Accounts, item)
	}
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = strings.TrimSpace(entry.Name)
		}
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		provider := entry.Provider
		if provider == "" {
			provider = entry.Type
		}
		add(accountView{
			ID:          id,
			Name:        entry.Name,
			Label:       entry.Label,
			Email:       entry.Email,
			Provider:    provider,
			Status:      entry.Status,
			Disabled:    entry.Disabled,
			Unavailable: entry.Unavailable,
			Priority:    entry.Priority,
		})
	}
	busy := make([]string, 0, len(load))
	for id := range load {
		if _, ok := seen[id]; !ok {
			busy = append(busy, id)
		}
	}
	sort.Strings(busy)
	for _, id := range busy {
		add(accountView{ID: id})
	}
	return view
}

type accountRulesRequest struct {
	Rules []accountRule `json:"rules"`
	Clear bool          `json:"clear"`
}

// applyAccountRules stores account rules that replace the configured ones.
func (s *store) applyAccountRules(req accountRulesRequest) error {
	if errValidate := validateAccountRules("rules", req.Rules); errValidate != nil {
		return errValidate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Clear {
		s.settingsLocked().AccountRules = nil
	} else {
		rules := cleanAccountRules(req.Rules)
		if rules == nil {
			rules = []accountRule{}
		}
		s.settingsLocked().AccountRules = &rules
	}
	// A raised ceiling may let waiting requests through.
	s.notifyReleaseLocked()
	s.dirty = true
	s.persistLocked(true)
	return nil
}

func handleAccounts(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	if !strings.EqualFold(strings.TrimSpace(req.Method), http.MethodGet) {
		var body accountRulesRequest
		if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
			return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		}
		if errApply := s.applyAccountRules(body); errApply != nil {
			return jsonResponse(http.StatusBadRequest, map[string]string{"error": errApply.Error()})
		}
	}
	entries, errList := listHostAuths()
	return jsonResponse(http.StatusOK, s.accountsSnapshot(entries, errList, s.now()))
}
