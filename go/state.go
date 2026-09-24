package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	stateVersion    = 1
	periodDaily     = "daily"
	periodMonthly   = "monthly"
	periodTotal     = "total"
	dailyLayout     = "2006-01-02"
	monthlyLayout   = "2006-01"
	totalPeriodName = "all-time"
)

// counters accumulates the three billable dimensions.
type counters struct {
	Tokens   int64   `json:"tokens"`
	Requests int64   `json:"requests"`
	CostUSD  float64 `json:"cost_usd"`
}

func (c *counters) add(delta counters) {
	c.Tokens += delta.Tokens
	c.Requests += delta.Requests
	c.CostUSD += delta.CostUSD
}

// bucket is the counter set of one accounting period plus the period it covers.
type bucket struct {
	Period   string   `json:"period"`
	Counters counters `json:"counters"`
}

// roll resets the bucket when it no longer covers the requested period.
func (b *bucket) roll(period string) {
	if b.Period == period {
		return
	}
	b.Period = period
	b.Counters = counters{}
}

// keyState is the persisted accounting record of one client API key.
type keyState struct {
	Hint       string              `json:"hint,omitempty"`
	Label      string              `json:"label,omitempty"`
	Daily      bucket              `json:"daily"`
	Monthly    bucket              `json:"monthly"`
	Total      bucket              `json:"total"`
	Models     map[string]counters `json:"models,omitempty"`
	Blocked    int64               `json:"blocked,omitempty"`
	Mapped     int64               `json:"mapped,omitempty"`
	Fallbacks  int64               `json:"fallbacks,omitempty"`
	LastUsedAt string              `json:"last_used_at,omitempty"`
	Overrides  *limitSet           `json:"overrides,omitempty"`
	Disabled   *bool               `json:"disabled,omitempty"`
	// BlockedModels replaces the configured per-key deny list when non-nil.
	BlockedModels *[]string `json:"blocked_models,omitempty"`
	// ModelMappings replaces the configured per-key mapping rules when non-nil.
	ModelMappings *[]modelMapping `json:"model_mappings,omitempty"`
	// FallbackModels replaces the configured fallback chain when non-nil; an
	// empty list turns fallback off for the key.
	FallbackModels *[]string `json:"fallback_models,omitempty"`
	// Note replaces the configured note when non-nil, including with an empty
	// string, which is how an operator clears a note set in the configuration.
	Note *string `json:"note,omitempty"`
	// RateOverrides overlays the configured rate limits when non-nil.
	RateOverrides *rateLimits `json:"rate_limits,omitempty"`
	// Schedule replaces the configured schedule when non-nil.
	Schedule *schedule `json:"schedule,omitempty"`
	// Windows holds the usage of the current occurrence of each schedule
	// window, keyed by window name.
	Windows map[string]bucket `json:"windows,omitempty"`
	// Accounts replaces the configured account binding when non-nil; an
	// empty list unbinds the key.
	Accounts *[]string `json:"accounts,omitempty"`
	// StrictAccounts replaces the configured strict flag when non-nil.
	StrictAccounts *bool `json:"strict_accounts,omitempty"`
}

// stateSettings holds management overrides that replace configured values.
// A nil field means the configured value applies.
type stateSettings struct {
	BlockedModels  *[]string       `json:"blocked_models,omitempty"`
	ModelMappings  *[]modelMapping `json:"model_mappings,omitempty"`
	FallbackModels *[]string       `json:"fallback_models,omitempty"`
	Pricing        *pricingConfig  `json:"pricing,omitempty"`
	AccountRules   *[]accountRule  `json:"account_rules,omitempty"`
}

// stateFile is the on-disk document holding every tracked key.
type stateFile struct {
	Version  int                  `json:"version"`
	Keys     map[string]*keyState `json:"keys"`
	Settings *stateSettings       `json:"settings,omitempty"`
}

// store owns the accounting state and serializes access to it.
type store struct {
	mu       sync.Mutex
	cfg      pluginConfig
	state    stateFile
	now      func() time.Time
	path     string
	dirty    bool
	lastSave time.Time
	// inflight maps a host request ID to the admitted request it belongs to.
	// It lives in memory only: concurrency is meaningless after a restart.
	inflight map[string]inflightRequest
	// providers holds the built-in provider keys the host last offered to
	// model.route. It lives in memory only and feeds dashboard suggestions.
	providers []string
	// rates holds the last minute of activity per key for the rate limits.
	rates map[string]*rateTrack
	// accountInflight maps a host request ID to the account serving it, and
	// accountPending holds the picks the host has not confirmed yet. Both
	// live in memory only, like inflight.
	accountInflight map[string]accountSlot
	accountPending  map[string][]time.Time
	// pickCursor rotates picks among equally suitable accounts.
	pickCursor uint64
	// released is closed and replaced whenever a concurrency slot frees up,
	// waking the requests queued for one.
	released chan struct{}
}

