package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	scheduleAllow = "allow"
	scheduleBlock = "block"
	allDays       = uint8(0x7f)
)

// scheduleWindow is one recurring time range of a schedule. While it is active
// it can refuse the key outright, give it a quota of its own that restarts with
// every occurrence, and replace its rate limits.
type scheduleWindow struct {
	// Name identifies the window in quota buckets and messages.
	Name string `yaml:"name" json:"name"`
	// Days lists the days the window starts on: mon..sun, ranges such as
	// mon-fri, or weekdays, weekends and all. Empty means every day.
	Days []string `yaml:"days" json:"days"`
	// Start and End are HH:MM clock times in the plugin time zone. An end at
	// or before the start runs past midnight; equal times cover a whole day.
	Start string `yaml:"start" json:"start"`
	End   string `yaml:"end" json:"end"`
	// Block refuses the key for as long as the window is active.
	Block bool `yaml:"block" json:"block"`
	// Limits is the quota of one occurrence of the window. Zero or negative
	// fields mean no ceiling.
	Limits limits `yaml:"limits" json:"limits"`
	// RateLimits overlays the key's rate limits while the window is active.
	RateLimits rateLimits `yaml:"rate_limits" json:"rate_limits"`
}

// schedule decides when a key may be used and under which limits. The first
// window that matches a moment wins.
type schedule struct {
	// Outside decides moments no window covers: allow (default) or block.
	Outside string           `yaml:"outside" json:"outside"`
	Windows []scheduleWindow `yaml:"windows" json:"windows"`
}

var dayIndex = map[string]int{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}

func dayOf(name string) (int, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if len(name) > 3 {
		name = name[:3]
	}
	index, ok := dayIndex[name]
	return index, ok
}

// parseDays turns a day list into a weekday bit mask.
func parseDays(days []string) (uint8, error) {
	var mask uint8
	for _, raw := range days {
		day := strings.ToLower(strings.TrimSpace(raw))
		switch day {
		case "":
			continue
		case "*", "all", "daily", "everyday":
			mask |= allDays
			continue
		case "weekdays":
			mask |= 0x3e
			continue
		case "weekends":
			mask |= 0x41
			continue
		}
		if from, to, isRange := strings.Cut(day, "-"); isRange {
			first, okFirst := dayOf(from)
			last, okLast := dayOf(to)
			if !okFirst || !okLast {
				return 0, fmt.Errorf("unknown day range %q", raw)
			}
			for index := first; ; index = (index + 1) % 7 {
				mask |= 1 << index
				if index == last {
					break
				}
			}
			continue
		}
		index, ok := dayOf(day)
		if !ok {
			return 0, fmt.Errorf("unknown day %q", raw)
		}
		mask |= 1 << index
	}
	if mask == 0 {
		mask = allDays
	}
	return mask, nil
}

// parseClock reads an HH:MM time as minutes after midnight; 24:00 is allowed.
func parseClock(value string) (int, error) {
	hours, minutes, ok := strings.Cut(strings.TrimSpace(value), ":")
	hour, errHour := strconv.Atoi(hours)
	minute, errMinute := strconv.Atoi(minutes)
	if !ok || errHour != nil || errMinute != nil || hour < 0 || hour > 24 || minute < 0 || minute > 59 ||
		(hour == 24 && minute != 0) {
		return 0, fmt.Errorf("invalid time %q, expected HH:MM", value)
	}
	return hour*60 + minute, nil
}

// span resolves a window's day mask and clock range. A window that does not
// parse never matches; validateSchedule keeps such windows out of the state.
func (w scheduleWindow) span() (uint8, int, int, bool) {
	mask, errDays := parseDays(w.Days)
	start, errStart := parseClock(w.Start)
	end, errEnd := parseClock(w.End)
	if errDays != nil || errStart != nil || errEnd != nil {
		return 0, 0, 0, false
	}
	return mask, start, end, true
}

// validateSchedule normalizes a schedule in place and rejects bad windows.
func validateSchedule(field string, sched *schedule) error {
	if sched == nil {
		return nil
	}
	sched.Outside = strings.ToLower(strings.TrimSpace(sched.Outside))
	if sched.Outside == "" {
		sched.Outside = scheduleAllow
	}
	if sched.Outside != scheduleAllow && sched.Outside != scheduleBlock {
		return fmt.Errorf("%s.outside must be allow or block", field)
	}
	seen := map[string]struct{}{}
	for index := range sched.Windows {
		window := &sched.Windows[index]
		prefix := fmt.Sprintf("%s.windows[%d]", field, index)
		window.Start = strings.TrimSpace(window.Start)
		window.End = strings.TrimSpace(window.End)
		if _, errStart := parseClock(window.Start); errStart != nil {
			return fmt.Errorf("%s.start: %w", prefix, errStart)
		}
		if _, errEnd := parseClock(window.End); errEnd != nil {
			return fmt.Errorf("%s.end: %w", prefix, errEnd)
		}
		days := []string{}
		for _, day := range window.Days {
			if day = strings.ToLower(strings.TrimSpace(day)); day != "" {
				days = append(days, day)
			}
		}
		window.Days = days
		if _, errDays := parseDays(window.Days); errDays != nil {
			return fmt.Errorf("%s.days: %w", prefix, errDays)
		}
		window.Name = strings.TrimSpace(window.Name)
		if window.Name == "" {
			window.Name = window.Start + "-" + window.End
		}
		lowered := strings.ToLower(window.Name)
		if _, dup := seen[lowered]; dup {
			return fmt.Errorf("%s.name %q is used by another window", prefix, window.Name)
		}
		seen[lowered] = struct{}{}
	}
	return nil
}

