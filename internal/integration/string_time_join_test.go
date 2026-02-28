//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"r-cli/internal/reql"
)

func TestFilterMatch(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "name": "alice"},
		{"id": "2", "name": "bob"},
		{"id": "3", "name": "anna"},
	})

	// match names starting with 'a'
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Filter(
			reql.Row().GetField("name").Match("^a"),
		), nil)
	if err != nil {
		t.Fatalf("filter match: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d rows for name^a, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal doc: %v", err)
		}
		if len(doc.Name) == 0 || doc.Name[0] != 'a' {
			t.Errorf("unexpected name %q, want name starting with 'a'", doc.Name)
		}
	}
}

func TestInsertNowReadBack(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	before := time.Now().Add(-5 * time.Second)
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(
			map[string]interface{}{"id": "ts1", "created": reql.Now()},
		), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert with now: %v", err)
	}
	after := time.Now().Add(5 * time.Second)

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("ts1"), nil)
	if err != nil {
		t.Fatalf("get doc: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var doc struct {
		Created struct {
			ReqlType  string  `json:"$reql_type$"`
			EpochTime float64 `json:"epoch_time"`
		} `json:"created"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	if doc.Created.ReqlType != "TIME" {
		t.Errorf("$reql_type$=%q, want TIME", doc.Created.ReqlType)
	}
	ts := time.Unix(int64(doc.Created.EpochTime), 0)
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v outside expected range [%v, %v]", ts, before, after)
	}
}

func TestGroupByYear(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	// insert three docs: two in 2023, one in 2024
	docs := []map[string]interface{}{
		{"id": "1", "ts": reql.EpochTime(1672531200)}, // 2023-01-01
		{"id": "2", "ts": reql.EpochTime(1700000000)}, // 2023-11-14
		{"id": "3", "ts": reql.EpochTime(1704067200)}, // 2024-01-01
	}
	seedTable(t, exec, dbName, "docs", docs)

	// map each doc to its year using .year() time method
	yearFn := reql.Func(reql.Var(1).GetField("ts").Year(), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(yearFn), nil)
	if err != nil {
		t.Fatalf("map year: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}

	yearCounts := make(map[float64]int)
	for _, raw := range rows {
		var year float64
		if err := json.Unmarshal(raw, &year); err != nil {
			t.Fatalf("unmarshal year: %v", err)
		}
		yearCounts[year]++
	}
	if yearCounts[2023] != 2 {
		t.Errorf("year 2023 count=%d, want 2", yearCounts[2023])
	}
	if yearCounts[2024] != 1 {
		t.Errorf("year 2024 count=%d, want 1", yearCounts[2024])
	}
}

func TestEpochTimeRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	const epochVal = 1704067200.0 // 2024-01-01 00:00:00 UTC
	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Insert(
			map[string]interface{}{"id": "ep1", "ts": reql.EpochTime(epochVal)},
		), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert epochTime: %v", err)
	}

	_, cur2, err := exec.Run(ctx,
		reql.DB(dbName).Table("docs").Get("ep1").GetField("ts").ToEpochTime(), nil)
	if err != nil {
		t.Fatalf("get epoch time: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal epoch time: %v", err)
	}
	if got != epochVal {
		t.Errorf("epoch time roundtrip: got %v, want %v", got, epochVal)
	}
}

func TestEqJoinSecondaryIndex(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "users")
	createTestTable(t, exec, dbName, "orders")

	seedTable(t, exec, dbName, "users", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
		{"id": "u2", "name": "bob"},
	})
	seedTable(t, exec, dbName, "orders", []map[string]interface{}{
		{"id": "o1", "user_id": "u1", "amount": 100},
		{"id": "o2", "user_id": "u2", "amount": 200},
		{"id": "o3", "user_id": "u1", "amount": 150},
	})

	// create secondary index on orders.user_id to test eqJoin via index
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("orders").IndexCreate("user_id"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index create: %v", err)
	}
	_, cur, err = exec.Run(ctx, reql.DB(dbName).Table("orders").IndexWait("user_id"), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("index wait: %v", err)
	}

	// eqJoin users.id -> orders via secondary index user_id
	_, cur, err = exec.Run(ctx,
		reql.DB(dbName).Table("users").EqJoin("id", reql.DB(dbName).Table("orders"), reql.OptArgs{"index": "user_id"}), nil)
	if err != nil {
		t.Fatalf("eqJoin: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	// u1 matches o1 and o3, u2 matches o2: eqJoin on secondary index returns all matches
	if len(rows) != 3 {
		t.Fatalf("eqJoin got %d rows, want 3", len(rows))
	}
	for _, raw := range rows {
		var pair struct {
			Left  map[string]interface{} `json:"left"`
			Right map[string]interface{} `json:"right"`
		}
		if err := json.Unmarshal(raw, &pair); err != nil {
			t.Fatalf("unmarshal join pair: %v", err)
		}
		if pair.Left == nil || pair.Right == nil {
			t.Errorf("join pair has nil side: left=%v right=%v", pair.Left, pair.Right)
		}
		leftID, _ := pair.Left["id"].(string)
		rightUserID, _ := pair.Right["user_id"].(string)
		if leftID != rightUserID {
			t.Errorf("join mismatch: user.id=%q != order.user_id=%q", leftID, rightUserID)
		}
	}
}

func TestToJSONString(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		term  reql.Term
		check func(t *testing.T, got string)
	}{
		{
			name: "object",
			term: reql.Datum(map[string]interface{}{"a": 1}).ToJSONString(),
			check: func(t *testing.T, got string) {
				// normalize both sides to compare key/value pairs
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(got), &obj); err != nil {
					t.Fatalf("toJSONString(object) result not valid JSON: %q, err: %v", got, err)
				}
				if obj["a"] != float64(1) {
					t.Errorf("toJSONString(object)[a]=%v, want 1", obj["a"])
				}
			},
		},
		{
			name: "array",
			term: reql.Array(1, 2).ToJSONString(),
			check: func(t *testing.T, got string) {
				var arr []float64
				if err := json.Unmarshal([]byte(got), &arr); err != nil {
					t.Fatalf("toJSONString(array) result not valid JSON: %q, err: %v", got, err)
				}
				if len(arr) != 2 || arr[0] != 1 || arr[1] != 2 {
					t.Errorf("toJSONString(array)=%v, want [1,2]", arr)
				}
			},
		},
		{
			name: "string",
			term: reql.Datum("hello").ToJSONString(),
			check: func(t *testing.T, got string) {
				// toJSONString of a string wraps it in JSON quotes
				if got != `"hello"` {
					t.Errorf("toJSONString(string)=%q, want %q", got, `"hello"`)
				}
			},
		},
		{
			name: "number",
			term: reql.Datum(42).ToJSONString(),
			check: func(t *testing.T, got string) {
				if got != "42" {
					t.Errorf("toJSONString(number)=%q, want %q", got, "42")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, cur, err := exec.Run(ctx, tc.term, nil)
			if err != nil {
				t.Fatalf("run: %v", err)
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
			tc.check(t, got)
		})
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	cases := []struct {
		name  string
		term  reql.Term
		want  []string
	}{
		{
			name: "delimiter",
			term: reql.Datum("a,b,c").Split(","),
			want: []string{"a", "b", "c"},
		},
		{
			name: "whitespace",
			term: reql.Datum("hello world foo").Split(),
			want: []string{"hello", "world", "foo"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, cur, err := exec.Run(ctx, tc.term, nil)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			defer closeCursor(cur)

			raw, err := cur.Next()
			if err != nil {
				t.Fatalf("cursor next: %v", err)
			}
			var got []string
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("split got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("split[%d]=%q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestDowncase(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)
	ctx := context.Background()

	cases := []struct {
		input string
		want  string
	}{
		{"HELLO", "hello"},
		{"MixedCase", "mixedcase"},
		{"already", "already"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, cur, err := exec.Run(ctx, reql.Datum(tc.input).Downcase(), nil)
			if err != nil {
				t.Fatalf("run: %v", err)
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
				t.Errorf("downcase(%q)=%q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSplitOnTableField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "tags": "go,rethinkdb,test"},
		{"id": "2", "tags": "foo,bar"},
	})

	fn := reql.Func(reql.Var(1).GetField("tags").Split(","), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(fn), nil)
	if err != nil {
		t.Fatalf("map split: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// collect expected split lengths (order not guaranteed)
	gotLens := make(map[int]int)
	for _, raw := range rows {
		var parts []string
		if err := json.Unmarshal(raw, &parts); err != nil {
			t.Fatalf("unmarshal split result: %v", err)
		}
		gotLens[len(parts)]++
	}
	// "go,rethinkdb,test" -> 3 parts, "foo,bar" -> 2 parts
	if gotLens[3] != 1 || gotLens[2] != 1 {
		t.Errorf("unexpected split lengths: %v", gotLens)
	}
}

func TestDowncaseOnTableField(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")
	seedTable(t, exec, dbName, "docs", []map[string]interface{}{
		{"id": "1", "label": "ALPHA"},
		{"id": "2", "label": "BETA"},
	})

	fn := reql.Func(reql.Var(1).GetField("label").Downcase(), 1)
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Map(fn), nil)
	if err != nil {
		t.Fatalf("map downcase: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}

	// collect lowercased values (order not guaranteed)
	gotSet := make(map[string]bool)
	for _, raw := range rows {
		var got string
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("unmarshal downcase result: %v", err)
		}
		gotSet[got] = true
	}
	for _, want := range []string{"alpha", "beta"} {
		if !gotSet[want] {
			t.Errorf("downcase result missing %q, got %v", want, gotSet)
		}
	}
}

func TestEqJoinZip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "users")
	createTestTable(t, exec, dbName, "orders")

	seedTable(t, exec, dbName, "users", []map[string]interface{}{
		{"id": "u1", "name": "alice"},
		{"id": "u2", "name": "bob"},
	})
	seedTable(t, exec, dbName, "orders", []map[string]interface{}{
		{"id": "o1", "user_id": "u1", "amount": 100},
		{"id": "o2", "user_id": "u2", "amount": 200},
	})

	_, cur, err := exec.Run(ctx,
		reql.DB(dbName).Table("orders").EqJoin("user_id", reql.DB(dbName).Table("users")).Zip(), nil)
	if err != nil {
		t.Fatalf("eqJoin zip: %v", err)
	}
	defer closeCursor(cur)

	rows, err := cur.All()
	if err != nil {
		t.Fatalf("cursor all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("eqJoin+zip got %d rows, want 2", len(rows))
	}
	for _, raw := range rows {
		var doc map[string]interface{}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("unmarshal zipped doc: %v", err)
		}
		// zipped doc should have fields from both sides: id, user_id, amount, name
		if _, ok := doc["name"]; !ok {
			t.Errorf("zipped doc missing 'name' field: %v", doc)
		}
		if _, ok := doc["amount"]; !ok {
			t.Errorf("zipped doc missing 'amount' field: %v", doc)
		}
	}
}
