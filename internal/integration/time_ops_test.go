//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"r-cli/internal/reql"
)

func TestToISO8601(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 00:00:00 UTC
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).ToISO8601(), nil)
	if err != nil {
		t.Fatalf("toISO8601: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(got, "2024-01-01") {
		t.Errorf("toISO8601=%q does not contain 2024-01-01", got)
	}
}

func TestInTimezone(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 00:00:00 UTC; in +05:00 the hour is 5
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).InTimezone("+05:00").Hours(), nil)
	if err != nil {
		t.Fatalf("inTimezone.hours: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 5 {
		t.Errorf("inTimezone(+05:00).hours=%v, want 5", got)
	}
}

func TestTimezone(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).InTimezone("+03:00").Timezone(), nil)
	if err != nil {
		t.Fatalf("timezone: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != "+03:00" {
		t.Errorf("timezone=%q, want +03:00", got)
	}
}

func TestDate(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704103200 = 2024-01-01 10:00:00 UTC; date() truncates to midnight = 1704067200
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704103200).Date().ToEpochTime(), nil)
	if err != nil {
		t.Fatalf("date: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1704067200 {
		t.Errorf("date().toEpochTime=%v, want 1704067200", got)
	}
}

func TestTimeOfDay(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704103200 = 2024-01-01 10:00:00 UTC; timeOfDay = 10*3600 = 36000 seconds
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704103200).TimeOfDay(), nil)
	if err != nil {
		t.Fatalf("timeOfDay: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 36000 {
		t.Errorf("timeOfDay=%v, want 36000", got)
	}
}

func TestMonth(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 -> month 1
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).Month(), nil)
	if err != nil {
		t.Fatalf("month: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1 {
		t.Errorf("month=%v, want 1", got)
	}
}

func TestDay(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 -> day 1
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).Day(), nil)
	if err != nil {
		t.Fatalf("day: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1 {
		t.Errorf("day=%v, want 1", got)
	}
}

func TestDayOfWeek(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 = Monday -> dayOfWeek 1
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).DayOfWeek(), nil)
	if err != nil {
		t.Fatalf("dayOfWeek: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1 {
		t.Errorf("dayOfWeek=%v, want 1 (Monday)", got)
	}
}

func TestDayOfYear(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1704067200 = 2024-01-01 -> dayOfYear 1
	_, cur, err := exec.Run(ctx, reql.EpochTime(1704067200).DayOfYear(), nil)
	if err != nil {
		t.Fatalf("dayOfYear: %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1 {
		t.Errorf("dayOfYear=%v, want 1", got)
	}
}

func TestHoursMinutesSeconds(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// 1705329045 = 2024-01-15 14:30:45 UTC
	ts := reql.EpochTime(1705329045)

	checks := []struct {
		name string
		term reql.Term
		want float64
	}{
		{"hours", ts.Hours(), 14},
		{"minutes", ts.Minutes(), 30},
		{"seconds", ts.Seconds(), 45},
	}

	for _, c := range checks {
		_, cur, err := exec.Run(ctx, c.term, nil)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("%s cursor next: %v", c.name, err)
		}
		var got float64
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("%s unmarshal: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s=%v, want %v", c.name, got, c.want)
		}
	}
}

func TestDuringBasicFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")
	// seed: 3 events at known epoch timestamps
	// e1: 2023-01-01 = 1672531200
	// e2: 2023-06-01 = 1685577600
	// e3: 2024-01-01 = 1704067200
	seedTable(t, exec, dbName, "events", []map[string]interface{}{
		{"id": "e1", "ts": reql.EpochTime(1672531200)},
		{"id": "e2", "ts": reql.EpochTime(1685577600)},
		{"id": "e3", "ts": reql.EpochTime(1704067200)},
	})

	// during [2023-02-01, 2023-12-31) -- only e2 falls in this range
	start := reql.EpochTime(1675209600) // 2023-02-01
	end := reql.EpochTime(1703980800)   // 2023-12-31
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("events").Filter(
			reql.Func(reql.Var(1).GetField("ts").During(start, end), 1),
		), nil)
	if err != nil {
		t.Fatalf("filter during: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("during filter got %d rows, want 1", len(rows))
	}
	var doc struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc.ID != "e2" {
		t.Errorf("during filter returned id=%q, want e2", doc.ID)
	}
}

func TestDuringBoundaryBehavior(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")
	// e1 is exactly at start, e2 is in middle, e3 is exactly at end
	seedTable(t, exec, dbName, "events", []map[string]interface{}{
		{"id": "e1", "ts": reql.EpochTime(1672531200)}, // exactly at start
		{"id": "e2", "ts": reql.EpochTime(1685577600)}, // in range
		{"id": "e3", "ts": reql.EpochTime(1704067200)}, // exactly at end
	})

	// default bounds: left=closed, right=open => [start, end)
	// e1 at start is included, e3 at end is excluded
	start := reql.EpochTime(1672531200)
	end := reql.EpochTime(1704067200)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("events").Filter(
			reql.Func(reql.Var(1).GetField("ts").During(start, end), 1),
		), nil)
	if err != nil {
		t.Fatalf("during boundary: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// [start, end) includes e1 and e2 but not e3
	if len(rows) != 2 {
		t.Fatalf("during [start, end) got %d rows, want 2", len(rows))
	}
	ids := make(map[string]bool)
	for _, r := range rows {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(r, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		ids[doc.ID] = true
	}
	if !ids["e1"] || !ids["e2"] || ids["e3"] {
		t.Errorf("during [start, end) returned ids=%v, want e1 and e2 only", ids)
	}
}
