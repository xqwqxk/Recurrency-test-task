package recurrence_test

import (
	"testing"
	"time"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func dailyRule(interval int) *recurrencedomain.Rule {
	return &recurrencedomain.Rule{
		TaskID: 1,
		Type:   recurrencedomain.TypeDaily,
		Active: true,
		Daily:  &recurrencedomain.DailyConfig{IntervalDays: interval},
	}
}

func fixedDayRule(day int) *recurrencedomain.Rule {
	return &recurrencedomain.Rule{
		TaskID:          1,
		Type:            recurrencedomain.TypeMonthlyFixedDay,
		Active:          true,
		MonthlyFixedDay: &recurrencedomain.MonthlyFixedDayConfig{Day: day},
	}
}

func customRule(days []int) *recurrencedomain.Rule {
	return &recurrencedomain.Rule{
		TaskID:        1,
		Type:          recurrencedomain.TypeMonthlyCustomDates,
		Active:        true,
		MonthlyCustom: &recurrencedomain.MonthlyCustomConfig{Days: days},
	}
}

func parityRule(p recurrencedomain.Parity) *recurrencedomain.Rule {
	return &recurrencedomain.Rule{
		TaskID:        1,
		Type:          recurrencedomain.TypeMonthlyParity,
		Active:        true,
		MonthlyParity: &recurrencedomain.MonthlyParityConfig{Parity: p},
	}
}

// ─── Validate tests ───────────────────────────────────────────────────────────

func TestValidate_Daily_OK(t *testing.T) {
	if err := dailyRule(1).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Daily_ZeroInterval(t *testing.T) {
	r := dailyRule(0)
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for interval_days=0")
	}
}

func TestValidate_Daily_NegativeInterval(t *testing.T) {
	r := dailyRule(-3)
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for interval_days=-3")
	}
}

func TestValidate_Daily_MissingConfig(t *testing.T) {
	r := &recurrencedomain.Rule{TaskID: 1, Type: recurrencedomain.TypeDaily}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when daily config is nil")
	}
}

func TestValidate_Daily_ExtraConfig(t *testing.T) {
	r := dailyRule(1)
	r.MonthlyFixedDay = &recurrencedomain.MonthlyFixedDayConfig{Day: 5}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when extra config block is present")
	}
}

