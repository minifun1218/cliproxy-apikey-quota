package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	routePrefix     = "/apikey-quota"
	routeUsage      = routePrefix + "/usage"
	routeConfig     = routePrefix + "/config"
	routeLimits     = routePrefix + "/limits"
	routeReset      = routePrefix + "/reset"
	routePricing    = routePrefix + "/pricing"
	routeModels     = routePrefix + "/models"
	routeMappings   = routePrefix + "/mappings"
	routeFallbacks  = routePrefix + "/fallbacks"
	resourceDashUrl = "/dashboard"
)

// managementRouteView mirrors pluginapi.ManagementRoute without its host-side handler.
type managementRouteView struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Description string `json:"Description,omitempty"`
}

// resourceRouteView mirrors pluginapi.ResourceRoute without its host-side handler.
type resourceRouteView struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRegistrationView struct {
	Routes    []managementRouteView `json:"routes,omitempty"`
	Resources []resourceRouteView   `json:"resources,omitempty"`
}

func managementRegistration() managementRegistrationView {
	return managementRegistrationView{
		Routes: []managementRouteView{
			{Method: http.MethodGet, Path: routeUsage, Description: "Per API key quota usage and remaining allowance."},
			{Method: http.MethodGet, Path: routeConfig, Description: "Effective apikey-quota plugin settings."},
			{Method: http.MethodPost, Path: routeLimits, Description: "Set or clear runtime limit overrides for one or several API keys."},
			{Method: http.MethodPost, Path: routeReset, Description: "Reset accumulated usage counters for one or several API keys."},
			{Method: http.MethodGet, Path: routePricing, Description: "Effective price table and the models it has been matched against."},
			{Method: http.MethodPost, Path: routePricing, Description: "Set or clear the runtime price table."},
			{Method: http.MethodPost, Path: routeModels, Description: "Set or clear the globally blocked model patterns."},
			{Method: http.MethodPost, Path: routeMappings, Description: "Set or clear the global model mapping rules."},
			{Method: http.MethodPost, Path: routeFallbacks, Description: "Set or clear the global fallback models."},
			{Method: http.MethodPost, Path: routeRotate, Description: "Move one API key's usage and settings to its replacement key."},
			{Method: http.MethodPost, Path: routeDelete, Description: "Delete the usage records and settings of one or several API keys."},
			{Method: http.MethodGet, Path: routeAccounts, Description: "Host accounts with their live concurrency, rules, and bound API keys."},
			{Method: http.MethodPost, Path: routeAccounts, Description: "Set or clear the runtime account rules."},
		},
		Resources: []resourceRouteView{
			{Path: resourceDashUrl, Menu: "API Key Quota", Description: "Dashboard for API key quota usage and limits."},
		},
	}
}

// limitView renders one limit ceiling. A nil field means unlimited.
type limitView struct {
	Tokens   *int64   `json:"tokens"`
	Requests *int64   `json:"requests"`
	CostUSD  *float64 `json:"cost_usd"`
}

func newLimitView(limit limits) limitView {
	view := limitView{}
	if limit.Tokens > 0 {
		value := limit.Tokens
		view.Tokens = &value
	}
	if limit.Requests > 0 {
		value := limit.Requests
		view.Requests = &value
	}
	if limit.CostUSD > 0 {
		value := limit.CostUSD
		view.CostUSD = &value
	}
	return view
}

// remainingView reports the allowance left before each ceiling is reached.
type remainingView struct {
	Tokens   *int64   `json:"tokens"`
	Requests *int64   `json:"requests"`
	CostUSD  *float64 `json:"cost_usd"`
}

func newRemainingView(used counters, limit limits) remainingView {
	view := remainingView{}
	if limit.Tokens > 0 {
		value := limit.Tokens - used.Tokens
		if value < 0 {
			value = 0
		}
		view.Tokens = &value
	}
	if limit.Requests > 0 {
		value := limit.Requests - used.Requests
		if value < 0 {
			value = 0
		}
		view.Requests = &value
	}
	if limit.CostUSD > 0 {
		value := limit.CostUSD - used.CostUSD
		if value < 0 {
			value = 0
		}
		view.CostUSD = &value
	}
	return view
}

type periodView struct {
	Period    string        `json:"period"`
	Used      counters      `json:"used"`
	Limits    limitView     `json:"limits"`
	Remaining remainingView `json:"remaining"`
}

