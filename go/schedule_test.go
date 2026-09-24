package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// monday is 2026-09-21 10:00 UTC, a Monday.
var monday = time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)

func newClockStore(t *testing.T, yaml string, clock *time.Time) *store {
	t.Helper()
	cfg, errParse := parseConfig([]byte("state_file: " + t.TempDir() + "/state.json\ntime_zone: UTC\n" + yaml))
	if errParse != nil {
		t.Fatal(errParse)
	}
	s := newStore(func() time.Time { return *clock })
	if errConfigure := s.configure(cfg); errConfigure != nil {
		t.Fatal(errConfigure)
	}
	return s
}

// intercept runs one request through the interceptor and returns its status
// (0 when admitted), error code, and Retry-After header.
func intercept(t *testing.T, s *store, key, requestID string) (int, string, string) {
	t.Helper()
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+key)
	raw, _ := json.Marshal(pluginapi.RequestInterceptRequest{Headers: headers, RequestID: requestID, Model: "gpt-5"})
	out, errIntercept := interceptBeforeAuth(s, raw)
	if errIntercept != nil {
		t.Fatal(errIntercept)
	}
	var env struct {
		Result pluginapi.RequestInterceptResponse `json:"result"`
	}
	if errUnmarshal := json.Unmarshal(out, &env); errUnmarshal != nil {
		t.Fatal(errUnmarshal)
	}
	if !env.Result.Terminate {
		return 0, "", ""
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(env.Result.ResponseBody, &body)
	return env.Result.StatusCode, body.Error.Code, env.Result.ResponseHeaders.Get("Retry-After")
}

func TestParseDaysAndClock(t *testing.T) {
	mask, err := parseDays([]string{"mon-fri"})
	if err != nil || mask != 0x3e {
		t.Fatalf("mon-fri: %b %v", mask, err)
	}
	if mask, _ = parseDays([]string{"fri-mon"}); mask != 0x63 {
		t.Fatalf("wrapping range: %b", mask)
	}
	if mask, _ = parseDays(nil); mask != allDays {
		t.Fatalf("empty means every day: %b", mask)
	}
	if _, err = parseDays([]string{"funday"}); err == nil {
		t.Fatal("unknown day must fail")
	}
	for _, bad := range []string{"9", "24:30", "12:60", "ab:cd"} {
		if _, err = parseClock(bad); err == nil {
			t.Fatalf("%q must fail", bad)
		}
	}
	if minutes, _ := parseClock("24:00"); minutes != 1440 {
		t.Fatal("24:00 must parse")
	}
}

func TestValidateScheduleNamesAndDuplicates(t *testing.T) {
	sched := &schedule{Windows: []scheduleWindow{{Start: "09:00", End: "18:00"}}}
	if err := validateSchedule("s", sched); err != nil || sched.Windows[0].Name != "09:00-18:00" || sched.Outside != "allow" {
		t.Fatalf("defaults not applied: %+v %v", sched, err)
	}
	dup := &schedule{Windows: []scheduleWindow{{Name: "a", Start: "01:00", End: "02:00"}, {Name: "A", Start: "03:00", End: "04:00"}}}
	if err := validateSchedule("s", dup); err == nil {
		t.Fatal("duplicate names must fail")
	}
	if err := validateSchedule("s", &schedule{Outside: "maybe"}); err == nil {
		t.Fatal("bad outside must fail")
	}
}

func TestActiveWindowAcrossMidnight(t *testing.T) {
	sched := &schedule{Windows: []scheduleWindow{{Name: "night", Days: []string{"mon"}, Start: "22:00", End: "06:00"}}}
	if _, ok := sched.activeAt(monday, time.UTC); ok {
		t.Fatal("10:00 is outside the night window")
	}
	active, ok := sched.activeAt(monday.Add(13*time.Hour), time.UTC)
	if !ok || active.occurrence() != "2026-09-21T22:00" {
		t.Fatalf("monday 23:00: %+v %v", active, ok)
	}
	// Tuesday 03:00 still belongs to the occurrence that started on Monday.
	active, ok = sched.activeAt(monday.Add(17*time.Hour), time.UTC)
	if !ok || active.occurrence() != "2026-09-21T22:00" || !active.End.Equal(monday.Add(20*time.Hour)) {
		t.Fatalf("tuesday 03:00: %+v %v", active, ok)
	}
	// Tuesday 23:00 is not covered: the window only starts on Mondays.
	if _, ok = sched.activeAt(monday.Add(37*time.Hour), time.UTC); ok {
		t.Fatal("tuesday 23:00 must be outside")
	}
}