func newStore(now func() time.Time) *store {
	if now == nil {
		now = time.Now
	}
	return &store{
		cfg:   defaultConfig(),
		state: stateFile{Version: stateVersion, Keys: map[string]*keyState{}},
		now:   now,
	}
}

// keyID derives a stable, non-reversible identifier for an API key so the raw
// secret never reaches the state file, the management API, or the dashboard.
func keyID(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])[:16]
}

// maskKey renders a display-safe hint for an API key.
func maskKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 8 {
		return strings.Repeat("*", len(trimmed))
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}

// configure applies a new configuration and reloads persisted state when the
// state file path changed.
func (s *store) configure(cfg pluginConfig) error {
	resolved, errResolve := filepath.Abs(cfg.StateFile)
	if errResolve != nil {
		return fmt.Errorf("resolving state_file: %w", errResolve)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	reload := s.path != resolved
	s.cfg = cfg
	s.path = resolved
	if !reload {
		return nil
	}
	loaded, errLoad := loadStateFile(resolved)
	if errLoad != nil {
		return errLoad
	}
	s.state = loaded
	s.dirty = false
	return nil
}

func loadStateFile(path string) (stateFile, error) {
	empty := stateFile{Version: stateVersion, Keys: map[string]*keyState{}}
	raw, errRead := os.ReadFile(path)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			return empty, nil
		}
		return empty, fmt.Errorf("reading state file %s: %w", path, errRead)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return empty, nil
	}
	var decoded stateFile
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		return empty, fmt.Errorf("decoding state file %s: %w", path, errUnmarshal)
	}
	if decoded.Keys == nil {
		decoded.Keys = map[string]*keyState{}
	}
	decoded.Version = stateVersion
	return decoded, nil
}

// periodsAt returns the daily and monthly period identifiers for the instant.
func (s *store) periodsAt(at time.Time) (string, string) {
	local := at.In(s.cfg.location)
	return local.Format(dailyLayout), local.Format(monthlyLayout)
}

// entryLocked returns the state of a key, creating it when requested.
func (s *store) entryLocked(id, hint string, create bool) *keyState {
	entry, ok := s.state.Keys[id]
	if !ok {
		if !create {
			return nil
		}
		entry = &keyState{}
		s.state.Keys[id] = entry
		s.dirty = true
	}
	if entry.Hint == "" && hint != "" {
		entry.Hint = hint
		s.dirty = true
	}
	if label := s.cfg.labelForID(id); label != "" && entry.Label != label {
		entry.Label = label
		s.dirty = true
	}
	return entry
}

// rollLocked advances the period buckets of an entry to the given instant.
func (s *store) rollLocked(entry *keyState, at time.Time) {
	daily, monthly := s.periodsAt(at)
	if entry.Daily.Period != daily || entry.Monthly.Period != monthly {
		s.dirty = true
	}
	entry.Daily.roll(daily)
	entry.Monthly.roll(monthly)
	if entry.Total.Period == "" {
		entry.Total.Period = totalPeriodName
	}
}

// settingsLocked returns the mutable settings block, creating it on demand.
func (s *store) settingsLocked() *stateSettings {
	if s.state.Settings == nil {
		s.state.Settings = &stateSettings{}
	}
	return s.state.Settings
}

// pricingSource names where the price table in force came from.
const (
	pricingSourceRuntime = "runtime"
	pricingSourceConfig  = "config"
	pricingSourceBuiltin = "builtin"
)

// effectivePricingLocked resolves the price table in force right now. A runtime
// override wins, then the configured table, then the built-in defaults.
func (s *store) effectivePricingLocked() (pricingConfig, string) {
	if s.state.Settings != nil && s.state.Settings.Pricing != nil {
		return *s.state.Settings.Pricing, pricingSourceRuntime
	}
	if !s.cfg.Pricing.isEmpty() {
		return s.cfg.Pricing, pricingSourceConfig
	}
	return builtinPricing(), pricingSourceBuiltin
}

// effectiveGlobalBlockedLocked resolves the deny list applied to every key.
func (s *store) effectiveGlobalBlockedLocked() ([]string, bool) {
	if s.state.Settings != nil && s.state.Settings.BlockedModels != nil {
		return *s.state.Settings.BlockedModels, true
	}
	return s.cfg.BlockedModels, false
}

// blockedModelsLocked resolves the deny list for one key: the global list plus
// either the runtime override or the configured per-key list.
func (s *store) blockedModelsLocked(id string, entry *keyState) []string {
	global, _ := s.effectiveGlobalBlockedLocked()
	var own []string
	if entry != nil && entry.BlockedModels != nil {
		own = *entry.BlockedModels
	} else if configured, ok := s.cfg.byID[id]; ok {
		own = configured.BlockedModels
	}
	if len(own) == 0 {
		return cleanPatterns(global)
	}
	if len(global) == 0 {
		return cleanPatterns(own)
	}
	return cleanPatterns(append(append([]string(nil), global...), own...))
}