type keyView struct {
	ID            string                `json:"id"`
	Label         string                `json:"label,omitempty"`
	Note          string                `json:"note"`
	Hint          string                `json:"hint,omitempty"`
	Configured    bool                  `json:"configured"`
	Disabled      bool                  `json:"disabled"`
	Overridden    bool                  `json:"overridden"`
	Blocked       int64                 `json:"blocked"`
	Concurrent    int                   `json:"concurrent"`
	LastUsedAt    string                `json:"last_used_at,omitempty"`
	Periods       map[string]periodView `json:"periods"`
	Models        map[string]counters   `json:"models"`
	BlockedModels []string              `json:"blocked_models"`
	OwnBlocked    []string              `json:"own_blocked_models"`
	Mapped        int64                 `json:"mapped"`
	ModelMappings []modelMapping        `json:"model_mappings"`
	OwnMappings   []modelMapping        `json:"own_model_mappings"`
	// FallbackModels is the chain in force; OwnFallbacks is the key's own
	// chain, nil when the key follows the global one.
	FallbackModels []string  `json:"fallback_models"`
	OwnFallbacks   *[]string `json:"own_fallback_models"`
	Fallbacks      int64     `json:"fallbacks"`
	// Rate reports the last minute of activity against the rate limits in
	// force now, including those of an active schedule window.
	Rate rateView `json:"rate"`
	// BaseRateLimits are the rate limits before any schedule window applies;
	// zero means unlimited.
	BaseRateLimits rateLimits `json:"base_rate_limits"`
	// Schedule is the schedule in force and ScheduleSource where it came
	// from: runtime, config, default, or empty for none.
	Schedule       *schedule `json:"schedule"`
	ScheduleSource string    `json:"schedule_source"`
	// Window describes the schedule window active now, if any.
	Window *windowView `json:"window"`
	// OutsideSchedule reports that the schedule refuses the key right now.
	OutsideSchedule bool `json:"outside_schedule"`
	// Accounts are the account patterns bound to the key, and StrictAccounts
	// whether the key may use no other account.
	Accounts       []string `json:"accounts"`
	StrictAccounts bool     `json:"strict_accounts"`
}

// rateDim is one rate dimension: the last minute's use and its ceiling, where
// a nil limit means unlimited.
type rateDim struct {
	Used  int64  `json:"used"`
	Limit *int64 `json:"limit"`
}

type rateView struct {
	RPM         rateDim `json:"rpm"`
	TPM         rateDim `json:"tpm"`
	Concurrency rateDim `json:"concurrency"`
}

func newRateDim(used, limit int64) rateDim {
	dim := rateDim{Used: used}
	if limit > 0 {
		value := limit
		dim.Limit = &value
	}
	return dim
}

// unlimitedRate clears negative (explicitly unlimited) fields to zero.
func unlimitedRate(rate rateLimits) rateLimits {
	if rate.RPM < 0 {
		rate.RPM = 0
	}
	if rate.TPM < 0 {
		rate.TPM = 0
	}
	if rate.Concurrency < 0 {
		rate.Concurrency = 0
	}
	if rate.QueueSeconds < 0 {
		rate.QueueSeconds = 0
	}
	return rate
}

type windowView struct {
	Name      string        `json:"name"`
	Block     bool          `json:"block"`
	Start     string        `json:"start"`
	End       string        `json:"end"`
	Used      counters      `json:"used"`
	Limits    limitView     `json:"limits"`
	Remaining remainingView `json:"remaining"`
}

type usageView struct {
	GeneratedAt   string    `json:"generated_at"`
	Enforce       bool      `json:"enforce"`
	TimeZone      string    `json:"time_zone"`
	Totals        counters  `json:"totals"`
	Concurrent    int       `json:"concurrent"`
	Keys          []keyView `json:"keys"`
	BlockedModels []string  `json:"blocked_models"`
	ModelsCustom  bool      `json:"blocked_models_overridden"`
	// ModelMappings are the global mapping rules in force.
	ModelMappings  []modelMapping `json:"model_mappings"`
	MappingsCustom bool           `json:"model_mappings_overridden"`
	// FallbackModels is the global fallback chain in force.
	FallbackModels  []string `json:"fallback_models"`
	FallbacksCustom bool     `json:"fallback_models_overridden"`
	// Providers lists the built-in providers the host last offered to
	// model.route; it stays empty until the first request is routed.
	Providers []string `json:"providers"`
}

