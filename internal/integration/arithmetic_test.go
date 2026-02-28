//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestSub(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Sub(3), nil)
	if err != nil {
		t.Fatalf("sub: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v != 7 {
		t.Errorf("10-3=%v, want 7", v)
	}
}

func TestSubChained(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Sub(3).Sub(2), nil)
	if err != nil {
		t.Fatalf("sub chained: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v != 5 {
		t.Errorf("10-3-2=%v, want 5", v)
	}
}

func TestDiv(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Div(4), nil)
	if err != nil {
		t.Fatalf("div: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v != 2.5 {
		t.Errorf("10/4=%v, want 2.5", v)
	}
}

func TestDivByZero(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Div(0), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for division by zero, got nil")
	}
	var rErr *response.ReqlRuntimeError
	if !errors.As(err, &rErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestMod(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Mod(3), nil)
	if err != nil {
		t.Fatalf("mod: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v != 1 {
		t.Errorf("10%%3=%v, want 1", v)
	}
}

func TestModByZero(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(10).Mod(0), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for mod by zero, got nil")
	}
	var rErr *response.ReqlRuntimeError
	if !errors.As(err, &rErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestFloor(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		input float64
		want  float64
	}{
		{3.7, 3},
		{-2.3, -3},
		{5.0, 5},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.input).Floor(), nil)
		if err != nil {
			t.Fatalf("floor(%v): %v", tc.input, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if v != tc.want {
			t.Errorf("floor(%v)=%v, want %v", tc.input, v, tc.want)
		}
	}
}

func TestCeil(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		input float64
		want  float64
	}{
		{3.2, 4},
		{-2.7, -2},
		{5.0, 5},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.input).Ceil(), nil)
		if err != nil {
			t.Fatalf("ceil(%v): %v", tc.input, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if v != tc.want {
			t.Errorf("ceil(%v)=%v, want %v", tc.input, v, tc.want)
		}
	}
}

func TestRound(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		input float64
		want  float64
	}{
		{3.5, 4},
		{3.4, 3},
		{-3.4, -3},
	}
	for _, tc := range tests {
		_, cur, err := exec.Run(ctx, reql.Datum(tc.input).Round(), nil)
		if err != nil {
			t.Fatalf("round(%v): %v", tc.input, err)
		}
		raw, err := cur.Next()
		closeCursor(cur)
		if err != nil {
			t.Fatalf("cursor next: %v", err)
		}
		var v float64
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if v != tc.want {
			t.Errorf("round(%v)=%v, want %v", tc.input, v, tc.want)
		}
	}
}

func TestArithmeticOnTableFields(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "nums")
	seedTable(t, exec, dbName, "nums", []map[string]interface{}{
		{"id": "a", "val": 10.7},
		{"id": "b", "val": -3.2},
		{"id": "c", "val": 9.0},
	})

	// map: floor(val - 1)
	// a=10.7: floor(9.7)=9, b=-3.2: floor(-4.2)=-5, c=9.0: floor(8.0)=8 => sorted: [-5,8,9]
	mapFn := reql.Func(reql.Var(1).GetField("val").Sub(1).Floor(), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("nums").Map(mapFn), nil)
	if err != nil {
		t.Fatalf("map sub+floor: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	floorVals := make([]float64, 3)
	for i, r := range rows {
		if err := json.Unmarshal(r, &floorVals[i]); err != nil {
			t.Fatalf("unmarshal floor row %d: %v", i, err)
		}
	}
	sort.Float64s(floorVals)
	for i, want := range []float64{-5, 8, 9} {
		if floorVals[i] != want {
			t.Errorf("floor[%d]=%v, want %v", i, floorVals[i], want)
		}
	}

	// map: ceil(val / 2)
	// a=10.7: ceil(5.35)=6, b=-3.2: ceil(-1.6)=-1, c=9.0: ceil(4.5)=5 => sorted: [-1,5,6]
	mapFn2 := reql.Func(reql.Var(1).GetField("val").Div(2).Ceil(), 1)
	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("nums").Map(mapFn2), nil)
	if err != nil {
		t.Fatalf("map div+ceil: %v", err)
	}
	rows2, err := cur2.All()
	closeCursor(cur2)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows2) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows2))
	}
	ceilVals := make([]float64, 3)
	for i, r := range rows2 {
		if err := json.Unmarshal(r, &ceilVals[i]); err != nil {
			t.Fatalf("unmarshal ceil row %d: %v", i, err)
		}
	}
	sort.Float64s(ceilVals)
	for i, want := range []float64{-1, 5, 6} {
		if ceilVals[i] != want {
			t.Errorf("ceil[%d]=%v, want %v", i, ceilVals[i], want)
		}
	}

	// map: round(val)
	// a=10.7: round=11, b=-3.2: round=-3, c=9.0: round=9 => sorted: [-3,9,11]
	mapFn3 := reql.Func(reql.Var(1).GetField("val").Round(), 1)
	_, cur3, err := exec.Run(ctx, reql.DB(dbName).Table("nums").Map(mapFn3), nil)
	if err != nil {
		t.Fatalf("map round: %v", err)
	}
	rows3, err := cur3.All()
	closeCursor(cur3)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows3) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows3))
	}
	roundVals := make([]float64, 3)
	for i, r := range rows3 {
		if err := json.Unmarshal(r, &roundVals[i]); err != nil {
			t.Fatalf("unmarshal round row %d: %v", i, err)
		}
	}
	sort.Float64s(roundVals)
	for i, want := range []float64{-3, 9, 11} {
		if roundVals[i] != want {
			t.Errorf("round[%d]=%v, want %v", i, roundVals[i], want)
		}
	}

	// map: val mod 3 (only positive integer values make sense for integer mod)
	seedTable(t, exec, dbName, "nums", []map[string]interface{}{
		{"id": "d", "val": 7.0},
	})
	mapFn4 := reql.Func(reql.Var(1).GetField("val").Mod(3), 1)
	_, cur4, err := exec.Run(ctx, reql.DB(dbName).Table("nums").Filter(
		reql.Func(reql.Var(1).GetField("id").Eq("d"), 1),
	).Map(mapFn4), nil)
	if err != nil {
		t.Fatalf("map mod: %v", err)
	}
	rows4, err := cur4.All()
	closeCursor(cur4)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows4) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows4))
	}
	var modResult float64
	if err := json.Unmarshal(rows4[0], &modResult); err != nil {
		t.Fatalf("unmarshal mod result: %v", err)
	}
	if modResult != 1 {
		t.Errorf("7 mod 3=%v, want 1", modResult)
	}
}