// cloneSchedule deep-copies a schedule so entries never share slices.
func cloneSchedule(sched *schedule) *schedule {
	if sched == nil {
		return nil
	}
	out := &schedule{Outside: sched.Outside, Windows: make([]scheduleWindow, len(sched.Windows))}
	for index, window := range sched.Windows {
		window.Days = append([]string{}, window.Days...)
		out.Windows[index] = window
	}
	return out
}

// activeWindow is one occurrence of a window that covers a moment.
type activeWindow struct {
	Window scheduleWindow
	Start  time.Time
	End    time.Time
}

// occurrence names the bucket period of one occurrence of the window.
func (a activeWindow) occurrence() string {
	return a.Start.Format("2006-01-02T15:04")
}

func clockOn(day time.Time, minutes int, location *time.Location) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), minutes/60, minutes%60, 0, 0, location)
}

// occurrenceFrom returns the occurrence of a window that starts on the given
// calendar day, when the window runs on that day.
func occurrenceFrom(window scheduleWindow, day time.Time, location *time.Location) (activeWindow, bool) {
	mask, start, end, ok := window.span()
	if !ok || mask&(1<<uint(day.Weekday())) == 0 {
		return activeWindow{}, false
	}
	from := clockOn(day, start, location)
	until := clockOn(day, end, location)
	if end <= start {
		until = clockOn(day.AddDate(0, 0, 1), end, location)
	}
	return activeWindow{Window: window, Start: from, End: until}, true
}

// activeAt returns the first window covering the moment. Occurrences that
// started the day before are considered too, for windows past midnight.
func (sched *schedule) activeAt(at time.Time, location *time.Location) (activeWindow, bool) {
	if sched == nil {
		return activeWindow{}, false
	}
	local := at.In(location)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	for _, window := range sched.Windows {
		for _, day := range []time.Time{today, today.AddDate(0, 0, -1)} {
			occurrence, ok := occurrenceFrom(window, day, location)
			if ok && !at.Before(occurrence.Start) && at.Before(occurrence.End) {
				return occurrence, true
			}
		}
	}
	return activeWindow{}, false
}

// blockedAt reports whether the schedule refuses the key at the moment, and
// the window that decided it, if any.
func (sched *schedule) blockedAt(at time.Time, location *time.Location) (activeWindow, bool, bool) {
	if sched == nil {
		return activeWindow{}, false, false
	}
	active, ok := sched.activeAt(at, location)
	if ok {
		return active, true, active.Window.Block
	}
	return activeWindow{}, false, sched.Outside == scheduleBlock
}

// nextAllowed finds the first moment within the coming week at which the
// schedule admits the key again. Only window boundaries can change the
// outcome, so those are the only moments tested.
func (sched *schedule) nextAllowed(at time.Time, location *time.Location) (time.Time, bool) {
	local := at.In(location)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	candidates := []time.Time{}
	for offset := -1; offset <= 8; offset++ {
		day := today.AddDate(0, 0, offset)
		for _, window := range sched.Windows {
			if occurrence, ok := occurrenceFrom(window, day, location); ok {
				candidates = append(candidates, occurrence.Start, occurrence.End)
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Before(candidates[j]) })
	for _, candidate := range candidates {
		if !candidate.After(at) {
			continue
		}
		if _, _, blocked := sched.blockedAt(candidate, location); !blocked {
			return candidate, true
		}
	}
	return time.Time{}, false
}

// scheduleLocked resolves the schedule in force for a key and where it came
// from: "runtime" for a dashboard override, "config" for the key's entry in
// the configuration, "default" for default_schedule, or empty for none.
func (s *store) scheduleLocked(id string, entry *keyState) (*schedule, string) {
	if entry != nil && entry.Schedule != nil {
		return entry.Schedule, "runtime"
	}
	if configured, ok := s.cfg.byID[id]; ok && configured.Schedule != nil {
		return configured.Schedule, "config"
	}
	if s.cfg.DefaultSchedule != nil {
		return s.cfg.DefaultSchedule, "default"
	}
	return nil, ""
}

func (s *store) activeWindowLocked(id string, entry *keyState, at time.Time) (activeWindow, bool) {
	sched, _ := s.scheduleLocked(id, entry)
	return sched.activeAt(at, s.cfg.location)
}

// recordWindowLocked adds usage to the bucket of the window active at the
// request time and drops buckets of windows that no longer exist.
func (s *store) recordWindowLocked(id string, entry *keyState, delta counters, at time.Time) {
	sched, _ := s.scheduleLocked(id, entry)
	if sched == nil {
		entry.Windows = nil
		return
	}
	for name := range entry.Windows {
		found := false
		for _, window := range sched.Windows {
			if window.Name == name {
				found = true
				break
			}
		}
		if !found {
			delete(entry.Windows, name)
		}
	}
	active, ok := sched.activeAt(at, s.cfg.location)
	if !ok {
		return
	}
	if entry.Windows == nil {
		entry.Windows = map[string]bucket{}
	}
	current := entry.Windows[active.Window.Name]
	current.roll(active.occurrence())
	current.Counters.add(delta)
	entry.Windows[active.Window.Name] = current
}

// windowUsed reports the usage of the current occurrence of a window.
func windowUsed(entry *keyState, active activeWindow) counters {
	if entry == nil || entry.Windows == nil {
		return counters{}
	}
	current, ok := entry.Windows[active.Window.Name]
	if !ok || current.Period != active.occurrence() {
		return counters{}
	}
	return current.Counters
}
