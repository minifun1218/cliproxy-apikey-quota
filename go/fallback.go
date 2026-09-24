package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// executorIdentifier names the provider the plugin executor registers as. No
// credential ever belongs to it; it only serves requests routed to it by
// model.route when a key has fallback models.
const executorIdentifier = "apikey-quota"

// fallbackHeader tells the client which fallback model answered a request.
const fallbackHeader = "X-Apikey-Quota-Fallback"

// executorFormats lists the protocols the executor accepts and emits. Each
// request is forwarded in the client's own protocol, so no translation happens
// on the way in or out. The native interactions protocol is left out because
// the host refuses to hand it to plugin executors.
var executorFormats = []string{"openai", "openai-response", "claude", "gemini", "gemini-cli", "codex", "antigravity"}

// defaultFallbackStatuses are the upstream statuses that move a request on to
// the next fallback model: timeouts, rate limits, overload, and server errors.
var defaultFallbackStatuses = []int{408, 429, 500, 502, 503, 504, 529}

// cleanFallbackModels trims and de-duplicates a fallback chain, keeping order.
func cleanFallbackModels(models []string) []string {
	return cleanPatterns(models)
}

// validateFallbackModels rejects fallback entries that are not concrete model names.
func validateFallbackModels(field string, models []string) error {
	for index, model := range models {
		if strings.ContainsAny(model, "*?[") {
			return fmt.Errorf("%s[%d] must be a concrete model name", field, index)
		}
	}
	return nil
}

// validateFallbackStatuses rejects statuses that are not HTTP error codes.
func validateFallbackStatuses(statuses []int) error {
	for index, status := range statuses {
		if status < 400 || status > 599 {
			return fmt.Errorf("fallback_status_codes[%d] must be a 4xx or 5xx status code", index)
		}
	}
	return nil
}

// effectiveGlobalFallbacksLocked resolves the fallback chain of keys without
// their own, and whether it is a runtime override.
func (s *store) effectiveGlobalFallbacksLocked() ([]string, bool) {
	if s.state.Settings != nil && s.state.Settings.FallbackModels != nil {
		return *s.state.Settings.FallbackModels, true
	}
	return s.cfg.FallbackModels, false
}

// ownFallbacksLocked resolves a key's own chain: the runtime override when one
// is set, including an empty one that turns fallback off for the key, otherwise
// the configured chain. The second result reports whether the key has one.
func (s *store) ownFallbacksLocked(id string, entry *keyState) ([]string, bool) {
	if entry != nil && entry.FallbackModels != nil {
		return *entry.FallbackModels, true
	}
	if configured, ok := s.cfg.byID[id]; ok && configured.FallbackModels != nil {
		return configured.FallbackModels, true
	}
	return nil, false
}

// fallbacksLocked resolves the chain in force for a key: its own chain replaces
// the global one rather than extending it.
func (s *store) fallbacksLocked(id string, entry *keyState) []string {
	if own, ok := s.ownFallbacksLocked(id, entry); ok {
		return own
	}
	global, _ := s.effectiveGlobalFallbacksLocked()
	return global
}

// fallbackStatusSet returns the statuses that trigger the next fallback.
func (c pluginConfig) fallbackStatusSet() map[int]struct{} {
	statuses := c.FallbackStatusCodes
	if len(statuses) == 0 {
		statuses = defaultFallbackStatuses
	}
	set := make(map[int]struct{}, len(statuses))
	for _, status := range statuses {
		set[status] = struct{}{}
	}
	return set
}

// routePlan is the outcome of resolving one request: the model and provider it
// runs on first, and the models tried in order when that attempt fails.
type routePlan struct {
	id        string
	tracked   bool
	mapping   *modelMapping
	model     string
	provider  string
	fallbacks []string
}

