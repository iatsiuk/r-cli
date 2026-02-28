//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestTypeOf(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		term  reql.Term
		want  string
	}{
		{"number", reql.Datum(1).TypeOf(), "NUMBER"},
		{"string", reql.Datum("s").TypeOf(), "STRING"},
		{"array", reql.Array().TypeOf(), "ARRAY"},
		{"null", reql.Datum(nil).TypeOf(), "NULL"},
		{"bool", reql.Datum(true).TypeOf(), "BOOL"},
		{"object", reql.Datum(map[string]interface{}{"a": 1}).TypeOf(), "OBJECT"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, cur, err := exec.Run(ctx, tc.term, nil)
			if err != nil {
				t.Fatalf("typeOf: %v", err)
			}
			defer closeCursor(cur)

			raw, err := cur.Next()
			if err != nil {
				t.Fatalf("cursor next: %v", err)
			}
			var got string
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != tc.want {
				t.Errorf("typeOf=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestCoerceToNumberToString(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum(42).CoerceTo("STRING"), nil)
	if err != nil {
		t.Fatalf("coerceTo string: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != "42" {
		t.Errorf("coerceTo string=%q, want %q", got, "42")
	}
}

func TestCoerceToStringToNumber(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	_, cur, err := exec.Run(ctx, reql.Datum("123").CoerceTo("NUMBER"), nil)
	if err != nil {
		t.Fatalf("coerceTo number: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 123 {
		t.Errorf("coerceTo number=%v, want 123", got)
	}
}

func TestCoerceToInvalidReturnsError(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// coercing a non-numeric string to NUMBER is a runtime error
	_, cur, err := exec.Run(ctx, reql.Datum("abc").CoerceTo("NUMBER"), nil)
	closeCursor(cur)
	if err == nil {
		t.Fatal("expected error for invalid coerceTo, got nil")
	}
	var rErr *response.ReqlRuntimeError
	if !errors.As(err, &rErr) {
		t.Errorf("expected ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestCoerceToOnTableField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "typed")
	seedTable(t, exec, dbName, "typed", []map[string]interface{}{
		{"id": "a", "score": 7},
		{"id": "b", "score": 42},
	})

	// map: coerce score field to STRING
	mapFn := reql.Func(reql.Var(1).GetField("score").CoerceTo("STRING"), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("typed").Map(mapFn), nil)
	if err != nil {
		t.Fatalf("map coerceTo: %v", err)
	}
	rows, err := cur.All()
	closeCursor(cur)
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// verify each row is a valid numeric string
	seen := map[string]bool{}
	for i, raw := range rows {
		var got string
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("row %d unmarshal: %v", i, err)
		}
		seen[got] = true
	}
	for _, want := range []string{"7", "42"} {
		if !seen[want] {
			t.Errorf("missing coerceTo result %q in %v", want, seen)
		}
	}
}