func TestScheduleBlocksOutsideWindows(t *testing.T) {
	clock := monday.Add(-4 * time.Hour) // Monday 06:00
	s := newClockStore(t, `
default_schedule:
  outside: block
  windows:
    - name: work
      days: [weekdays]
      start: "09:00"
      end: "18:00"
`, &clock)
	status, code, retry := intercept(t, s, "sk-a", "r1")
	if status != http.StatusForbidden || code != "outside_schedule" || retry != "10800" {
		t.Fatalf("06:00 must be refused until 09:00: %d %s %s", status, code, retry)
	}
	clock = monday
	if status, _, _ = intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatalf("10:00 must be admitted, got %d", status)
	}
	// Friday 20:00 waits for Monday 09:00.
	clock = monday.AddDate(0, 0, 4).Add(10 * time.Hour)
	if _, _, retry = intercept(t, s, "sk-a", "r3"); retry != "219600" {
		t.Fatalf("friday evening must wait until monday: %s", retry)
	}
}

func TestBlockWindowAndWindowQuota(t *testing.T) {
	clock := monday
	s := newClockStore(t, `
default_schedule:
  windows:
    - name: lunch
      start: "12:00"
      end: "13:00"
      block: true
    - name: day
      start: "08:00"
      end: "20:00"
      limits:
        requests: 2
      rate_limits:
        rpm: 100
`, &clock)
	id := keyID("sk-a")
	s.record(id, "", "gpt-5", counters{Requests: 1}, clock)
	s.record(id, "", "gpt-5", counters{Requests: 1}, clock)
	status, code, retry := intercept(t, s, "sk-a", "r1")
	if status != 429 || code != "quota_exceeded" || retry != "36000" {
		t.Fatalf("window quota must refuse until 20:00: %d %s %s", status, code, retry)
	}
	// The next day's occurrence starts with an empty bucket.
	clock = monday.AddDate(0, 0, 1)
	if status, _, _ = intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatalf("new occurrence must admit, got %d", status)
	}
	clock = monday.Add(2*time.Hour + 30*time.Minute)
	if status, code, retry = intercept(t, s, "sk-a", "r3"); status != 403 || code != "outside_schedule" || retry != "1800" {
		t.Fatalf("lunch must block until 13:00: %d %s %s", status, code, retry)
	}
	view := s.usageSnapshot(monday)
	if view.Keys[0].Window == nil || view.Keys[0].Window.Name != "day" || view.Keys[0].Window.Used.Requests != 2 {
		t.Fatalf("usage view must report the active window: %+v", view.Keys[0].Window)
	}
	if limit := view.Keys[0].Rate.RPM.Limit; limit == nil || *limit != 100 {
		t.Fatalf("window rate limits must apply: %+v", view.Keys[0].Rate)
	}
}

func TestRuntimeScheduleOverride(t *testing.T) {
	clock := monday
	s := newClockStore(t, "", &clock)
	id := keyID("sk-a")
	blocked := &schedule{Outside: "block"}
	if err := s.applyLimits(limitsRequest{ID: id, Schedule: blocked}); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := intercept(t, s, "sk-a", "r1"); status != 403 {
		t.Fatalf("runtime schedule must apply, got %d", status)
	}
	if err := s.applyLimits(limitsRequest{ID: id, ClearSchedule: true}); err != nil {
		t.Fatal(err)
	}
	if status, _, _ := intercept(t, s, "sk-a", "r2"); status != 0 {
		t.Fatalf("cleared schedule must admit, got %d", status)
	}
	if err := s.applyLimits(limitsRequest{ID: id, Schedule: &schedule{Outside: "sometimes"}}); err == nil ||
		!strings.Contains(err.Error(), "outside") {
		t.Fatalf("invalid schedule must be rejected: %v", err)
	}
}
