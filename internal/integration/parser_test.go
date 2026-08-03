//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"r-cli/internal/query"
	"r-cli/internal/reql"
	"r-cli/internal/reql/parser"
)

func TestParserFixesBracketNumericIndex(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "1", "val": 10},
		{"id": "2", "val": 20},
		{"id": "3", "val": 30},
	})

	expr := fmt.Sprintf(`r.db("%s").table("items").orderBy("id").limit(1)(0)`, dbName)
	term, err := parser.Parse(expr)
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
		t.Fatalf("cursor next: %v", err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["id"] != "1" {
		t.Errorf("got id=%v, want 1", doc["id"])
	}
}

func TestParserFixesSample(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	docs := make([]map[string]interface{}, 10)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": fmt.Sprintf("%d", i+1), "val": i + 1}
	}
	seedTable(t, exec, dbName, "items", docs)

	expr := fmt.Sprintf(`r.db("%s").table("items").sample(3)`, dbName)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err := exec.Run(ctx, term, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	defer closeCursor(cur)

	// sample returns SUCCESS_ATOM with an array value; cur.Next() gives the whole array
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var sampled []json.RawMessage
	if err := json.Unmarshal(raw, &sampled); err != nil {
		t.Fatalf("unmarshal sample array: %v", err)
	}
	if len(sampled) != 3 {
		t.Errorf("got %d docs in sample, want 3", len(sampled))
	}
}

func TestParserFixesNestedFunction(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// seed using parser so nested arrays produce correct MAKE_ARRAY ReQL terms
	insertExpr := fmt.Sprintf(
		`r.db("%s").table("docs").insert([{id: "1", items: [{type: "a"}, {type: "b"}]}, {id: "2", items: [{type: "a"}, {type: "c"}]}])`,
		dbName,
	)
	insertTerm, err := parser.Parse(insertExpr)
	if err != nil {
		t.Fatalf("parse insert: %v", err)
	}
	_, cur, err := exec.Run(ctx, insertTerm, nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert docs: %v", err)
	}

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").map(function(doc){ return doc("items").filter(function(i){ return i("type").eq("a") }) })`,
		dbName,
	)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err = exec.Run(ctx, term, nil)
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
		var items []map[string]interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("unmarshal items: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 filtered item, got %d: %s", len(items), string(raw))
			continue
		}
		if items[0]["type"] != "a" {
			t.Errorf("expected type=a, got %v", items[0]["type"])
		}
	}
}

func TestParserFixesNestedArrow(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// seed using parser so nested arrays produce correct MAKE_ARRAY ReQL terms
	insertExpr := fmt.Sprintf(
		`r.db("%s").table("docs").insert([{id: "1", items: [{type: "a"}, {type: "b"}]}, {id: "2", items: [{type: "a"}, {type: "c"}]}])`,
		dbName,
	)
	insertTerm, err := parser.Parse(insertExpr)
	if err != nil {
		t.Fatalf("parse insert: %v", err)
	}
	_, cur, err := exec.Run(ctx, insertTerm, nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert docs: %v", err)
	}

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").map((doc) => doc("items").filter((i) => i("type").eq("a")))`,
		dbName,
	)
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	_, cur, err = exec.Run(ctx, term, nil)
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
		var items []map[string]interface{}
		if err := json.Unmarshal(raw, &items); err != nil {
			t.Fatalf("unmarshal items: %v", err)
		}
		if len(items) != 1 {
			t.Errorf("expected 1 filtered item, got %d: %s", len(items), string(raw))
			continue
		}
		if items[0]["type"] != "a" {
			t.Errorf("expected type=a, got %v", items[0]["type"])
		}
	}
}

func TestParserFixesInsertOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")

	expr := fmt.Sprintf(`r.db("%s").table("items").insert({id: "new", val: 1}, {return_changes: true})`, dbName)
	term, err := parser.Parse(expr)
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
		t.Fatalf("cursor next: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["inserted"] != float64(1) {
		t.Errorf("inserted=%v, want 1", result["inserted"])
	}
	changes, ok := result["changes"].([]interface{})
	if !ok {
		t.Fatalf("changes field missing or not array: %v", result)
	}
	if len(changes) != 1 {
		t.Errorf("changes has %d entries, want 1", len(changes))
	}
	change, ok := changes[0].(map[string]interface{})
	if !ok {
		t.Fatalf("changes[0] is not object: %v", changes[0])
	}
	if change["new_val"] == nil {
		t.Error("changes[0].new_val is nil")
	}
}

func TestParserFixesArrowParenObject(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "people")
	seedTable(t, exec, dbName, "people", []map[string]interface{}{
		{"id": "1", "first": "Alice", "last": "Smith"},
	})

	expr := fmt.Sprintf(
		`r.db("%s").table("people").map(row => ({full: row("first").add(" ").add(row("last"))}))`,
		dbName,
	)
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
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(rows[0], &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if doc["full"] != "Alice Smith" {
		t.Errorf("full=%v, want 'Alice Smith'", doc["full"])
	}
}

// ungroupedCounts runs expr and reads the array returned by group(...).count().ungroup()
// into a map keyed by the JSON encoding of the group key.
func ungroupedCounts(t *testing.T, exec *query.Executor, expr string) map[string]float64 {
	t.Helper()
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	_, cur, err := exec.Run(context.Background(), term, nil)
	if err != nil {
		t.Fatalf("run %q: %v", expr, err)
	}
	defer closeCursor(cur)
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var rows []struct {
		Group     json.RawMessage `json:"group"`
		Reduction float64         `json:"reduction"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("unmarshal ungroup rows: %v", err)
	}
	counts := make(map[string]float64, len(rows))
	for _, row := range rows {
		counts[string(row.Group)] = row.Reduction
	}
	return counts
}

func TestParserGroupMultipleKeys(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng", "city": "berlin"},
		{"id": "2", "dept": "eng", "city": "berlin"},
		{"id": "3", "dept": "eng", "city": "lisbon"},
		{"id": "4", "dept": "hr", "city": "lisbon"},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").group("dept","city").count().ungroup()`, dbName)
	counts := ungroupedCounts(t, exec, expr)
	want := map[string]float64{
		`["eng","berlin"]`: 2,
		`["eng","lisbon"]`: 1,
		`["hr","lisbon"]`:  1,
	}
	if len(counts) != len(want) {
		t.Fatalf("got %d groups (%v), want %d", len(counts), counts, len(want))
	}
	for key, n := range want {
		if counts[key] != n {
			t.Errorf("group %s count=%v, want %v", key, counts[key], n)
		}
	}
}

func TestParserGroupLambdaMatchesRow(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng"},
		{"id": "2", "dept": "eng"},
		{"id": "3", "dept": "hr"},
	})

	lambda := ungroupedCounts(t, exec, fmt.Sprintf(
		`r.db("%s").table("docs").group(d => d("dept")).count().ungroup()`, dbName))
	row := ungroupedCounts(t, exec, fmt.Sprintf(
		`r.db("%s").table("docs").group(r.row("dept")).count().ungroup()`, dbName))

	if !reflect.DeepEqual(lambda, row) {
		t.Fatalf("lambda group %v differs from r.row group %v", lambda, row)
	}
	if lambda[`"eng"`] != 2 || lambda[`"hr"`] != 1 {
		t.Errorf("group counts = %v, want eng=2 hr=1", lambda)
	}
}

func TestParserGroupWithIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng", "city": "berlin"},
		{"id": "2", "dept": "eng", "city": "lisbon"},
		{"id": "3", "dept": "hr", "city": "lisbon"},
	})
	waitForIndex(t, exec, dbName, "docs", "dept")

	// index optarg supplies the group key; a positional key adds a second one
	expr := fmt.Sprintf(`r.db("%s").table("docs").group("city",{index:"dept"}).count().ungroup()`, dbName)
	counts := ungroupedCounts(t, exec, expr)
	want := map[string]float64{
		`["berlin","eng"]`: 1,
		`["lisbon","eng"]`: 1,
		`["lisbon","hr"]`:  1,
	}
	if len(counts) != len(want) {
		t.Fatalf("got %d groups (%v), want %d", len(counts), counts, len(want))
	}
	for key, n := range want {
		if counts[key] != n {
			t.Errorf("group %s count=%v, want %v", key, counts[key], n)
		}
	}
}

// parseRunAtom runs expr and returns its single atom result.
func parseRunAtom(t *testing.T, exec *query.Executor, expr string) json.RawMessage {
	t.Helper()
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	_, cur, err := exec.Run(context.Background(), term, nil)
	if err != nil {
		t.Fatalf("run %q: %v", expr, err)
	}
	defer closeCursor(cur)
	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	return raw
}

func TestParserAggregateNoArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "1", "val": 10},
		{"id": "2", "val": 30},
		{"id": "3", "val": 20},
	})

	cases := []struct {
		name string
		expr string
		want float64
	}{
		{"min", fmt.Sprintf(`r.db("%s").table("items").map(function(d){ return d("val") }).min()`, dbName), 10},
		{"max", fmt.Sprintf(`r.db("%s").table("items").map(function(d){ return d("val") }).max()`, dbName), 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got float64
			if err := json.Unmarshal(parseRunAtom(t, exec, tc.expr), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != tc.want {
				t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestParserAggregateWithIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "items")
	seedTable(t, exec, dbName, "items", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 30},
		{"id": "3", "score": 20},
	})
	waitForIndex(t, exec, dbName, "items", "score")

	cases := []struct {
		name string
		expr string
		want string
	}{
		{"min", fmt.Sprintf(`r.db("%s").table("items").min({index:"score"})`, dbName), "1"},
		{"max", fmt.Sprintf(`r.db("%s").table("items").max({index:"score"})`, dbName), "2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var doc map[string]interface{}
			if err := json.Unmarshal(parseRunAtom(t, exec, tc.expr), &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if doc["id"] != tc.want {
				t.Errorf("%s returned id=%v, want %v (full doc %v)", tc.name, doc["id"], tc.want, doc)
			}
		})
	}
}

func TestParserAggregateNestedFieldLambda(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "accounts")
	seedTable(t, exec, dbName, "accounts", []map[string]interface{}{
		{"id": "1", "balance": map[string]interface{}{"amount": 5}},
		{"id": "2", "balance": map[string]interface{}{"amount": 7}},
		{"id": "3", "balance": map[string]interface{}{"amount": 13}},
	})

	expr := fmt.Sprintf(`r.db("%s").table("accounts").sum(x => x("balance")("amount"))`, dbName)
	var got float64
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 25 {
		t.Errorf("sum = %v, want 25", got)
	}
}

func TestParserFieldProbingIdiom(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice", "city": "berlin"},
		{"id": "2", "name": "bob"},
		{"id": "3", "city": "lisbon"},
	})

	// lambda parameter used as a hasFields selector, one count per probed field name
	expr := fmt.Sprintf(
		`r.expr(["name","city","zip"]).map(function(f){ return [f, r.db("%s").table("docs").filter(function(o){ return o.hasFields(f) }).count()] })`,
		dbName,
	)
	var rows [][]interface{}
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &rows); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := make(map[string]float64, len(rows))
	for _, row := range rows {
		if len(row) != 2 {
			t.Fatalf("row %v has %d elements, want 2", row, len(row))
		}
		name, ok := row[0].(string)
		if !ok {
			t.Fatalf("row key %v is not a string", row[0])
		}
		count, ok := row[1].(float64)
		if !ok {
			t.Fatalf("row count %v is not a number", row[1])
		}
		got[name] = count
	}
	want := map[string]float64{"name": 2, "city": 2, "zip": 0}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("field counts = %v, want %v", got, want)
	}
}

