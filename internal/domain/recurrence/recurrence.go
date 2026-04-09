// Package recurrence defines the core domain model for task recurrence scheduling.
// All time values are stored and computed in UTC; conversion to local timezone
// is performed at the transport (I/O) boundary only.
package recurrence

import (
	"fmt"
	"sort"
	"time"
)

// Type is a discriminated enum that identifies the recurrence strategy.
// Using a named string type makes JSON serialisation human-readable and
// keeps the wire format stable across API versions.
type Type string

const (
	// TypeDaily repeats the task every N calendar days (N ≥ 1).
	TypeDaily Type = "daily"

	// TypeMonthlyFixedDay repeats on one fixed calendar day of every month (1–30).
	TypeMonthlyFixedDay Type = "monthly_fixed_day"

	// TypeMonthlyCustomDates repeats on an explicit, user-defined list of
	// calendar days each month. Days outside the valid range for a given
	// month are skipped rather than clamped (deterministic skip semantics).
	TypeMonthlyCustomDates Type = "monthly_custom_dates"

	// TypeMonthlyParity repeats on all even or all odd calendar days.
	TypeMonthlyParity Type = "monthly_parity"
)

// Parity classifies calendar days as even or odd.
type Parity string

const (
	ParityEven Parity = "even"
	ParityOdd  Parity = "odd"
)

// maxConfiguredDay is the highest calendar day index accepted by this module.
// Days 29–31 exist only in some months; the scheduler skips them when the
// target month is too short (see NextOccurrences).
const maxConfiguredDay = 30

// minInterval is the smallest allowed daily interval.
const minInterval = 1

