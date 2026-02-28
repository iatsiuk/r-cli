//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
)

func TestObjectConstructor(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Object("a", 1, "b", 2), nil)
	if err != nil {
		t.Fatalf("Object: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("r.object got %d keys, want 2", len(got))
	}
	if got["a"] != float64(1) || got["b"] != float64(2) {
		t.Errorf("r.object got %v, want {a:1, b:2}", got)
	}
}

func TestRangeConstructor(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// r.range(5) returns [0, 1, 2, 3, 4]
	_, cur, err := exec.Run(ctx, reql.Range(5), nil)
	if err != nil {
		t.Fatalf("Range(5): %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 5 {
		t.Fatalf("r.range(5) got %d rows, want 5", len(rows))
	}
	for i, raw := range rows {
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		if int(v) != i {
			t.Errorf("r.range(5)[%d]=%v, want %d", i, v, i)
		}
	}

	// r.range(2, 5) returns [2, 3, 4]
	_, cur2, err := exec.Run(ctx, reql.Range(2, 5), nil)
	if err != nil {
		t.Fatalf("Range(2, 5): %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows2) != 3 {
		t.Fatalf("r.range(2, 5) got %d rows, want 3", len(rows2))
	}
	for i, raw := range rows2 {
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		want := i + 2
		if int(v) != want {
			t.Errorf("r.range(2, 5)[%d]=%v, want %d", i, v, want)
		}
	}
}

func TestRandomConstructor(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// r.random() returns a float in [0, 1)
	_, cur, err := exec.Run(ctx, reql.Random(), nil)
	if err != nil {
		t.Fatalf("Random(): %v", err)
	}
	raw, err := cur.Next()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v < 0 || v >= 1 {
		t.Errorf("r.random()=%v, want in [0, 1)", v)
	}

	// r.random(10) returns a number in [0, 10)
	_, cur2, err := exec.Run(ctx, reql.Random(10), nil)
	if err != nil {
		t.Fatalf("Random(10): %v", err)
	}
	raw2, err := cur2.Next()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var v2 float64
	if err := json.Unmarshal(raw2, &v2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v2 < 0 || v2 >= 10 {
		t.Errorf("r.random(10)=%v, want in [0, 10)", v2)
	}
}