func TestParserHasFieldsArraySelector(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice", "city": "berlin"},
		{"id": "2", "name": "bob"},
		{"id": "3", "city": "lisbon"},
	})

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").filter(f => f.hasFields(["name","city"])).count()`, dbName)
	var got float64
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 1 {
		t.Errorf("count = %v, want 1 (only the document with both fields)", got)
	}
}

func TestParserBracketLambdaParamKey(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice", "city": "berlin"},
	})

	expr := fmt.Sprintf(
		`r.expr(["name","city"]).map(function(f){ return r.db("%s").table("docs").get("1")(f) })`, dbName)
	var got []string
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []string{"alice", "berlin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("field values = %v, want %v", got, want)
	}
}

// parseRunRows runs expr and collects every row the cursor yields, preserving order.
func parseRunRows(t *testing.T, exec *query.Executor, expr string) []json.RawMessage {
	t.Helper()
	term, err := parser.Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	_, cur, err := exec.Run(context.Background(), term, nil)
	if err != nil {
		t.Fatalf("run %q: %v", expr, err)
	}
	defer closeCursor(cur)
	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	return rows
}

// rowIDs extracts the "id" field of each row.
func rowIDs(t *testing.T, rows []json.RawMessage) []string {
	t.Helper()
	ids := make([]string, len(rows))
	for i, raw := range rows {
		var doc struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal row %d: %v", i, err)
		}
		ids[i] = doc.ID
	}
	return ids
}

func TestParserOrderByIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "a", "score": 10},
		{"id": "b", "score": 30},
		{"id": "c", "score": 20},
	})
	waitForIndex(t, exec, dbName, "docs", "score")

	cases := []struct {
		name string
		expr string
		want []string
	}{
		{
			"desc_term_optarg",
			fmt.Sprintf(`r.db("%s").table("docs").orderBy({index: r.desc("score")})`, dbName),
			[]string{"b", "c", "a"},
		},
		{
			"asc_string_optarg",
			fmt.Sprintf(`r.db("%s").table("docs").orderBy({index: "score"})`, dbName),
			[]string{"a", "c", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rowIDs(t, parseRunRows(t, exec, tc.expr))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ids = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParserGetAllIndexOptArgs(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng"},
		{"id": "2", "dept": "hr"},
		{"id": "3", "dept": "eng"},
	})
	waitForIndex(t, exec, dbName, "docs", "dept")

	expr := fmt.Sprintf(`r.db("%s").table("docs").getAll("eng", {index: "dept"}).count()`, dbName)
	var got float64
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got != 2 {
		t.Errorf("count = %v, want 2", got)
	}
}

func TestParserFixesCLI(t *testing.T) {
	t.Parallel()
	qexec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, qexec, dbName)
	createTestTable(t, qexec, dbName, "items")
	docs := make([]map[string]interface{}, 10)
	for i := range docs {
		docs[i] = map[string]interface{}{"id": fmt.Sprintf("%d", i+1), "val": i + 1}
	}
	seedTable(t, qexec, dbName, "items", docs)

	expr := fmt.Sprintf(`r.db("%s").table("items").sample(3)`, dbName)
	stdout, _, code := cliRun(t, "", cliArgs("-f", "json", expr)...)
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &result); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	if len(result) != 3 {
		t.Errorf("got %d items, want 3", len(result))
	}
}

func TestParserTableListTopLevel(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "alpha")
	createTestTable(t, exec, dbName, "beta")

	stdout, stderr, code := cliRun(t, "", cliArgs("-d", dbName, "-f", "json", "r.tableList()")...)
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr)
	}
	var tables []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &tables); err != nil {
		t.Fatalf("unmarshal output: %v\noutput: %q", err, stdout)
	}
	sort.Strings(tables)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(tables, want) {
		t.Errorf("tables = %v, want %v", tables, want)
	}
}

