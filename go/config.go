package main

import (
	"fmt"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// limits describes the ceiling applied to one accounting period.
// A value of zero inherits the default limit, a negative value means unlimited,
// and a positive value is the ceiling itself.
type limits struct {
	Tokens   int64   `yaml:"tokens" json:"tokens"`
	Requests int64   `yaml:"requests" json:"requests"`
	CostUSD  float64 `yaml:"cost_usd" json:"cost_usd"`
}

// merge overlays non-zero fields of other on top of the receiver.
func (l limits) merge(other limits) limits {
	if other.Tokens != 0 {
		l.Tokens = other.Tokens
	}
	if other.Requests != 0 {
		l.Requests = other.Requests
	}
	if other.CostUSD != 0 {
		l.CostUSD = other.CostUSD
	}
	return l
}

// limitSet groups the ceilings of every supported accounting period.
type limitSet struct {
	Daily   limits `yaml:"daily" json:"daily"`
	Monthly limits `yaml:"monthly" json:"monthly"`
	Total   limits `yaml:"total" json:"total"`
}

// merge overlays non-zero fields of other on top of the receiver.
func (l limitSet) merge(other limitSet) limitSet {
	return limitSet{
		Daily:   l.Daily.merge(other.Daily),
		Monthly: l.Monthly.merge(other.Monthly),
		Total:   l.Total.merge(other.Total),
	}
}

// keyConfig binds a configured client API key to a label, a free-form note, its
// own limits, the models it may not call, and its model mapping rules.
type keyConfig struct {
	Key           string         `yaml:"key"`
	Label         string         `yaml:"label"`
	Note          string         `yaml:"note"`
	Disabled      bool           `yaml:"disabled"`
	BlockedModels []string       `yaml:"blocked_models"`
	ModelMappings []modelMapping `yaml:"model_mappings"`
	// FallbackModels replaces the global fallback chain for this key when set.
	FallbackModels []string   `yaml:"fallback_models"`
	Limits         limitSet   `yaml:"limits"`
	RateLimits     rateLimits `yaml:"rate_limits"`
	// Schedule replaces default_schedule for this key when set.
	Schedule *schedule `yaml:"schedule"`
	// Accounts binds the key to host credentials, by ID or glob pattern; its
	// requests are scheduled on them first.
	Accounts []string `yaml:"accounts"`
	// StrictAccounts keeps the key on its bound accounts: when none of them
	// can serve a request, the request fails instead of using other accounts.
	StrictAccounts bool `yaml:"strict_accounts"`
}

// modelPrice describes USD prices per one million tokens for a model pattern.
type modelPrice struct {
	Match      string  `yaml:"match" json:"match,omitempty"`
	Input      float64 `yaml:"input" json:"input"`
	Output     float64 `yaml:"output" json:"output"`
	Reasoning  float64 `yaml:"reasoning" json:"reasoning"`
	CacheRead  float64 `yaml:"cache_read" json:"cache_read"`
	CacheWrite float64 `yaml:"cache_write" json:"cache_write"`
}

// pricingConfig holds the fallback price and the per-model price overrides.
type pricingConfig struct {
	Default modelPrice   `yaml:"default" json:"default"`
	Models  []modelPrice `yaml:"models" json:"models"`
}

// pluginConfig is the plugins.configs.apikey-quota document.
type pluginConfig struct {
	Enforce             bool           `yaml:"enforce"`
	StateFile           string         `yaml:"state_file"`
	PersistIntervalSecs int            `yaml:"persist_interval_seconds"`
	RejectStatus        int            `yaml:"reject_status"`
	TimeZone            string         `yaml:"time_zone"`
	CountFailedRequests bool           `yaml:"count_failed_requests"`
	TrackUnknownKeys    bool           `yaml:"track_unknown_keys"`
	BlockedModels       []string       `yaml:"blocked_models"`
	ModelMappings       []modelMapping `yaml:"model_mappings"`
	FallbackModels      []string       `yaml:"fallback_models"`
	FallbackStatusCodes []int          `yaml:"fallback_status_codes"`
	DefaultLimits       limitSet       `yaml:"default_limits"`
	DefaultRateLimits   rateLimits     `yaml:"default_rate_limits"`
	DefaultSchedule     *schedule      `yaml:"default_schedule"`
	Keys                []keyConfig    `yaml:"keys"`
	Pricing             pricingConfig  `yaml:"pricing"`
	// AccountRules caps the concurrency of host credentials and reserves
	// them for the keys bound to them.
	AccountRules []accountRule `yaml:"accounts"`

	location *time.Location
	byID     map[string]keyConfig
}

func defaultConfig() pluginConfig {
	return pluginConfig{
		Enforce:             true,
		StateFile:           "apikey-quota-state.json",
		PersistIntervalSecs: 5,
		RejectStatus:        429,
		TimeZone:            "Local",
		CountFailedRequests: false,
		TrackUnknownKeys:    true,
		location:            time.Local,
		byID:                map[string]keyConfig{},
	}
}

// parseConfig decodes the plugin configuration document and resolves derived state.
func parseConfig(raw []byte) (pluginConfig, error) {
	cfg := defaultConfig()
	if len(raw) > 0 {
		if errUnmarshal := yaml.Unmarshal(raw, &cfg); errUnmarshal != nil {
			return pluginConfig{}, fmt.Errorf("decoding apikey-quota config: %w", errUnmarshal)
		}
	}
	if cfg.PersistIntervalSecs < 0 {
		return pluginConfig{}, fmt.Errorf("persist_interval_seconds must not be negative")
	}
	if cfg.RejectStatus < 400 || cfg.RejectStatus > 599 {
		return pluginConfig{}, fmt.Errorf("reject_status must be a 4xx or 5xx status code")
	}
	if strings.TrimSpace(cfg.StateFile) == "" {
		return pluginConfig{}, fmt.Errorf("state_file must not be empty")
	}

	location, errLocation := loadLocation(cfg.TimeZone)
	if errLocation != nil {
		return pluginConfig{}, errLocation
	}
	cfg.location = location

	cfg.BlockedModels = cleanPatterns(cfg.BlockedModels)
	if errMappings := validateMappings("model_mappings", cfg.ModelMappings); errMappings != nil {
		return pluginConfig{}, errMappings
	}
	cfg.ModelMappings = cleanMappings(cfg.ModelMappings)
	if errFallbacks := validateFallbackModels("fallback_models", cfg.FallbackModels); errFallbacks != nil {
		return pluginConfig{}, errFallbacks
	}
	cfg.FallbackModels = cleanFallbackModels(cfg.FallbackModels)
	if errStatuses := validateFallbackStatuses(cfg.FallbackStatusCodes); errStatuses != nil {
		return pluginConfig{}, errStatuses
	}
	if errSchedule := validateSchedule("default_schedule", cfg.DefaultSchedule); errSchedule != nil {
		return pluginConfig{}, errSchedule
	}
	if errRules := validateAccountRules("accounts", cfg.AccountRules); errRules != nil {
		return pluginConfig{}, errRules
	}
	cfg.AccountRules = cleanAccountRules(cfg.AccountRules)

	cfg.byID = make(map[string]keyConfig, len(cfg.Keys))
	for index := range cfg.Keys {
		entry := cfg.Keys[index]
		entry.Key = strings.TrimSpace(entry.Key)
		if entry.Key == "" {
			return pluginConfig{}, fmt.Errorf("keys[%d].key must not be empty", index)
		}
		if strings.TrimSpace(entry.Label) == "" {
			entry.Label = maskKey(entry.Key)
		}
		entry.Note = strings.TrimSpace(entry.Note)
		entry.BlockedModels = cleanPatterns(entry.BlockedModels)
		if errMappings := validateMappings(fmt.Sprintf("keys[%d].model_mappings", index), entry.ModelMappings); errMappings != nil {
			return pluginConfig{}, errMappings
		}
		entry.ModelMappings = cleanMappings(entry.ModelMappings)
		if errFallbacks := validateFallbackModels(fmt.Sprintf("keys[%d].fallback_models", index), entry.FallbackModels); errFallbacks != nil {
			return pluginConfig{}, errFallbacks
		}
		entry.FallbackModels = cleanFallbackModels(entry.FallbackModels)
		if errSchedule := validateSchedule(fmt.Sprintf("keys[%d].schedule", index), entry.Schedule); errSchedule != nil {
			return pluginConfig{}, errSchedule
		}
		if errAccounts := validateAccountPatterns(fmt.Sprintf("keys[%d].accounts", index), entry.Accounts); errAccounts != nil {
			return pluginConfig{}, errAccounts
		}
		entry.Accounts = cleanPatterns(entry.Accounts)
		cfg.byID[keyID(entry.Key)] = entry
	}

	for index := range cfg.Pricing.Models {
		if strings.TrimSpace(cfg.Pricing.Models[index].Match) == "" {
			return pluginConfig{}, fmt.Errorf("pricing.models[%d].match must not be empty", index)
		}
	}
	return cfg, nil
}

func loadLocation(name string) (*time.Location, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || strings.EqualFold(trimmed, "local") {
		return time.Local, nil
	}
	if strings.EqualFold(trimmed, "utc") {
		return time.UTC, nil
	}
	location, errLoad := time.LoadLocation(trimmed)
	if errLoad != nil {
		return nil, fmt.Errorf("loading time_zone %q: %w", trimmed, errLoad)
	}
	return location, nil
}

// limitsForID resolves the configured limits and disabled flag for a key identifier.
func (c pluginConfig) limitsForID(id string) (limitSet, bool) {
	effective := c.DefaultLimits
	entry, ok := c.byID[id]
	if !ok {
		return effective, false
	}
	return effective.merge(entry.Limits), entry.Disabled
}

// cleanPatterns trims, drops empty entries, and de-duplicates a pattern list.
func cleanPatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		trimmed := strings.TrimSpace(pattern)
		if trimmed == "" {
			continue
		}
		lowered := strings.ToLower(trimmed)
		if _, ok := seen[lowered]; ok {
			continue
		}
		seen[lowered] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// matchPattern reports the first pattern matching the model, ignoring case.
// Patterns use path.Match glob syntax, and a pattern without a wildcard still
// matches the model it names exactly.
func matchPattern(patterns []string, model string) (string, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(model))
	if trimmed == "" {
		return "", false
	}
	for _, pattern := range patterns {
		lowered := strings.ToLower(strings.TrimSpace(pattern))
		if lowered == "" {
			continue
		}
		if matched, errMatch := path.Match(lowered, trimmed); errMatch == nil && matched {
			return pattern, true
		}
	}
	return "", false
}

// labelForID returns the configured label for a key identifier when one exists.
func (c pluginConfig) labelForID(id string) string {
	if entry, ok := c.byID[id]; ok {
		return entry.Label
	}
	return ""
}

// noteForID returns the configured note for a key identifier when one exists.
func (c pluginConfig) noteForID(id string) string {
	if entry, ok := c.byID[id]; ok {
		return entry.Note
	}
	return ""
}

// priceForModel selects the most specific price entry matching the model name.
// Later entries win on equal specificity, and the longest pattern wins overall.
// The second result names the matched pattern, or is empty when the default
// price applied.
func (p pricingConfig) priceForModel(model string) (modelPrice, string) {
	best := p.Default
	bestPattern := ""
	bestScore := -1
	trimmed := strings.ToLower(strings.TrimSpace(model))
	for _, entry := range p.Models {
		pattern := strings.TrimSpace(entry.Match)
		matched, errMatch := path.Match(strings.ToLower(pattern), trimmed)
		if errMatch != nil || !matched {
			continue
		}
		score := len(pattern)
		if strings.ContainsAny(pattern, "*?[") {
			score--
		}
		if score >= bestScore {
			best = entry
			bestPattern = pattern
			bestScore = score
		}
	}
	return best, bestPattern
}
