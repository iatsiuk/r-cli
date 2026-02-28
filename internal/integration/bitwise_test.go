//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/reql/parser"
)

func TestBitwiseOps(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		name string
		term reql.Term
		want float64
	}{
		{"bitAnd_5_3", reql.Datum(5).BitAnd(3), 1},
		{"bitOr_5_3", reql.Datum(5).BitOr(3), 7},
		{"bitXor_5_3", reql.Datum(5).BitXor(3), 6},
		{"bitNot_7", reql.Datum(7).BitNot(), -8},
		{"bitSal_5_2", reql.Datum(5).BitSal(2), 20},
		{"bitSar_20_2", reql.Datum(20).BitSar(2), 5},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, cur, err := exec.Run(ctx, tc.term, nil)
			if err != nil {
				t.Fatalf("%s: run: %v", tc.name, err)
			}
			raw, err := cur.Next()
			closeCursor(cur)
			if err != nil {
				t.Fatalf("%s: next: %v", tc.name, err)
			}
			var got float64
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("%s: unmarshal: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestBitwiseParser(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	tests := []struct {
		name  string
		query string
		want  float64
	}{
		{"bitAnd", `r.expr(5).bitAnd(3)`, 1},
		{"bitOr", `r.expr(5).bitOr(3)`, 7},
		{"bitXor", `r.expr(5).bitXor(3)`, 6},
		{"bitNot", `r.expr(7).bitNot()`, -8},
		{"bitSal", `r.expr(5).bitSal(2)`, 20},
		{"bitSar", `r.expr(20).bitSar(2)`, 5},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			term, err := parser.Parse(tc.query)
			if err != nil {
				t.Fatalf("parse %q: %v", tc.query, err)
			}
			_, cur, err := exec.Run(ctx, term, nil)
			if err != nil {
				t.Fatalf("%s: run: %v", tc.name, err)
			}
			raw, err := cur.Next()
			closeCursor(cur)
			if err != nil {
				t.Fatalf("%s: next: %v", tc.name, err)
			}
			var got float64
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("%s: unmarshal: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
