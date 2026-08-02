//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"r-cli/internal/query"
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
