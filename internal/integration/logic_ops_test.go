//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestNe(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		a, b interface{}
		want bool
	}{
		{1, 2, true},
		{1, 1, false},
		{"a", "b", true},
		{"a", "a", false},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.a).Ne(tc.b), nil)
		if err != nil {
			t.Fatalf("ne(%v,%v): %v", tc.a, tc.b, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("ne(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLt(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		a, b interface{}
		want bool
	}{
		{1, 2, true},
		{2, 1, false},
		{1, 1, false},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.a).Lt(tc.b), nil)
		if err != nil {
			t.Fatalf("lt(%v,%v): %v", tc.a, tc.b, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("lt(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestLe(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		a, b interface{}
		want bool
	}{
		{1, 1, true},
		{2, 1, false},
		{0, 1, true},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.a).Le(tc.b), nil)
		if err != nil {
			t.Fatalf("le(%v,%v): %v", tc.a, tc.b, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("le(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestGe(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		a, b interface{}
		want bool
	}{
		{2, 2, true},
		{1, 2, false},
		{3, 2, true},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.a).Ge(tc.b), nil)
		if err != nil {
			t.Fatalf("ge(%v,%v): %v", tc.a, tc.b, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("ge(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestOr(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		a, b interface{}
		want bool
	}{
		{false, true, true},
		{false, false, false},
		{true, false, true},
		{true, true, true},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.a).Or(reql.Datum(tc.b)), nil)
		if err != nil {
			t.Fatalf("or(%v,%v): %v", tc.a, tc.b, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("or(%v,%v)=%v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNot(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		input interface{}
		want  bool
	}{
		{true, false},
		{false, true},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.input).Not(), nil)
		if err != nil {
			t.Fatalf("not(%v): %v", tc.input, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var got bool
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got != tc.want {
			t.Errorf("not(%v)=%v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestFilterNe(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "1", "status": "active"},
		{"id": "2", "status": "inactive"},
		{"id": "3", "status": "active"},
		{"id": "4", "status": "pending"},
	})

	// filter where status != "active" -> 2 rows
	pred := reql.Row().GetField("status").Ne("active")
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("items").Filter(pred), nil)
	if err != nil {
		t.Fatalf("filter ne: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for status!=active, want 2", len(rows))
	}
}

func TestFilterLtLe(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "scores")
	seedTable(t, exec, dbName, "scores", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 20},
		{"id": "3", "score": 30},
		{"id": "4", "score": 40},
	})

	// lt: score < 30 -> 2 rows (10, 20)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("scores").Filter(
		reql.Row().GetField("score").Lt(30),
	), nil)
	if err != nil {
		t.Fatalf("filter lt: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("lt: got %d rows for score<30, want 2", len(rows))
	}

	// le: score <= 30 -> 3 rows (10, 20, 30)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("scores").Filter(
		reql.Row().GetField("score").Le(30),
	), nil)
	if err != nil {
		t.Fatalf("filter le: %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows2) != 3 {
		t.Errorf("le: got %d rows for score<=30, want 3", len(rows2))
	}
}

func TestFilterGeOr(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "vals")
	seedTable(t, exec, dbName, "vals", []map[string]interface{}{
		{"id": "1", "v": 5},
		{"id": "2", "v": 10},
		{"id": "3", "v": 15},
		{"id": "4", "v": 20},
	})

	// ge: v >= 15 -> 2 rows (15, 20)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("vals").Filter(
		reql.Row().GetField("v").Ge(15),
	), nil)
	if err != nil {
		t.Fatalf("filter ge: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("ge: got %d rows for v>=15, want 2", len(rows))
	}

	// or: v <= 5 OR v >= 20 -> 2 rows (5, 20)
	pred := reql.Row().GetField("v").Le(5).Or(reql.Row().GetField("v").Ge(20))
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("vals").Filter(pred), nil)
	if err != nil {
		t.Fatalf("filter or: %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows2) != 2 {
		t.Errorf("or: got %d rows for v<=5 OR v>=20, want 2", len(rows2))
	}
}