// planRoute resolves the mapping and fallback chain for a request without
// touching any counter.
func (s *store) planRoute(headers http.Header, requested string, available []string) (routePlan, bool) {
	base, suffix := splitThinkingSuffix(requested)
	if base == "" {
		return routePlan{}, false
	}

	s.mu.Lock()
	plan := routePlan{model: requested}
	if apiKey := apiKeyFromHeaders(headers); apiKey != "" {
		plan.id = keyID(apiKey)
		_, configured := s.cfg.byID[plan.id]
		plan.tracked = configured || s.cfg.TrackUnknownKeys
	}
	var entry *keyState
	if plan.tracked {
		entry = s.entryLocked(plan.id, "", false)
	}
	var fallbacks []string
	var mappings []modelMapping
	if plan.tracked {
		mappings = s.mappingsLocked(plan.id, entry)
		fallbacks = s.fallbacksLocked(plan.id, entry)
	} else {
		mappings, _ = s.effectiveGlobalMappingsLocked()
		fallbacks, _ = s.effectiveGlobalFallbacksLocked()
	}
	s.mu.Unlock()

	if mapping, found := matchMapping(mappings, base); found {
		target := withSuffix(mapping.To, suffix)
		provider := mapping.Provider
		if provider == "" {
			targetBase, _ := splitThinkingSuffix(target)
			inferred, ok := inferProvider(targetBase, available)
			if !ok {
				hostLogf("apikey-quota: mapping %s -> %s names no provider and none could be inferred; set provider on the rule", mapping.From, mapping.To)
			} else {
				provider = inferred
			}
		}
		if provider != "" {
			matched := mapping
			plan.mapping = &matched
			plan.model = target
			plan.provider = provider
		}
	}

	primaryBase, _ := splitThinkingSuffix(plan.model)
	for _, model := range fallbacks {
		if strings.EqualFold(model, primaryBase) {
			continue
		}
		plan.fallbacks = append(plan.fallbacks, withSuffix(model, suffix))
	}
	return plan, true
}

// withSuffix carries a thinking suffix over to a model that has none of its own.
func withSuffix(model, suffix string) string {
	if suffix == "" {
		return model
	}
	if _, own := splitThinkingSuffix(model); own != "" {
		return model
	}
	return model + suffix
}

// fallbackEligible reports whether a request may be handed to the plugin
// executor. Token counting has no host callback to forward to, and the native
// interactions protocol never reaches plugin executors.
func fallbackEligible(req pluginapi.ModelRouteRequest) bool {
	if !hostAvailable() {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(req.SourceFormat), "interactions") {
		return false
	}
	if requestPath, ok := req.Metadata["request_path"].(string); ok {
		lowered := strings.ToLower(requestPath)
		if strings.Contains(lowered, "count_tokens") || strings.Contains(lowered, "counttokens") {
			return false
		}
	}
	return true
}

// countFallback records that a request of the key was answered by a fallback model.
func (s *store) countFallback(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entryLocked(id, "", true)
	entry.Fallbacks++
	s.dirty = true
	s.persistLocked(false)
}

type fallbacksRequest struct {
	FallbackModels []string `json:"fallback_models"`
	Clear          bool     `json:"clear"`
}

