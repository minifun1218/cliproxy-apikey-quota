package main

// builtinPrices is the price table used when the plugin configuration carries no
// pricing of its own. Values are USD per one million tokens, taken from the
// providers' published rates in September 2026:
//
//   - Anthropic: https://docs.anthropic.com/en/docs/about-claude/pricing
//   - OpenAI:    https://developers.openai.com/api/docs/pricing
//   - Google:    https://ai.google.dev/gemini-api/docs/pricing
//
// Published prices change and promotional rates expire, so treat this as a
// starting point. Override it in the configuration or from the dashboard rather
// than editing this file. Anthropic cache rates follow the standard multipliers
// (read at 0.1x input, five-minute write at 1.25x input); OpenAI and Google
// publish cached input rates directly and charge nothing to write a cache entry.
var builtinPrices = []modelPrice{
	// Anthropic.
	{Match: "claude-fable-5-1*", Input: 10, Output: 50, CacheRead: 0.25, CacheWrite: 12.5},
	{Match: "claude-fable-5*", Input: 10, Output: 50, CacheRead: 1, CacheWrite: 12.5},
	{Match: "claude-mythos-5*", Input: 10, Output: 50, CacheRead: 1, CacheWrite: 12.5},
	{Match: "claude-opus-5*", Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25},
	{Match: "claude-opus-4*", Input: 5, Output: 25, CacheRead: 0.5, CacheWrite: 6.25},
	{Match: "claude-sonnet-5*", Input: 2, Output: 10, CacheRead: 0.2, CacheWrite: 2.5},
	{Match: "claude-sonnet-4-6*", Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75},
	{Match: "claude-haiku-4-5*", Input: 1, Output: 5, CacheRead: 0.1, CacheWrite: 1.25},

	// OpenAI.
	{Match: "gpt-5.6-sol*", Input: 4, Output: 20, CacheRead: 0.4},
	{Match: "gpt-5.6-terra*", Input: 2, Output: 12, CacheRead: 0.2},
	{Match: "gpt-5.6-luna*", Input: 0.2, Output: 1.2, CacheRead: 0.02},
	{Match: "gpt-5.5*", Input: 5, Output: 30, CacheRead: 0.5},
	{Match: "gpt-5.4-mini*", Input: 0.75, Output: 4.5, CacheRead: 0.075},
	{Match: "gpt-5.4-nano*", Input: 0.2, Output: 1.25, CacheRead: 0.02},
	{Match: "gpt-5.4*", Input: 2.5, Output: 15, CacheRead: 0.25},
	{Match: "gpt-5.2*", Input: 1.75, Output: 14, CacheRead: 0.175},
	{Match: "gpt-5.1*", Input: 1.25, Output: 10, CacheRead: 0.125},
	{Match: "gpt-5-mini*", Input: 0.25, Output: 2, CacheRead: 0.025},
	{Match: "gpt-5-nano*", Input: 0.05, Output: 0.4, CacheRead: 0.005},
	{Match: "gpt-5*", Input: 1.25, Output: 10, CacheRead: 0.125},

	// Google. The 3.6 to 3.8 Flash rates are promotional through 2026-12-31,
	// after which the published price doubles.
	{Match: "gemini-3.8-flash*", Input: 0.75, Output: 3.75, CacheRead: 0.075},
	{Match: "gemini-3.7-flash*", Input: 0.75, Output: 3.75, CacheRead: 0.075},
	{Match: "gemini-3.6-flash*", Input: 0.75, Output: 3.75, CacheRead: 0.075},
	{Match: "gemini-3.5-flash-lite*", Input: 0.3, Output: 2.5, CacheRead: 0.03},
	{Match: "gemini-3.1-flash-lite*", Input: 0.25, Output: 1.5, CacheRead: 0.025},
	{Match: "gemini-2.5-pro*", Input: 1.25, Output: 10, CacheRead: 0.125},
	{Match: "gemini-2.5-flash-lite*", Input: 0.1, Output: 0.4, CacheRead: 0.01},
	{Match: "gemini-2.5-flash*", Input: 0.3, Output: 2.5, CacheRead: 0.03},
}

// builtinPricing returns a fresh copy of the built-in table so callers cannot
// mutate the package-level slice.
func builtinPricing() pricingConfig {
	models := make([]modelPrice, len(builtinPrices))
	copy(models, builtinPrices)
	return pricingConfig{Models: models}
}

// isEmpty reports whether a price table carries no usable rate, which is how the
// plugin decides to fall back to the built-in table.
func (p pricingConfig) isEmpty() bool {
	if len(p.Models) > 0 {
		return false
	}
	d := p.Default
	return d.Input == 0 && d.Output == 0 && d.Reasoning == 0 && d.CacheRead == 0 && d.CacheWrite == 0
}