func TestValidate_FixedDay_OK(t *testing.T) {
	if err := fixedDayRule(15).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_FixedDay_Zero(t *testing.T) {
	if err := fixedDayRule(0).Validate(); err == nil {
		t.Fatal("expected error for day=0")
	}
}

func TestValidate_FixedDay_31(t *testing.T) {
	if err := fixedDayRule(31).Validate(); err == nil {
		t.Fatal("expected error for day=31 (above maxConfiguredDay=30)")
	}
}

func TestValidate_CustomDates_OK(t *testing.T) {
	if err := customRule([]int{1, 15, 28}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_CustomDates_Empty(t *testing.T) {
	if err := customRule([]int{}).Validate(); err == nil {
		t.Fatal("expected error for empty days list")
	}
}

func TestValidate_CustomDates_Duplicate(t *testing.T) {
	if err := customRule([]int{1, 15, 15}).Validate(); err == nil {
		t.Fatal("expected error for duplicate day")
	}
}

func TestValidate_CustomDates_OutOfRange(t *testing.T) {
	if err := customRule([]int{0, 10}).Validate(); err == nil {
		t.Fatal("expected error for day=0")
	}
	if err := customRule([]int{10, 31}).Validate(); err == nil {
		t.Fatal("expected error for day=31")
	}
}

func TestValidate_Parity_OK(t *testing.T) {
	if err := parityRule(recurrencedomain.ParityEven).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := parityRule(recurrencedomain.ParityOdd).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_Parity_Invalid(t *testing.T) {
	r := &recurrencedomain.Rule{
		TaskID:        1,
		Type:          recurrencedomain.TypeMonthlyParity,
		MonthlyParity: &recurrencedomain.MonthlyParityConfig{Parity: "weekly"},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for invalid parity value")
	}
}

func TestValidate_UnknownType(t *testing.T) {
	r := &recurrencedomain.Rule{TaskID: 1, Type: "hourly"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestValidate_ZeroTaskID(t *testing.T) {
	r := dailyRule(1)
	r.TaskID = 0
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for task_id=0")
	}
}

// ─── NextOccurrences: daily ───────────────────────────────────────────────────

func TestNextOccurrences_Daily_Every1Day(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := dailyRule(1).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-02T00:00:00Z"),
		mustTime("2026-01-03T00:00:00Z"),
		mustTime("2026-01-04T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

func TestNextOccurrences_Daily_Every7Days(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := dailyRule(7).NextOccurrences(from, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-08T00:00:00Z"),
		mustTime("2026-01-15T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// ─── NextOccurrences: monthly fixed day ──────────────────────────────────────

func TestNextOccurrences_FixedDay_Normal(t *testing.T) {
	from := mustTime("2026-01-05T00:00:00Z")
	got, err := fixedDayRule(10).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-10T00:00:00Z"),
		mustTime("2026-02-10T00:00:00Z"),
		mustTime("2026-03-10T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// Day 30 does not exist in February — that month must be skipped.
func TestNextOccurrences_FixedDay_SkipFebruary(t *testing.T) {
	from := mustTime("2026-01-31T00:00:00Z")
	got, err := fixedDayRule(30).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	// Feb has no day 30 → skip → March, April, May
	want := []time.Time{
		mustTime("2026-03-30T00:00:00Z"),
		mustTime("2026-04-30T00:00:00Z"),
		mustTime("2026-05-30T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// Leap year: Feb 2028 has 29 days, still no day 30.
func TestNextOccurrences_FixedDay_SkipFebruary_LeapYear(t *testing.T) {
	from := mustTime("2028-01-31T00:00:00Z")
	got, err := fixedDayRule(30).NextOccurrences(from, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2028-03-30T00:00:00Z"),
		mustTime("2028-04-30T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// ─── NextOccurrences: monthly custom dates ────────────────────────────────────

func TestNextOccurrences_Custom_MultipleMonths(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := customRule([]int{1, 15}).NextOccurrences(from, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-15T00:00:00Z"),
		mustTime("2026-02-01T00:00:00Z"),
		mustTime("2026-02-15T00:00:00Z"),
		mustTime("2026-03-01T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// Day 30 in February is skipped; the scheduler continues with other days.
func TestNextOccurrences_Custom_SkipInvalidDay(t *testing.T) {
	from := mustTime("2026-01-31T00:00:00Z")
	got, err := customRule([]int{15, 30}).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	// Feb: 15 ok, 30 skipped → Mar: 15 ok, 30 ok
	want := []time.Time{
		mustTime("2026-02-15T00:00:00Z"),
		mustTime("2026-03-15T00:00:00Z"),
		mustTime("2026-03-30T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// Input days need not be sorted; the scheduler normalises them.
func TestNextOccurrences_Custom_UnsortedInput(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := customRule([]int{28, 1, 15}).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-15T00:00:00Z"),
		mustTime("2026-01-28T00:00:00Z"),
		mustTime("2026-02-01T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// ─── NextOccurrences: monthly parity ─────────────────────────────────────────

func TestNextOccurrences_Parity_Even(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := parityRule(recurrencedomain.ParityEven).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-02T00:00:00Z"),
		mustTime("2026-01-04T00:00:00Z"),
		mustTime("2026-01-06T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

func TestNextOccurrences_Parity_Odd(t *testing.T) {
	from := mustTime("2026-01-01T00:00:00Z")
	got, err := parityRule(recurrencedomain.ParityOdd).NextOccurrences(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-01-03T00:00:00Z"),
		mustTime("2026-01-05T00:00:00Z"),
		mustTime("2026-01-07T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// Parity must roll over month boundaries correctly.
func TestNextOccurrences_Parity_RollOver(t *testing.T) {
	from := mustTime("2026-01-30T00:00:00Z")
	got, err := parityRule(recurrencedomain.ParityEven).NextOccurrences(from, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-02-02T00:00:00Z"),
		mustTime("2026-02-04T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// February parity: even days in a 28-day month end at 28.
func TestNextOccurrences_Parity_February28(t *testing.T) {
	from := mustTime("2026-02-01T00:00:00Z")
	got, err := parityRule(recurrencedomain.ParityEven).NextOccurrences(from, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		mustTime("2026-02-02T00:00:00Z"),
		mustTime("2026-02-04T00:00:00Z"),
		mustTime("2026-02-06T00:00:00Z"),
		mustTime("2026-02-08T00:00:00Z"),
		mustTime("2026-02-10T00:00:00Z"),
	}
	assertTimes(t, want, got)
}

// ─── NextOccurrences: error cases ────────────────────────────────────────────

func TestNextOccurrences_InvalidN(t *testing.T) {
	_, err := dailyRule(1).NextOccurrences(time.Now().UTC(), 0)
	if err == nil {
		t.Fatal("expected error for n=0")
	}
}

// ─── assert helper ────────────────────────────────────────────────────────────

func assertTimes(t *testing.T, want, got []time.Time) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d\n  want: %v\n  got:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if !want[i].Equal(got[i]) {
			t.Errorf("index %d: want %s, got %s", i, want[i].Format(time.RFC3339), got[i].Format(time.RFC3339))
		}
	}
}