// applyGlobalFallbacks stores the fallback chain of keys without their own.
func (s *store) applyGlobalFallbacks(req fallbacksRequest) error {
	if !req.Clear {
		if errValidate := validateFallbackModels("fallback_models", req.FallbackModels); errValidate != nil {
			return errValidate
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Clear {
		s.settingsLocked().FallbackModels = nil
	} else {
		models := cleanFallbackModels(req.FallbackModels)
		if models == nil {
			models = []string{}
		}
		s.settingsLocked().FallbackModels = &models
	}
	s.dirty = true
	s.persistLocked(true)
	return nil
}

func handleFallbacks(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body fallbacksRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if errApply := s.applyGlobalFallbacks(body); errApply != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": errApply.Error()})
	}
	return jsonResponse(http.StatusOK, map[string]any{"ok": true})
}

// hostCall sends an RPC to the host; tests replace it with a fake host.
var hostCall = callHost

// executorRequest is the executor.execute and executor.execute_stream payload.
type executorRequest struct {
	pluginapi.ExecutorRequest
	StreamID       string `json:"stream_id,omitempty"`
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type hostModelRequest struct {
	pluginapi.HostModelExecutionRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type executorStreamResult struct {
	Headers http.Header `json:"headers,omitempty"`
}

type streamEmitRequest struct {
	StreamID string `json:"stream_id"`
	Payload  []byte `json:"payload,omitempty"`
	Error    string `json:"error,omitempty"`
}

type streamCloseRequest struct {
	StreamID string `json:"stream_id"`
	Error    string `json:"error,omitempty"`
}

// attempt is one model the executor tries, with the provider it is pinned to
// when a mapping rule named or implied one.
type attempt struct {
	model    string
	provider string
	fallback bool
}

// executorPlan recomputes the attempts for a request routed to the executor.
// It runs the same resolution as model.route, so both always agree.
func executorPlan(s *store, req pluginapi.ExecutorRequest) (routePlan, []attempt) {
	requested := req.Model
	if value, ok := req.Metadata["requested_model"].(string); ok && strings.TrimSpace(value) != "" {
		requested = value
	}
	s.mu.Lock()
	available := append([]string(nil), s.providers...)
	s.mu.Unlock()
	plan, ok := s.planRoute(req.Headers, requested, available)
	if !ok {
		return routePlan{}, []attempt{{model: requested}}
	}
	attempts := []attempt{{model: plan.model, provider: plan.provider}}
	for _, model := range plan.fallbacks {
		attempts = append(attempts, attempt{model: model, fallback: true})
	}
	return plan, attempts
}

// hostRequest builds the forwarded request, tagged so the scheduler can hold
// the account slot of this attempt until the executor releases it.
func (a attempt) hostRequest(req executorRequest, stream bool, attemptID string) hostModelRequest {
	headers := req.Headers.Clone()
	if headers == nil {
		headers = http.Header{}
	}
	headers.Set(attemptHeader, attemptID)
	return hostModelRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol:  req.SourceFormat,
			ExitProtocol:   req.Format,
			Model:          a.model,
			Stream:         stream,
			Body:           req.Payload,
			Headers:        headers,
			Query:          cloneQuery(req.Query),
			Alt:            req.Alt,
			ForcedProvider: a.provider,
		},
		HostCallbackID: req.HostCallbackID,
	}
}

func cloneQuery(query url.Values) url.Values {
	if query == nil {
		return nil
	}
	out := make(url.Values, len(query))
	for key, values := range query {
		out[key] = append([]string(nil), values...)
	}
	return out
}

// shouldFallBack reports whether a failed attempt moves on to the next model.
// A failure without a status never reached an upstream answer, for example
// because no credential serves the model, so it falls back too.
func shouldFallBack(errCall error, statuses map[int]struct{}) bool {
	var failure *hostError
	if !errors.As(errCall, &failure) {
		return false
	}
	if failure.Status == 0 {
		return true
	}
	_, ok := statuses[failure.Status]
	return ok
}

// executorFailure turns the last failed attempt into the executor's error
// envelope, keeping the upstream status so the client sees it unchanged.
func executorFailure(errCall error) ([]byte, error) {
	status := http.StatusBadGateway
	code := "upstream_failed"
	message := errCall.Error()
	var failure *hostError
	if errors.As(errCall, &failure) {
		message = failure.Message
		if failure.Status > 0 {
			status = failure.Status
		}
		if failure.Code != "" {
			code = failure.Code
		}
	}
	return pluginabi.NewErrorEnvelope(code, message, status)
}

// runAttempts tries each attempt in order until one succeeds or fails with a
// status that does not trigger a fallback.
func runAttempts(s *store, plan routePlan, attempts []attempt, try func(attempt) error) (attempt, error) {
	s.mu.Lock()
	statuses := s.cfg.fallbackStatusSet()
	s.mu.Unlock()
	var lastErr error
	for index, current := range attempts {
		errTry := try(current)
		if errTry == nil {
			if current.fallback {
				hostLogf("apikey-quota: key %s served by fallback model %s after %d failed attempt(s)", plan.id, current.model, index)
				if plan.tracked {
					s.countFallback(plan.id)
				}
			}
			return current, nil
		}
		lastErr = errTry
		if index == len(attempts)-1 || !shouldFallBack(errTry, statuses) {
			break
		}
		hostLogf("apikey-quota: model %s failed for key %s, trying %s: %v", current.model, plan.id, attempts[index+1].model, errTry)
	}
	return attempt{}, lastErr
}

// executeWithFallback answers executor.execute.
func executeWithFallback(s *store, raw []byte) ([]byte, error) {
	var req executorRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	plan, attempts := executorPlan(s, req.ExecutorRequest)
	var resp pluginapi.HostModelExecutionResponse
	served, errRun := runAttempts(s, plan, attempts, func(current attempt) error {
		resp = pluginapi.HostModelExecutionResponse{}
		attemptID := newAttemptID()
		defer s.endRequest(attemptSlot(attemptID))
		return hostCall(pluginabi.MethodHostModelExecute, current.hostRequest(req, false, attemptID), &resp)
	})
	if errRun != nil {
		return executorFailure(errRun)
	}
	headers := resp.Headers
	if served.fallback {
		headers = markFallback(headers, served.model)
	}
	return okEnvelope(pluginapi.ExecutorResponse{Payload: resp.Body, Headers: headers})
}

// executeStreamWithFallback answers executor.execute_stream. The host only
// returns a stream once its first chunk arrived, so an upstream that fails
// before sending anything still falls back; a stream that breaks midway is
// passed on as it is, since part of it already reached the client.
func executeStreamWithFallback(s *store, raw []byte) ([]byte, error) {
	var req executorRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	if req.StreamID == "" {
		return pluginabi.NewErrorEnvelope("invalid_request", "executor stream request has no stream_id", http.StatusInternalServerError)
	}
	plan, attempts := executorPlan(s, req.ExecutorRequest)
	var stream pluginapi.HostModelStreamResponse
	var slot string
	served, errRun := runAttempts(s, plan, attempts, func(current attempt) error {
		stream = pluginapi.HostModelStreamResponse{}
		attemptID := newAttemptID()
		errCall := hostCall(pluginabi.MethodHostModelExecuteStream, current.hostRequest(req, true, attemptID), &stream)
		if errCall != nil {
			s.endRequest(attemptSlot(attemptID))
			return errCall
		}
		slot = attemptSlot(attemptID)
		return nil
	})
	if errRun != nil {
		return executorFailure(errRun)
	}
	headers := stream.Headers
	if served.fallback {
		headers = markFallback(headers, served.model)
	}
	// The account keeps serving the request until the stream ends.
	go func() {
		defer s.endRequest(slot)
		pumpStream(stream.StreamID, req.StreamID)
	}()
	return okEnvelope(executorStreamResult{Headers: headers})
}

// pumpStream copies the host model stream into the executor stream until
// either side ends.
func pumpStream(source, target string) {
	closeTarget := func(message string) {
		if errClose := hostCall(pluginabi.MethodHostStreamClose, streamCloseRequest{StreamID: target, Error: message}, nil); errClose != nil {
			hostLogf("apikey-quota: closing fallback stream: %v", errClose)
		}
	}
	defer func() {
		_ = hostCall(pluginabi.MethodHostModelStreamClose, pluginapi.HostModelStreamCloseRequest{StreamID: source}, nil)
	}()
	for {
		var chunk pluginapi.HostModelStreamReadResponse
		if errRead := hostCall(pluginabi.MethodHostModelStreamRead, pluginapi.HostModelStreamReadRequest{StreamID: source}, &chunk); errRead != nil {
			closeTarget(errRead.Error())
			return
		}
		if len(chunk.Payload) > 0 {
			if errEmit := hostCall(pluginabi.MethodHostStreamEmit, streamEmitRequest{StreamID: target, Payload: chunk.Payload}, nil); errEmit != nil {
				// The client went away; nothing is left to deliver.
				closeTarget("")
				return
			}
		}
		if chunk.Error != "" {
			closeTarget(chunk.Error)
			return
		}
		if chunk.Done {
			closeTarget("")
			return
		}
	}
}

func markFallback(headers http.Header, model string) http.Header {
	out := headers.Clone()
	if out == nil {
		out = http.Header{}
	}
	out.Set(fallbackHeader, model)
	return out
}

// countTokensUnsupported answers executor.count_tokens. Token counts are never
// routed here on purpose, so reaching this means the route could not tell a
// count request apart, which only Gemini countTokens does.
func countTokensUnsupported() ([]byte, error) {
	return pluginabi.NewErrorEnvelope("unsupported", "apikey-quota cannot count tokens for keys with fallback models", http.StatusNotImplemented)
}