// usageSnapshot renders the full accounting state for management clients.
func (s *store) usageSnapshot(at time.Time) usageView {
	s.mu.Lock()
	defer s.mu.Unlock()

	view := usageView{
		GeneratedAt: at.In(s.cfg.location).Format(time.RFC3339),
		Enforce:     s.cfg.Enforce,
		TimeZone:    s.cfg.TimeZone,
		Keys:        []keyView{},
	}
	globalBlocked, globalCustom := s.effectiveGlobalBlockedLocked()
	view.BlockedModels = globalBlocked
	if view.BlockedModels == nil {
		view.BlockedModels = []string{}
	}
	view.ModelsCustom = globalCustom
	globalMappings, mappingsCustom := s.effectiveGlobalMappingsLocked()
	view.ModelMappings = nonNilMappings(globalMappings)
	view.MappingsCustom = mappingsCustom
	globalFallbacks, fallbacksCustom := s.effectiveGlobalFallbacksLocked()
	view.FallbackModels = nonNilStrings(globalFallbacks)
	view.FallbacksCustom = fallbacksCustom
	view.Providers = append([]string{}, s.providers...)

	daily, monthly := s.periodsAt(at)
	concurrency := s.concurrencyLocked(at)
	for _, id := range s.sortedKeyIDs() {
		entry := s.state.Keys[id]
		effective, disabled := s.effectiveLimitsLocked(id, entry)
		configured, hasConfig := s.cfg.byID[id]

		item := keyView{
			ID:            id,
			Configured:    hasConfig,
			Disabled:      disabled,
			Periods:       map[string]periodView{},
			BlockedModels: s.blockedModelsLocked(id, entry),
			Note:          s.noteLocked(id, entry),
			Models:        map[string]counters{},
			Concurrent:    concurrency[id],
		}
		if entry != nil && entry.BlockedModels != nil {
			item.OwnBlocked = *entry.BlockedModels
		} else if hasConfig {
			item.OwnBlocked = configured.BlockedModels
		}
		if item.OwnBlocked == nil {
			item.OwnBlocked = []string{}
		}
		if item.BlockedModels == nil {
			item.BlockedModels = []string{}
		}
		item.ModelMappings = nonNilMappings(s.mappingsLocked(id, entry))
		item.OwnMappings = nonNilMappings(s.ownMappingsLocked(id, entry))
		item.FallbackModels = nonNilStrings(s.fallbacksLocked(id, entry))
		if own, ok := s.ownFallbacksLocked(id, entry); ok {
			list := nonNilStrings(own)
			item.OwnFallbacks = &list
		}
		rpm, tpm := s.rateUsageLocked(id, at)
		rate := s.effectiveRateLocked(id, entry, at)
		item.Rate = rateView{
			RPM:         newRateDim(rpm, rate.RPM),
			TPM:         newRateDim(tpm, rate.TPM),
			Concurrency: newRateDim(int64(concurrency[id]), rate.Concurrency),
		}
		item.BaseRateLimits = unlimitedRate(s.baseRateLocked(id, entry))
		accounts, strict := s.ownAccountsLocked(id, entry)
		item.Accounts = nonNilStrings(accounts)
		item.StrictAccounts = strict
		sched, source := s.scheduleLocked(id, entry)
		item.Schedule = cloneSchedule(sched)
		item.ScheduleSource = source
		active, inWindow, outside := sched.blockedAt(at, s.cfg.location)
		item.OutsideSchedule = outside
		if inWindow {
			used := windowUsed(entry, active)
			item.Window = &windowView{
				Name:      active.Window.Name,
				Block:     active.Window.Block,
				Start:     active.Start.Format(time.RFC3339),
				End:       active.End.Format(time.RFC3339),
				Used:      used,
				Limits:    newLimitView(active.Window.Limits),
				Remaining: newRemainingView(used, active.Window.Limits),
			}
		}
		if hasConfig {
			item.Label = configured.Label
			item.Hint = maskKey(configured.Key)
		}

		var dailyUsed, monthlyUsed, totalUsed counters
		if entry != nil {
			if entry.Label != "" {
				item.Label = entry.Label
			}
			if item.Hint == "" {
				item.Hint = entry.Hint
			}
			item.Blocked = entry.Blocked
			item.Mapped = entry.Mapped
			item.Fallbacks = entry.Fallbacks
			item.LastUsedAt = entry.LastUsedAt
			if entry.Models != nil {
				item.Models = entry.Models
			}
			item.Overridden = entry.Overrides != nil || entry.Disabled != nil ||
				entry.BlockedModels != nil || entry.ModelMappings != nil || entry.FallbackModels != nil || entry.Note != nil ||
				entry.RateOverrides != nil || entry.Schedule != nil || entry.Accounts != nil || entry.StrictAccounts != nil
			if entry.Daily.Period == daily {
				dailyUsed = entry.Daily.Counters
			}
			if entry.Monthly.Period == monthly {
				monthlyUsed = entry.Monthly.Counters
			}
			totalUsed = entry.Total.Counters
		}

		item.Periods[periodDaily] = periodView{
			Period:    daily,
			Used:      dailyUsed,
			Limits:    newLimitView(effective.Daily),
			Remaining: newRemainingView(dailyUsed, effective.Daily),
		}
		item.Periods[periodMonthly] = periodView{
			Period:    monthly,
			Used:      monthlyUsed,
			Limits:    newLimitView(effective.Monthly),
			Remaining: newRemainingView(monthlyUsed, effective.Monthly),
		}
		item.Periods[periodTotal] = periodView{
			Period:    totalPeriodName,
			Used:      totalUsed,
			Limits:    newLimitView(effective.Total),
			Remaining: newRemainingView(totalUsed, effective.Total),
		}
		view.Totals.add(totalUsed)
		view.Concurrent += item.Concurrent
		view.Keys = append(view.Keys, item)
	}
	return view
}