// Rule is the aggregate root for a recurrence configuration.
// Exactly one of the strategy fields must be non-nil, matching the Type value.
// This invariant is enforced by the constructor and validated on reads.
type Rule struct {
	// ID is populated by the persistence layer; zero means "not yet persisted".
	ID     int64  `json:"id"`
	TaskID int64  `json:"task_id"`
	Type   Type   `json:"type"`
	Active bool   `json:"active"`

	// Strategy fields — only the field matching Type is populated.
	Daily           *DailyConfig           `json:"daily,omitempty"`
	MonthlyFixedDay *MonthlyFixedDayConfig `json:"monthly_fixed_day,omitempty"`
	MonthlyCustom   *MonthlyCustomConfig   `json:"monthly_custom_dates,omitempty"`
	MonthlyParity   *MonthlyParityConfig   `json:"monthly_parity,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DailyConfig holds parameters for the daily recurrence strategy.
type DailyConfig struct {
	// IntervalDays is the number of calendar days between occurrences (≥ 1).
	IntervalDays int `json:"interval_days"`
}

// MonthlyFixedDayConfig holds parameters for the fixed-day monthly strategy.
type MonthlyFixedDayConfig struct {
	// Day is the calendar day of the month (1–30).
	Day int `json:"day"`
}

// MonthlyCustomConfig holds parameters for the custom-dates monthly strategy.
type MonthlyCustomConfig struct {
	// Days is the sorted, deduplicated list of calendar days (1–30).
	// At least one day must be present.
	Days []int `json:"days"`
}

// MonthlyParityConfig holds parameters for the parity-based monthly strategy.
type MonthlyParityConfig struct {
	// Parity indicates whether the task runs on even or odd days.
	Parity Parity `json:"parity"`
}

// Validate checks the structural consistency of the Rule.
// It is called by the usecase layer before any persistence operation and by
// NextOccurrences before computing dates.
func (r *Rule) Validate() error {
	if r.TaskID <= 0 {
		return fmt.Errorf("task_id must be a positive integer, got %d", r.TaskID)
	}

	switch r.Type {
	case TypeDaily:
		return r.validateDaily()
	case TypeMonthlyFixedDay:
		return r.validateMonthlyFixedDay()
	case TypeMonthlyCustomDates:
		return r.validateMonthlyCustom()
	case TypeMonthlyParity:
		return r.validateMonthlyParity()
	default:
		return fmt.Errorf("unsupported recurrence type %q; valid values: daily, monthly_fixed_day, monthly_custom_dates, monthly_parity", r.Type)
	}
}

func (r *Rule) validateDaily() error {
	if r.MonthlyFixedDay != nil || r.MonthlyCustom != nil || r.MonthlyParity != nil {
		return fmt.Errorf("only the 'daily' config block must be present when type=%q", TypeDaily)
	}
	if r.Daily == nil {
		return fmt.Errorf("'daily' config block is required when type=%q", TypeDaily)
	}
	if r.Daily.IntervalDays < minInterval {
		return fmt.Errorf("daily.interval_days must be ≥ %d, got %d", minInterval, r.Daily.IntervalDays)
	}
	return nil
}

func (r *Rule) validateMonthlyFixedDay() error {
	if r.Daily != nil || r.MonthlyCustom != nil || r.MonthlyParity != nil {
		return fmt.Errorf("only the 'monthly_fixed_day' config block must be present when type=%q", TypeMonthlyFixedDay)
	}
	if r.MonthlyFixedDay == nil {
		return fmt.Errorf("'monthly_fixed_day' config block is required when type=%q", TypeMonthlyFixedDay)
	}
	if r.MonthlyFixedDay.Day < 1 || r.MonthlyFixedDay.Day > maxConfiguredDay {
		return fmt.Errorf("monthly_fixed_day.day must be between 1 and %d, got %d", maxConfiguredDay, r.MonthlyFixedDay.Day)
	}
	return nil
}

func (r *Rule) validateMonthlyCustom() error {
	if r.Daily != nil || r.MonthlyFixedDay != nil || r.MonthlyParity != nil {
		return fmt.Errorf("only the 'monthly_custom_dates' config block must be present when type=%q", TypeMonthlyCustomDates)
	}
	if r.MonthlyCustom == nil {
		return fmt.Errorf("'monthly_custom_dates' config block is required when type=%q", TypeMonthlyCustomDates)
	}
	if len(r.MonthlyCustom.Days) == 0 {
		return fmt.Errorf("monthly_custom_dates.days must contain at least one day")
	}
	seen := make(map[int]struct{}, len(r.MonthlyCustom.Days))
	for _, d := range r.MonthlyCustom.Days {
		if d < 1 || d > maxConfiguredDay {
			return fmt.Errorf("monthly_custom_dates.days: each day must be between 1 and %d, got %d", maxConfiguredDay, d)
		}
		if _, dup := seen[d]; dup {
			return fmt.Errorf("monthly_custom_dates.days: duplicate day %d", d)
		}
		seen[d] = struct{}{}
	}
	return nil
}

func (r *Rule) validateMonthlyParity() error {
	if r.Daily != nil || r.MonthlyFixedDay != nil || r.MonthlyCustom != nil {
		return fmt.Errorf("only the 'monthly_parity' config block must be present when type=%q", TypeMonthlyParity)
	}
	if r.MonthlyParity == nil {
		return fmt.Errorf("'monthly_parity' config block is required when type=%q", TypeMonthlyParity)
	}
	switch r.MonthlyParity.Parity {
	case ParityEven, ParityOdd:
		return nil
	default:
		return fmt.Errorf("monthly_parity.parity must be 'even' or 'odd', got %q", r.MonthlyParity.Parity)
	}
}

// NextOccurrences returns the next n UTC dates on which the task should run,
// starting strictly after 'from'. The result is sorted ascending.
//
// Calendar edge-case policy:
//   - Days that do not exist in a given month (e.g. day 30 in February) are
//     silently skipped; the next valid month is tried.
//   - Leap years are handled transparently by time.Date normalisation.
//   - DST is irrelevant because all output is UTC.
func (r *Rule) NextOccurrences(from time.Time, n int) ([]time.Time, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, fmt.Errorf("n must be a positive integer, got %d", n)
	}

	from = from.UTC()

	switch r.Type {
	case TypeDaily:
		return nextDaily(from, n, r.Daily.IntervalDays), nil
	case TypeMonthlyFixedDay:
		return nextMonthlyFixed(from, n, r.MonthlyFixedDay.Day), nil
	case TypeMonthlyCustomDates:
		days := sortedUnique(r.MonthlyCustom.Days)
		return nextMonthlyCustom(from, n, days), nil
	case TypeMonthlyParity:
		return nextMonthlyParity(from, n, r.MonthlyParity.Parity), nil
	default:
		// unreachable after Validate()
		return nil, fmt.Errorf("unknown recurrence type: %q", r.Type)
	}
}

// ─── scheduling helpers ──────────────────────────────────────────────────────

// nextDaily generates n evenly-spaced dates every intervalDays after from.
func nextDaily(from time.Time, n, intervalDays int) []time.Time {
	result := make([]time.Time, 0, n)
	cur := midnightUTC(from).AddDate(0, 0, intervalDays)
	for len(result) < n {
		result = append(result, cur)
		cur = cur.AddDate(0, 0, intervalDays)
	}
	return result
}

// nextMonthlyFixed generates n occurrences on a fixed calendar day each month.
// Months where the day does not exist are skipped.
func nextMonthlyFixed(from time.Time, n, day int) []time.Time {
	result := make([]time.Time, 0, n)
	from = midnightUTC(from)
	year, month, _ := from.Date()

	// Start from current month; advance if the target day has already passed.
	candidate := utcDate(year, month, day)
	if !candidate.IsZero() && candidate.After(from) {
		result = append(result, candidate)
	}

	// Advance month by month until we have n results.
	for len(result) < n {
		month++
		if month > 12 {
			month = 1
			year++
		}
		if d := utcDate(year, month, day); !d.IsZero() {
			result = append(result, d)
		}
	}
	return result
}

// nextMonthlyCustom generates n occurrences from a sorted list of days per month.
func nextMonthlyCustom(from time.Time, n int, days []int) []time.Time {
	result := make([]time.Time, 0, n)
	from = midnightUTC(from)
	year, month, _ := from.Date()

	for len(result) < n {
		for _, day := range days {
			if len(result) >= n {
				break
			}
			if d := utcDate(year, month, day); !d.IsZero() && d.After(from) {
				result = append(result, d)
			}
		}
		month++
		if month > 12 {
			month = 1
			year++
		}
	}
	return result
}

// nextMonthlyParity generates n occurrences on all even or all odd days of
// each month, advancing month-by-month.
func nextMonthlyParity(from time.Time, n int, parity Parity) []time.Time {
	result := make([]time.Time, 0, n)
	from = midnightUTC(from)
	year, month, _ := from.Date()
	daysInMonth := daysIn(year, month)

	for len(result) < n {
		for day := 1; day <= daysInMonth; day++ {
			if len(result) >= n {
				break
			}
			if matchesParity(day, parity) {
				if d := time.Date(year, month, day, 0, 0, 0, 0, time.UTC); d.After(from) {
					result = append(result, d)
				}
			}
		}
		month++
		if month > 12 {
			month = 1
			year++
		}
		daysInMonth = daysIn(year, month)
	}
	return result
}

// ─── calendar utilities ───────────────────────────────────────────────────────

// midnightUTC truncates t to the start of the UTC day.
func midnightUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// utcDate returns midnight UTC for the given year/month/day, or zero if the
// day does not exist in that month (e.g. day 30 in February).
func utcDate(year int, month time.Month, day int) time.Time {
	t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	// time.Date normalises overflow (e.g. Feb 30 → Mar 1/2).
	// We detect and reject that overflow to implement skip semantics.
	if t.Month() != month {
		return time.Time{} // zero value signals "skip this month"
	}
	return t
}

// daysIn returns the number of days in the given month, handling leap years.
func daysIn(year int, month time.Month) int {
	// time.Date with day=0 of next month gives the last day of current month.
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// matchesParity returns true when day matches the requested parity.
func matchesParity(day int, parity Parity) bool {
	switch parity {
	case ParityEven:
		return day%2 == 0
	case ParityOdd:
		return day%2 != 0
	default:
		return false
	}
}

// sortedUnique returns a new sorted, deduplicated copy of the input slice.
func sortedUnique(days []int) []int {
	seen := make(map[int]struct{}, len(days))
	out := make([]int, 0, len(days))
	for _, d := range days {
		if _, ok := seen[d]; !ok {
			seen[d] = struct{}{}
			out = append(out, d)
		}
	}
	sort.Ints(out)
	return out
}