func TestParserBranchChain(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "val": 1},
		{"id": "2", "val": 2},
	})

	cases := []struct {
		name string
		expr string
		want string
	}{
		{
			"true_side",
			fmt.Sprintf(`r.db("%s").table("docs").count().gt(0).branch("yes","no")`, dbName),
			"yes",
		},
		{
			"false_side",
			fmt.Sprintf(`r.db("%s").table("docs").count().gt(100).branch("yes","no")`, dbName),
			"no",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			if err := json.Unmarshal(parseRunAtom(t, exec, tc.expr), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != tc.want {
				t.Errorf("branch = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParserSliceSingleBound(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "vals": reql.Array(1, 2, 3, 4)},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").get("1")("vals").slice(-2)`, dbName)
	var got []float64
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	want := []float64{3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("slice(-2) = %v, want %v", got, want)
	}
}

func TestParserGroupZeroParamFunction(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "dept": "eng"},
		{"id": "2", "dept": "hr"},
		{"id": "3", "dept": "eng"},
	})

	expr := fmt.Sprintf(`r.db("%s").table("docs").group(function(){ return true }).count().ungroup()`, dbName)
	counts := ungroupedCounts(t, exec, expr)
	if len(counts) != 1 {
		t.Fatalf("got %d groups (%v), want 1", len(counts), counts)
	}
	if counts["true"] != 3 {
		t.Errorf("single group count = %v, want 3", counts["true"])
	}
}

func TestParserInfixArithmetic(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	cases := []struct {
		name string
		expr string
		want float64
	}{
		{"mul_chain", `r.expr(60*60*24*30)`, 2592000},
		{"precedence", `r.expr(1+2*3)`, 7},
		{"grouping", `r.expr((1+2)*3)`, 9},
		{"div_mod", `r.expr(10/2%3)`, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got float64
			if err := json.Unmarshal(parseRunAtom(t, exec, tc.expr), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got != tc.want {
				t.Errorf("%s = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

// atomStrings runs expr and decodes its single atom result as a string array.
func atomStrings(t *testing.T, exec *query.Executor, expr string) []string {
	t.Helper()
	var out []string
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &out); err != nil {
		t.Fatalf("unmarshal %q: %v", expr, err)
	}
	return out
}

func TestParserArithmeticBetweenBound(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")
	seedTable(t, exec, dbName, "events", []map[string]interface{}{
		{"id": "a", "ts": 1779222884700},
		{"id": "b", "ts": 1779222884701},
		{"id": "c", "ts": 1779222884705},
	})
	waitForIndex(t, exec, dbName, "events", "ts")

	// between over a selection returns an atom array, so the ids are collected server-side
	const tmpl = `r.db("%s").table("events").between(%s, %s, {index:"ts"}).orderBy("id").map(function(e){ return e("id") })`
	computed := fmt.Sprintf(tmpl, dbName, "1779222884700", "1779222884700+2")
	literal := fmt.Sprintf(tmpl, dbName, "1779222884700", "1779222884702")

	got := atomStrings(t, exec, computed)
	want := atomStrings(t, exec, literal)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("arithmetic bound ids = %v, literal bound ids = %v", got, want)
	}
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("ids = %v, want [a b]", got)
	}
}

func TestParserArithmeticNowFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "events")

	insert := fmt.Sprintf(
		`r.db("%s").table("events").insert([{id:"recent", ts: r.now()}, {id:"old", ts: r.now().sub(60*60*24*7)}])`,
		dbName)
	parseRunAtom(t, exec, insert)

	expr := fmt.Sprintf(
		`r.db("%s").table("events").filter(function(e){ return e("ts").gt(r.now().sub(60*60*24)) })`, dbName)
	got := rowIDs(t, parseRunRows(t, exec, expr))
	want := []string{"recent"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestParserFunctionLocalInFilter(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice"},
		{"id": "2", "name": "amy"},
		{"id": "3", "name": "bob"},
	})

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").filter(function(p){ var re = "^a"; return p("name").match(re) })`, dbName)
	got := rowIDs(t, parseRunRows(t, exec, expr))
	sort.Strings(got)
	want := []string{"1", "2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestParserFunctionLocalUsedTwice(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "score": 10},
		{"id": "2", "score": 20},
		{"id": "3", "score": 30},
	})

	// a local bound to a subquery and referenced twice must match the hand-inlined form
	local := fmt.Sprintf(
		`r.db("%s").table("docs").map(function(d){ var n = r.db("%s").table("docs").count(); return d("score").add(n).add(n) }).sum()`,
		dbName, dbName)
	inlined := fmt.Sprintf(
		`r.db("%s").table("docs").map(function(d){ return d("score").add(r.db("%s").table("docs").count()).add(r.db("%s").table("docs").count()) }).sum()`,
		dbName, dbName, dbName)

	var gotLocal, gotInlined float64
	if err := json.Unmarshal(parseRunAtom(t, exec, local), &gotLocal); err != nil {
		t.Fatalf("unmarshal local: %v", err)
	}
	if err := json.Unmarshal(parseRunAtom(t, exec, inlined), &gotInlined); err != nil {
		t.Fatalf("unmarshal inlined: %v", err)
	}
	if gotLocal != gotInlined {
		t.Errorf("local form = %v, inlined form = %v", gotLocal, gotInlined)
	}
	if gotLocal != 78 {
		t.Errorf("sum = %v, want 78", gotLocal)
	}
}

func TestParserArrowBlockBodyMap(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice"},
		{"id": "2", "name": "bob"},
	})

	expr := fmt.Sprintf(
		`r.db("%s").table("docs").map(g => { return {id: g("id"), n: g("name").upcase()} })`, dbName)
	got := make(map[string]string)
	for _, raw := range parseRunRows(t, exec, expr) {
		var doc struct {
			ID string `json:"id"`
			N  string `json:"n"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		got[doc.ID] = doc.N
	}
	want := map[string]string{"1": "ALICE", "2": "BOB"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reshaped rows = %v, want %v", got, want)
	}
}

// TestParserArrowBlockBodyGroupPipeline runs the account-balance report from the
// parser error log: group, ungroup, an arrow lambda with a block body, then orderBy.
func TestParserArrowBlockBodyGroupPipeline(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "accounts")
	seedTable(t, exec, dbName, "accounts", []map[string]interface{}{
		{"id": "1", "currency": "USD", "balance": map[string]interface{}{"amount": 10}},
		{"id": "2", "currency": "USD", "balance": map[string]interface{}{"amount": 30}},
		{"id": "3", "currency": "EUR", "balance": map[string]interface{}{"amount": 5}},
		{"id": "4", "currency": "EUR", "balance": map[string]interface{}{"amount": 0}},
	})

	expr := fmt.Sprintf(`r.db("%s").table("accounts").filter(t => t("balance")("amount").gt(0))`+
		`.group("currency").ungroup()`+
		`.map(g => {return {currency:g("group"), accounts:g("reduction").count(), `+
		`total:g("reduction").sum(x=>x("balance")("amount"))}})`+
		`.orderBy(r.desc("accounts"))`, dbName)

	var got []struct {
		Currency string  `json:"currency"`
		Accounts float64 `json:"accounts"`
		Total    float64 `json:"total"`
	}
	if err := json.Unmarshal(parseRunAtom(t, exec, expr), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows (%v), want 2", len(got), got)
	}
	if got[0].Currency != "USD" || got[0].Accounts != 2 || got[0].Total != 40 {
		t.Errorf("first row = %+v, want USD/2/40", got[0])
	}
	if got[1].Currency != "EUR" || got[1].Accounts != 1 || got[1].Total != 5 {
		t.Errorf("second row = %+v, want EUR/1/5", got[1])
	}
}
