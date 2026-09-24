package main

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// modelMapping rewrites a client-requested model to another model, optionally
// on a named built-in provider. From is a glob matched case-insensitively
// against the requested model with any thinking suffix removed.
type modelMapping struct {
	From     string `yaml:"from" json:"from"`
	To       string `yaml:"to" json:"to"`
	Provider string `yaml:"provider" json:"provider"`
}

// cleanMappings trims every rule, drops incomplete ones, and keeps only the
// first rule for each source pattern so rule order stays meaningful.
func cleanMappings(mappings []modelMapping) []modelMapping {
	if len(mappings) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(mappings))
	out := make([]modelMapping, 0, len(mappings))
	for _, mapping := range mappings {
		mapping.From = strings.TrimSpace(mapping.From)
		mapping.To = strings.TrimSpace(mapping.To)
		mapping.Provider = strings.ToLower(strings.TrimSpace(mapping.Provider))
		if mapping.From == "" || mapping.To == "" {
			continue
		}
		lowered := strings.ToLower(mapping.From)
		if _, ok := seen[lowered]; ok {
			continue
		}
		seen[lowered] = struct{}{}
		out = append(out, mapping)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// validateMappings rejects rules that cannot be applied, naming the offender.
func validateMappings(field string, mappings []modelMapping) error {
	for index, mapping := range mappings {
		from := strings.TrimSpace(mapping.From)
		to := strings.TrimSpace(mapping.To)
		if from == "" && to == "" && strings.TrimSpace(mapping.Provider) == "" {
			continue
		}
		if from == "" {
			return fmt.Errorf("%s[%d].from must not be empty", field, index)
		}
		if to == "" {
			return fmt.Errorf("%s[%d].to must not be empty", field, index)
		}
		if _, errMatch := path.Match(strings.ToLower(from), ""); errMatch != nil {
			return fmt.Errorf("%s[%d].from %q is not a valid pattern", field, index, from)
		}
		if strings.ContainsAny(to, "*?[") {
			return fmt.Errorf("%s[%d].to must be a concrete model name", field, index)
		}
	}
	return nil
}

// splitThinkingSuffix separates a trailing "(...)" thinking suffix, such as
// "claude-sonnet-4-5(high)", from the base model name.
func splitThinkingSuffix(model string) (string, string) {
	trimmed := strings.TrimSpace(model)
	if !strings.HasSuffix(trimmed, ")") {
		return trimmed, ""
	}
	open := strings.LastIndex(trimmed, "(")
	if open <= 0 {
		return trimmed, ""
	}
	return strings.TrimSpace(trimmed[:open]), trimmed[open:]
}

// matchMapping returns the first rule whose pattern matches the model.
func matchMapping(mappings []modelMapping, model string) (modelMapping, bool) {
	lowered := strings.ToLower(strings.TrimSpace(model))
	if lowered == "" {
		return modelMapping{}, false
	}
	for _, mapping := range mappings {
		pattern := strings.ToLower(strings.TrimSpace(mapping.From))
		if pattern == "" {
			continue
		}
		if matched, errMatch := path.Match(pattern, lowered); errMatch == nil && matched {
			return mapping, true
		}
	}
	return modelMapping{}, false
}

// providerHints lists the built-in providers that usually serve a model family,
// in preference order. It is only consulted when a rule names no provider.
var providerHints = []struct {
	prefixes  []string
	providers []string
}{
	{[]string{"claude"}, []string{"claude", "antigravity"}},
	{[]string{"gpt", "o1", "o3", "o4", "codex", "chatgpt"}, []string{"codex"}},
	{[]string{"gemini"}, []string{"gemini-cli", "gemini", "aistudio", "vertex", "antigravity"}},
	{[]string{"grok"}, []string{"xai"}},
	{[]string{"kimi"}, []string{"kimi"}},
}

// inferProvider picks the built-in provider that should serve the target model.
// When the host reports a single provider with auth registered, that provider
// is the only possible answer.
func inferProvider(model string, available []string) (string, bool) {
	availableSet := make(map[string]struct{}, len(available))
	for _, provider := range available {
		availableSet[strings.ToLower(strings.TrimSpace(provider))] = struct{}{}
	}
	lowered := strings.ToLower(model)
	for _, hint := range providerHints {
		for _, prefix := range hint.prefixes {
			if !strings.HasPrefix(lowered, prefix) {
				continue
			}
			for _, provider := range hint.providers {
				if _, ok := availableSet[provider]; ok {
					return provider, true
				}
			}
		}
	}
	if len(availableSet) == 1 {
		for provider := range availableSet {
			return provider, true
		}
	}
	return "", false
}

// effectiveGlobalMappingsLocked resolves the mapping rules applied to every key.
func (s *store) effectiveGlobalMappingsLocked() ([]modelMapping, bool) {
	if s.state.Settings != nil && s.state.Settings.ModelMappings != nil {
		return *s.state.Settings.ModelMappings, true
	}
	return s.cfg.ModelMappings, false
}

// ownMappingsLocked resolves a key's own rules: the runtime override when one
// is set, otherwise the configured rules.
func (s *store) ownMappingsLocked(id string, entry *keyState) []modelMapping {
	if entry != nil && entry.ModelMappings != nil {
		return *entry.ModelMappings
	}
	if configured, ok := s.cfg.byID[id]; ok {
		return configured.ModelMappings
	}
	return nil
}

// mappingsLocked resolves the rules for one key in evaluation order: the key's
// own rules first, so a key can override a global rule, then the global rules.
func (s *store) mappingsLocked(id string, entry *keyState) []modelMapping {
	global, _ := s.effectiveGlobalMappingsLocked()
	own := s.ownMappingsLocked(id, entry)
	if len(own) == 0 {
		return global
	}
	return append(append([]modelMapping(nil), own...), global...)
}

// rememberProviders keeps the provider keys the host last offered, so the
// dashboard can suggest valid provider names.
func (s *store) rememberProvidersLocked(available []string) {
	if len(available) == 0 {
		return
	}
	providers := make([]string, 0, len(available))
	for _, provider := range available {
		if trimmed := strings.ToLower(strings.TrimSpace(provider)); trimmed != "" {
			providers = append(providers, trimmed)
		}
	}
	sort.Strings(providers)
	s.providers = providers
}

// routeModel answers model.route. A key with fallback models is routed to the
// plugin's own executor, which runs the mapped or requested model first and
// moves down the fallback chain when it fails. Otherwise the first mapping rule
// matching the requested model sends the request to the target provider, and
// every unmatched request is left to the host's normal model resolution.
func routeModel(s *store, raw []byte) ([]byte, error) {
	var req pluginapi.ModelRouteRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}
	unhandled := pluginapi.ModelRouteResponse{}

	s.mu.Lock()
	s.rememberProvidersLocked(req.AvailableProviders)
	s.mu.Unlock()

	plan, ok := s.planRoute(req.Headers, req.RequestedModel, req.AvailableProviders)
	if !ok {
		return okEnvelope(unhandled)
	}
	if plan.mapping != nil && plan.tracked {
		s.countMapped(plan.id)
	}

	if len(plan.fallbacks) > 0 && fallbackEligible(req) {
		return okEnvelope(pluginapi.ModelRouteResponse{
			Handled:    true,
			TargetKind: pluginapi.ModelRouteTargetSelf,
			Reason:     fmt.Sprintf("apikey-quota fallback %s -> %s", plan.model, strings.Join(plan.fallbacks, ", ")),
		})
	}

	if plan.mapping == nil {
		return okEnvelope(unhandled)
	}
	return okEnvelope(pluginapi.ModelRouteResponse{
		Handled:     true,
		TargetKind:  pluginapi.ModelRouteTargetProvider,
		Target:      plan.provider,
		TargetModel: plan.model,
		Reason:      fmt.Sprintf("apikey-quota mapping %s -> %s", plan.mapping.From, plan.mapping.To),
	})
}

// countMapped records that a request of the key was rewritten by a mapping.
func (s *store) countMapped(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entryLocked(id, "", true)
	entry.Mapped++
	s.dirty = true
	s.persistLocked(false)
}

type mappingsRequest struct {
	ModelMappings []modelMapping `json:"model_mappings"`
	Clear         bool           `json:"clear"`
}

// applyGlobalMappings stores the mapping rules applied to every API key.
func (s *store) applyGlobalMappings(req mappingsRequest) error {
	if !req.Clear {
		if errValidate := validateMappings("model_mappings", req.ModelMappings); errValidate != nil {
			return errValidate
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Clear {
		s.settingsLocked().ModelMappings = nil
	} else {
		mappings := cleanMappings(req.ModelMappings)
		if mappings == nil {
			mappings = []modelMapping{}
		}
		s.settingsLocked().ModelMappings = &mappings
	}
	s.dirty = true
	s.persistLocked(true)
	return nil
}