// priceForModel resolves the price entry and matched pattern for a model.
func (s *store) priceForModel(model string) (modelPrice, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pricing, _ := s.effectivePricingLocked()
	return pricing.priceForModel(model)
}

// noteLocked resolves the note in force for a key: the runtime override when
// one is set, otherwise the configured note.
func (s *store) noteLocked(id string, entry *keyState) string {
	if entry != nil && entry.Note != nil {
		return *entry.Note
	}
	return s.cfg.noteForID(id)
}

// effectiveLimitsLocked resolves config limits overlaid with runtime overrides.
func (s *store) effectiveLimitsLocked(id string, entry *keyState) (limitSet, bool) {
	effective, disabled := s.cfg.limitsForID(id)
	if entry == nil {
		return effective, disabled
	}
	if entry.Overrides != nil {
		effective = effective.merge(*entry.Overrides)
	}
	if entry.Disabled != nil {
		disabled = *entry.Disabled
	}
	return effective, disabled
}

// record applies a completed usage sample to the accounting state.
func (s *store) record(id, hint, model string, delta counters, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entryLocked(id, hint, true)
	s.rollLocked(entry, at)
	entry.Daily.Counters.add(delta)
	entry.Monthly.Counters.add(delta)
	entry.Total.Counters.add(delta)
	s.recordWindowLocked(id, entry, delta, at)
	// The TPM window is measured at completion, when the tokens are known,
	// not at the request start the usage record carries.
	s.recordTokensLocked(id, delta.Tokens, s.now())
	if model = strings.TrimSpace(model); model != "" {
		if entry.Models == nil {
			entry.Models = map[string]counters{}
		}
		aggregate := entry.Models[model]
		aggregate.add(delta)
		entry.Models[model] = aggregate
	}
	entry.LastUsedAt = at.In(s.cfg.location).Format(time.RFC3339)
	s.dirty = true
	s.persistLocked(false)
}

// countBlocked records that a request was rejected for the key.
func (s *store) countBlocked(id, hint string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.entryLocked(id, hint, true)
	s.rollLocked(entry, at)
	entry.Blocked++
	s.dirty = true
	s.persistLocked(false)
}

// persistLocked writes the state file, honouring the configured throttle unless
// the caller forces an immediate write.
func (s *store) persistLocked(force bool) {
	if !s.dirty || s.path == "" {
		return
	}
	now := s.now()
	interval := time.Duration(s.cfg.PersistIntervalSecs) * time.Second
	if !force && interval > 0 && !s.lastSave.IsZero() && now.Sub(s.lastSave) < interval {
		return
	}
	if errWrite := writeStateFile(s.path, s.state); errWrite != nil {
		hostLogf("apikey-quota: persisting state failed: %v", errWrite)
		return
	}
	s.dirty = false
	s.lastSave = now
}

// flush forces a state file write.
func (s *store) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistLocked(true)
}

func writeStateFile(path string, state stateFile) error {
	dir := filepath.Dir(path)
	if errMkdir := os.MkdirAll(dir, 0o755); errMkdir != nil {
		return fmt.Errorf("creating state directory %s: %w", dir, errMkdir)
	}
	raw, errMarshal := json.MarshalIndent(state, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("encoding state: %w", errMarshal)
	}
	temp, errTemp := os.CreateTemp(dir, ".apikey-quota-*.tmp")
	if errTemp != nil {
		return fmt.Errorf("creating temp state file: %w", errTemp)
	}
	tempName := temp.Name()
	if _, errWrite := temp.Write(raw); errWrite != nil {
		closeAndRemoveTemp(temp, tempName)
		return fmt.Errorf("writing temp state file: %w", errWrite)
	}
	if errClose := temp.Close(); errClose != nil {
		removeTemp(tempName)
		return fmt.Errorf("closing temp state file: %w", errClose)
	}
	if errRename := os.Rename(tempName, path); errRename != nil {
		removeTemp(tempName)
		return fmt.Errorf("replacing state file %s: %w", path, errRename)
	}
	return nil
}

func closeAndRemoveTemp(temp *os.File, name string) {
	if errClose := temp.Close(); errClose != nil {
		hostLogf("apikey-quota: closing temp state file failed: %v", errClose)
	}
	removeTemp(name)
}

func removeTemp(name string) {
	if errRemove := os.Remove(name); errRemove != nil && !os.IsNotExist(errRemove) {
		hostLogf("apikey-quota: removing temp state file failed: %v", errRemove)
	}
}

// sortedKeyIDs returns every tracked identifier plus every configured one.
func (s *store) sortedKeyIDs() []string {
	seen := make(map[string]struct{}, len(s.state.Keys)+len(s.cfg.byID))
	ids := make([]string, 0, len(s.state.Keys)+len(s.cfg.byID))
	for id := range s.state.Keys {
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	for id := range s.cfg.byID {
		if _, ok := seen[id]; ok {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
