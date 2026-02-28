//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"r-cli/internal/reql/parser"
)

func TestTimeParser_4ArgForm(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	term, err := parser.Parse(`r.time(2024, 1, 15, "+00:00").toISO8601()`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.HasPrefix(got, "2024-01-15") {
		t.Errorf("r.time(2024,1,15,+00:00).toISO8601()=%q, want prefix 2024-01-15", got)
	}
}

func TestTimeParser_7ArgForm(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	term, err := parser.Parse(`r.time(2024, 1, 15, 10, 30, 0, "+00:00").toISO8601()`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.HasPrefix(got, "2024-01-15T10:30:00") {
		t.Errorf("r.time(7-arg).toISO8601()=%q, want prefix 2024-01-15T10:30:00", got)
	}
}

func TestTimeParser_BinaryRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// passing "hello" as string argument; server stores its bytes as binary
	// and returns base64("hello") = "aGVsbG8=" in the BINARY pseudo-type
	term, err := parser.Parse(`r.binary("hello")`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	// server returns BINARY pseudo-type: {"$reql_type$":"BINARY","data":"aGVsbG8="}
	var got struct {
		ReqlType string `json:"$reql_type$"`
		Data     string `json:"data"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ReqlType != "BINARY" {
		t.Errorf("r.binary $reql_type$=%q, want BINARY", got.ReqlType)
	}
	if got.Data != "aGVsbG8=" {
		t.Errorf("r.binary data=%q, want aGVsbG8= (base64 of 'hello')", got.Data)
	}
}
