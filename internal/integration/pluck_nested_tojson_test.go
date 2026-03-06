//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"r-cli/internal/reql"
	"r-cli/internal/reql/parser"
)

func TestPluckNestedObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "address": map[string]interface{}{"city": "NYC", "zip": "10001"}},
		{"id": "u2", "name": "bob", "address": map[string]interface{}{"city": "LA", "zip": "90001"}},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").pluck("name", {address: ["city"]})`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := doc["name"]; !ok {
			t.Error("pluck result must contain 'name'")
		}
		addr, ok := doc["address"]
		if !ok {
			t.Fatal("pluck result must contain 'address'")
		}
		addrMap, ok := addr.(map[string]interface{})
		if !ok {
			t.Fatalf("address must be object, got %T", addr)
		}
		if _, ok := addrMap["city"]; !ok {
			t.Error("address must contain 'city'")
		}
		if _, ok := addrMap["zip"]; ok {
			t.Error("address must not contain 'zip' (not selected)")
		}
	}
}

func TestWithoutNestedObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "address": map[string]interface{}{"city": "NYC", "zip": "10001"}},
		{"id": "u2", "name": "bob", "address": map[string]interface{}{"city": "LA", "zip": "90001"}},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").without({address: {zip: true}})`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := doc["name"]; !ok {
			t.Error("without result must contain 'name'")
		}
		addr, ok := doc["address"]
		if !ok {
			t.Fatal("without result must still contain 'address' (only zip removed)")
		}
		addrMap, ok := addr.(map[string]interface{})
		if !ok {
			t.Fatalf("address must be object, got %T", addr)
		}
		if _, ok := addrMap["zip"]; ok {
			t.Error("address must not contain 'zip' (removed by without)")
		}
		if _, ok := addrMap["city"]; !ok {
			t.Error("address must still contain 'city'")
		}
	}
}

func TestHasFieldsNestedObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "profile": map[string]interface{}{"bio": "writer"}},
		{"id": "u2", "name": "bob"},
		{"id": "u3", "name": "carol", "profile": map[string]interface{}{"bio": "artist"}},
	})

	// hasFields({profile: true}) returns docs that have a 'profile' field
	expr := fmt.Sprintf(`r.db("%s").table("docs").hasFields({profile: true})`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows with profile field, want 2", len(rows))
	}
}

func TestWithFieldsNestedObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "u1", "name": "alice", "stats": map[string]interface{}{"score": 10}},
		{"id": "u2", "name": "bob"},
		{"id": "u3", "name": "carol", "stats": map[string]interface{}{"score": 20}},
	})

	// withFields("id", {stats: true}) returns docs having 'id' and 'stats', plucking only those fields
	expr := fmt.Sprintf(`r.db("%s").table("docs").withFields("id", {stats: true})`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// only u1 and u3 have 'stats' field
	if len(rows) != 2 {
		t.Errorf("got %d rows with stats field, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := doc["name"]; ok {
			t.Error("withFields result must not contain 'name' (not selected)")
		}
		if _, ok := doc["stats"]; !ok {
			t.Error("withFields result must contain 'stats'")
		}
	}
}

func TestToJSONAlias(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	// toJSON() is an alias for toJSONString()
	term := reql.Datum(map[string]interface{}{"a": 1}).ToJSONString()
	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run toJSONString: %v", err)
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

	// verify via parser toJSON()
	exprToJSON := `r.expr({a: 1}).toJSON()`
	termParsed, err := parser.Parse(exprToJSON)
	if err != nil {
		t.Fatalf("parse toJSON: %v", err)
	}
	_, cur2, err := exec.Run(ctx, termParsed, nil)
	if err != nil {
		t.Fatalf("run toJSON: %v", err)
	}
	defer closeCursor(cur2)
	raw2, err := cur2.Next()
	if err != nil {
		t.Fatalf("next2: %v", err)
	}
	var got2 string
	if err := json.Unmarshal(raw2, &got2); err != nil {
		t.Fatalf("unmarshal2: %v", err)
	}

	// both must be valid JSON strings containing {"a":1}
	var obj1, obj2 map[string]interface{}
	if err := json.Unmarshal([]byte(got), &obj1); err != nil {
		t.Fatalf("toJSONString result not valid JSON: %q", got)
	}
	if err := json.Unmarshal([]byte(got2), &obj2); err != nil {
		t.Fatalf("toJSON result not valid JSON: %q", got2)
	}
	if obj1["a"] != obj2["a"] {
		t.Errorf("toJSON result %q differs from toJSONString %q", got2, got)
	}
}

func TestToJsonStringAlias(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	expr := `r.expr({b: 2}).toJsonString()`
	termParsed, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse toJsonString: %v", err)
	}
	_, cur, err := exec.Run(ctx, termParsed, nil)
	if err != nil {
		t.Fatalf("run toJsonString: %v", err)
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
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(got), &obj); err != nil {
		t.Fatalf("toJsonString result not valid JSON: %q", got)
	}
	if obj["b"] != float64(2) {
		t.Errorf("toJsonString result[b]=%v, want 2", obj["b"])
	}
}