type configView struct {
	Enforce             bool     `json:"enforce"`
	StateFile           string   `json:"state_file"`
	PersistIntervalSecs int      `json:"persist_interval_seconds"`
	RejectStatus        int      `json:"reject_status"`
	TimeZone            string   `json:"time_zone"`
	CountFailedRequests bool     `json:"count_failed_requests"`
	TrackUnknownKeys    bool     `json:"track_unknown_keys"`
	DefaultLimits       limitSet `json:"default_limits"`
	ConfiguredKeys      int      `json:"configured_keys"`
	PricedModels        int      `json:"priced_models"`
	PricingSource       string   `json:"pricing_source"`
	BlockedModels       []string `json:"blocked_models"`

	ModelMappings       []modelMapping `json:"model_mappings"`
	FallbackModels      []string       `json:"fallback_models"`
	FallbackStatusCodes []int          `json:"fallback_status_codes"`
	DefaultRateLimits   rateLimits     `json:"default_rate_limits"`
	DefaultSchedule     *schedule      `json:"default_schedule"`
}

func (s *store) configSnapshot() configView {
	s.mu.Lock()
	defer s.mu.Unlock()
	pricing, pricingSource := s.effectivePricingLocked()
	blocked, _ := s.effectiveGlobalBlockedLocked()
	if blocked == nil {
		blocked = []string{}
	}
	mappings, _ := s.effectiveGlobalMappingsLocked()
	fallbacks, _ := s.effectiveGlobalFallbacksLocked()
	statuses := s.cfg.FallbackStatusCodes
	if len(statuses) == 0 {
		statuses = defaultFallbackStatuses
	}
	return configView{
		Enforce:             s.cfg.Enforce,
		StateFile:           s.path,
		PersistIntervalSecs: s.cfg.PersistIntervalSecs,
		RejectStatus:        s.cfg.RejectStatus,
		TimeZone:            s.cfg.TimeZone,
		CountFailedRequests: s.cfg.CountFailedRequests,
		TrackUnknownKeys:    s.cfg.TrackUnknownKeys,
		DefaultLimits:       s.cfg.DefaultLimits,
		ConfiguredKeys:      len(s.cfg.byID),
		PricedModels:        len(pricing.Models),
		PricingSource:       pricingSource,
		BlockedModels:       blocked,
		ModelMappings:       nonNilMappings(mappings),
		FallbackModels:      nonNilStrings(fallbacks),
		FallbackStatusCodes: append([]int{}, statuses...),
		DefaultRateLimits:   s.cfg.DefaultRateLimits,
		DefaultSchedule:     cloneSchedule(s.cfg.DefaultSchedule),
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilMappings(mappings []modelMapping) []modelMapping {
	if mappings == nil {
		return []modelMapping{}
	}
	return mappings
}

// observedModelView reports a model the plugin has actually accounted for,
// together with the price entry that was matched to compute its cost. Priced is
// false when no rule matched and the default price is zero, which means the
// model contributes nothing to cost quotas.
type observedModelView struct {
	Model   string     `json:"model"`
	Matched string     `json:"matched"`
	Price   modelPrice `json:"price"`
	Used    counters   `json:"used"`
	Blocked bool       `json:"blocked"`
	Priced  bool       `json:"priced"`
}

type pricingView struct {
	Default    modelPrice          `json:"default"`
	Models     []modelPrice        `json:"models"`
	Source     string              `json:"source"`
	Overridden bool                `json:"overridden"`
	Observed   []observedModelView `json:"observed"`
}

// pricingSnapshot renders the effective price table plus every model the plugin
// has priced so far, so an operator can see which rule a model resolved to.
func (s *store) pricingSnapshot() pricingView {
	s.mu.Lock()
	defer s.mu.Unlock()

	pricing, source := s.effectivePricingLocked()
	view := pricingView{
		Default:    pricing.Default,
		Models:     pricing.Models,
		Source:     source,
		Overridden: source == pricingSourceRuntime,
		Observed:   []observedModelView{},
	}
	if view.Models == nil {
		view.Models = []modelPrice{}
	}

	aggregate := map[string]counters{}
	for _, entry := range s.state.Keys {
		for model, used := range entry.Models {
			total := aggregate[model]
			total.add(used)
			aggregate[model] = total
		}
	}
	models := make([]string, 0, len(aggregate))
	for model := range aggregate {
		models = append(models, model)
	}
	sort.Strings(models)

	globalBlocked, _ := s.effectiveGlobalBlockedLocked()
	for _, model := range models {
		price, matched := pricing.priceForModel(model)
		_, blocked := matchPattern(globalBlocked, model)
		view.Observed = append(view.Observed, observedModelView{
			Model:   model,
			Matched: matched,
			Price:   price,
			Used:    aggregate[model],
			Blocked: blocked,
			Priced:  price.Input != 0 || price.Output != 0 || price.Reasoning != 0,
		})
	}
	return view
}

type pricingRequest struct {
	Default *modelPrice  `json:"default"`
	Models  []modelPrice `json:"models"`
	Clear   bool         `json:"clear"`
}

// applyPricing stores a runtime price table that replaces the configured one.
func (s *store) applyPricing(req pricingRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Clear {
		s.settingsLocked().Pricing = nil
		s.dirty = true
		s.persistLocked(true)
		return nil
	}
	pricing := pricingConfig{Models: []modelPrice{}}
	if req.Default != nil {
		pricing.Default = *req.Default
	}
	for index := range req.Models {
		entry := req.Models[index]
		entry.Match = strings.TrimSpace(entry.Match)
		if entry.Match == "" {
			return fmt.Errorf("models[%d].match must not be empty", index)
		}
		pricing.Models = append(pricing.Models, entry)
	}
	s.settingsLocked().Pricing = &pricing
	s.dirty = true
	s.persistLocked(true)
	return nil
}

type modelsRequest struct {
	BlockedModels []string `json:"blocked_models"`
	Clear         bool     `json:"clear"`
}

// applyGlobalModels stores the deny list applied to every API key.
func (s *store) applyGlobalModels(req modelsRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Clear {
		s.settingsLocked().BlockedModels = nil
	} else {
		patterns := cleanPatterns(req.BlockedModels)
		if patterns == nil {
			patterns = []string{}
		}
		s.settingsLocked().BlockedModels = &patterns
	}
	s.dirty = true
	s.persistLocked(true)
}

type limitsRequest struct {
	ID string `json:"id"`
	// IDs applies the same change to several keys at once. It may be combined
	// with ID; duplicates are ignored.
	IDs           []string        `json:"ids"`
	Limits        *limitSet       `json:"limits"`
	Disabled      *bool           `json:"disabled"`
	BlockedModels *[]string       `json:"blocked_models"`
	ModelMappings *[]modelMapping `json:"model_mappings"`
	// FallbackModels replaces the key's fallback chain; an empty list turns
	// fallback off for the key. ClearFallbacks drops that replacement so the
	// configured or global chain applies again.
	FallbackModels *[]string   `json:"fallback_models"`
	ClearFallbacks bool        `json:"clear_fallback_models"`
	Note           *string     `json:"note"`
	RateLimits     *rateLimits `json:"rate_limits"`
	// Schedule replaces the key's schedule; ClearSchedule drops that
	// replacement so the configured schedule applies again.
	Schedule      *schedule `json:"schedule"`
	ClearSchedule bool      `json:"clear_schedule"`
	// Accounts replaces the key's account binding; an empty list unbinds it.
	// ClearAccounts drops the replacement and the strict flag override so the
	// configured binding applies again.
	Accounts       *[]string `json:"accounts"`
	StrictAccounts *bool     `json:"strict_accounts"`
	ClearAccounts  bool      `json:"clear_accounts"`
	Clear          bool      `json:"clear"`
}

// targetIDs merges a single id and a list of ids into one ordered list
// without blanks or duplicates.
func targetIDs(id string, ids []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, candidate := range append([]string{id}, ids...) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, dup := seen[candidate]; dup {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

// applyLimits stores runtime limit overrides for one or several keys.
func (s *store) applyLimits(req limitsRequest) error {
	if req.ModelMappings != nil {
		if errValidate := validateMappings("model_mappings", *req.ModelMappings); errValidate != nil {
			return errValidate
		}
	}
	if errSchedule := validateSchedule("schedule", req.Schedule); errSchedule != nil {
		return errSchedule
	}
	if req.FallbackModels != nil {
		if errValidate := validateFallbackModels("fallback_models", *req.FallbackModels); errValidate != nil {
			return errValidate
		}
	}
	if req.Accounts != nil {
		if errValidate := validateAccountPatterns("accounts", *req.Accounts); errValidate != nil {
			return errValidate
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range targetIDs(req.ID, req.IDs) {
		s.applyLimitsLocked(id, req)
	}
	s.dirty = true
	s.persistLocked(true)
	return nil
}

func (s *store) applyLimitsLocked(id string, req limitsRequest) {
	entry := s.entryLocked(id, "", true)
	if req.Clear {
		entry.Overrides = nil
		entry.Disabled = nil
		entry.BlockedModels = nil
		entry.ModelMappings = nil
		entry.FallbackModels = nil
		entry.Note = nil
		entry.RateOverrides = nil
		entry.Schedule = nil
		entry.Accounts = nil
		entry.StrictAccounts = nil
	}
	if req.ClearAccounts {
		entry.Accounts = nil
		entry.StrictAccounts = nil
	}
	if req.Accounts != nil {
		accounts := cleanPatterns(*req.Accounts)
		if accounts == nil {
			accounts = []string{}
		}
		entry.Accounts = &accounts
	}
	if req.StrictAccounts != nil {
		strict := *req.StrictAccounts
		entry.StrictAccounts = &strict
	}
	if req.ClearSchedule {
		entry.Schedule = nil
	}
	if req.ClearFallbacks {
		entry.FallbackModels = nil
	}
	if req.FallbackModels != nil {
		models := cleanFallbackModels(*req.FallbackModels)
		if models == nil {
			models = []string{}
		}
		entry.FallbackModels = &models
	}
	if req.Schedule != nil {
		entry.Schedule = cloneSchedule(req.Schedule)
	}
	if req.RateLimits != nil {
		rate := *req.RateLimits
		entry.RateOverrides = &rate
	}
	if req.Limits != nil {
		overrides := *req.Limits
		entry.Overrides = &overrides
	}
	if req.Disabled != nil {
		disabled := *req.Disabled
		entry.Disabled = &disabled
	}
	if req.BlockedModels != nil {
		patterns := cleanPatterns(*req.BlockedModels)
		if patterns == nil {
			patterns = []string{}
		}
		entry.BlockedModels = &patterns
	}
	if req.ModelMappings != nil {
		mappings := cleanMappings(*req.ModelMappings)
		if mappings == nil {
			mappings = []modelMapping{}
		}
		entry.ModelMappings = &mappings
	}
	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		entry.Note = &note
	}
}

type resetRequest struct {
	ID     string   `json:"id"`
	IDs    []string `json:"ids"`
	Period string   `json:"period"`
}

// resetUsage clears the counters of one or every period for one or several keys.
func (s *store) resetUsage(req resetRequest, at time.Time) error {
	period := strings.ToLower(strings.TrimSpace(req.Period))
	if period == "" {
		period = "all"
	}
	switch period {
	case periodDaily, periodMonthly, periodTotal, "all":
	default:
		return errInvalidPeriod
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	daily, monthly := s.periodsAt(at)
	for _, id := range targetIDs(req.ID, req.IDs) {
		resetEntry(s.entryLocked(id, "", true), period, daily, monthly)
	}
	s.dirty = true
	s.persistLocked(true)
	return nil
}

func resetEntry(entry *keyState, period, daily, monthly string) {
	switch period {
	case periodDaily:
		entry.Daily = bucket{Period: daily}
	case periodMonthly:
		entry.Monthly = bucket{Period: monthly}
	case periodTotal:
		entry.Total = bucket{Period: totalPeriodName}
	case "all":
		entry.Daily = bucket{Period: daily}
		entry.Monthly = bucket{Period: monthly}
		entry.Total = bucket{Period: totalPeriodName}
		entry.Models = nil
		entry.Windows = nil
		entry.Blocked = 0
		entry.Mapped = 0
		entry.Fallbacks = 0
	}
}

// handleManagement dispatches one Management API or resource request.
func handleManagement(s *store, raw []byte) ([]byte, error) {
	var req pluginapi.ManagementRequest
	if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
		return nil, errUnmarshal
	}

	path := strings.TrimRight(strings.TrimSpace(req.Path), "/")
	switch {
	case strings.HasSuffix(path, resourceDashUrl):
		return htmlResponse(http.StatusOK, dashboardHTML)
	case strings.HasSuffix(path, routeUsage):
		return jsonResponse(http.StatusOK, s.usageSnapshot(s.now()))
	case strings.HasSuffix(path, routeConfig):
		return jsonResponse(http.StatusOK, s.configSnapshot())
	case strings.HasSuffix(path, routeLimits):
		return handleLimits(s, req)
	case strings.HasSuffix(path, routeReset):
		return handleReset(s, req)
	case strings.HasSuffix(path, routePricing):
		return handlePricing(s, req)
	case strings.HasSuffix(path, routeModels):
		return handleModels(s, req)
	case strings.HasSuffix(path, routeMappings):
		return handleMappings(s, req)
	case strings.HasSuffix(path, routeFallbacks):
		return handleFallbacks(s, req)
	case strings.HasSuffix(path, routeRotate):
		return handleRotate(s, req)
	case strings.HasSuffix(path, routeDelete):
		return handleDelete(s, req)
	case strings.HasSuffix(path, routeAccounts):
		return handleAccounts(s, req)
	default:
		return jsonResponse(http.StatusNotFound, map[string]string{"error": "unknown apikey-quota route"})
	}
}

func handleLimits(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body limitsRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	body.ID = strings.TrimSpace(body.ID)
	ids := targetIDs(body.ID, body.IDs)
	if len(ids) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "id or ids is required"})
	}
	if errApply := s.applyLimits(body); errApply != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": errApply.Error()})
	}
	return jsonResponse(http.StatusOK, map[string]any{"ok": true, "id": body.ID, "ids": ids})
}

func handleReset(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body resetRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	body.ID = strings.TrimSpace(body.ID)
	ids := targetIDs(body.ID, body.IDs)
	if len(ids) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "id or ids is required"})
	}
	if errReset := s.resetUsage(body, s.now()); errReset != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": errReset.Error()})
	}
	return jsonResponse(http.StatusOK, map[string]any{"ok": true, "id": body.ID, "ids": ids})
}

func handlePricing(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	if strings.EqualFold(strings.TrimSpace(req.Method), http.MethodGet) {
		return jsonResponse(http.StatusOK, s.pricingSnapshot())
	}
	var body pricingRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if errApply := s.applyPricing(body); errApply != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": errApply.Error()})
	}
	return jsonResponse(http.StatusOK, s.pricingSnapshot())
}

func handleModels(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body modelsRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	s.applyGlobalModels(body)
	return jsonResponse(http.StatusOK, map[string]any{"ok": true})
}

func handleMappings(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body mappingsRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	if errApply := s.applyGlobalMappings(body); errApply != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": errApply.Error()})
	}
	return jsonResponse(http.StatusOK, map[string]any{"ok": true})
}

func jsonResponse(status int, payload any) ([]byte, error) {
	body, errMarshal := json.Marshal(payload)
	if errMarshal != nil {
		return nil, errMarshal
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json; charset=utf-8")
	return okEnvelope(pluginapi.ManagementResponse{StatusCode: status, Headers: headers, Body: body})
}

func htmlResponse(status int, body string) ([]byte, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "text/html; charset=utf-8")
	return okEnvelope(pluginapi.ManagementResponse{StatusCode: status, Headers: headers, Body: []byte(body)})
}
