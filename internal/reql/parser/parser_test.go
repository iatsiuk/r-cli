package parser

import (
	"encoding/json"
	"reflect"
	"testing"

	"r-cli/internal/reql"
)

func assertTermEqual(t *testing.T, got, want reql.Term) {
	t.Helper()
	g, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	w, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	var gv, wv interface{}
	if err := json.Unmarshal(g, &gv); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(w, &wv); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if !reflect.DeepEqual(gv, wv) {
		t.Errorf("term mismatch:\ngot:  %s\nwant: %s", g, w)
	}
}

func mustParse(t *testing.T, input string) reql.Term {
	t.Helper()
	term, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q) error: %v", input, err)
	}
	return term
}

func TestParse_DB(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test")`)
	assertTermEqual(t, got, reql.DB("test"))
}

func TestParse_DBTable(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users")`)
	assertTermEqual(t, got, reql.DB("test").Table("users"))
}

func TestParse_FilterObject(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").filter({name: "foo"})`)
	want := reql.DB("test").Table("users").Filter(reql.Datum(map[string]interface{}{"name": "foo"}))
	assertTermEqual(t, got, want)
}

func TestParse_Get(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").get("id")`)
	want := reql.DB("test").Table("users").Get("id")
	assertTermEqual(t, got, want)
}

func TestParse_Insert(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").insert({name: "foo"})`)
	want := reql.DB("test").Table("users").Insert(reql.Datum(map[string]interface{}{"name": "foo"}))
	assertTermEqual(t, got, want)
}

func TestParse_OrderByDesc(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").orderBy(r.desc("name"))`)
	want := reql.DB("test").Table("users").OrderBy(reql.Desc("name"))
	assertTermEqual(t, got, want)
}

func TestParse_Limit(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").limit(10)`)
	want := reql.DB("test").Table("users").Limit(10)
	assertTermEqual(t, got, want)
}

func TestParse_RowFieldGt(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.row("field").gt(21)`)
	want := reql.Row().Bracket("field").Gt(21)
	assertTermEqual(t, got, want)
}

func TestParse_FilterNestedRow(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").filter(r.row("age").gt(21))`)
	want := reql.DB("test").Table("users").Filter(reql.Row().Bracket("age").Gt(21))
	assertTermEqual(t, got, want)
}

func TestParse_BracketChain(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.row("field")("subfield")`)
	want := reql.Row().Bracket("field").Bracket("subfield")
	assertTermEqual(t, got, want)
}

func TestParse_Expr(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.expr([1, 2, 3])`)
	want := reql.Array(1, 2, 3)
	assertTermEqual(t, got, want)
}

func TestParse_MinVal(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.minval`)
	assertTermEqual(t, got, reql.MinVal())
}

func TestParse_MaxVal(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.maxval`)
	assertTermEqual(t, got, reql.MaxVal())
}

func TestParse_Branch(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.branch(r.row("x").gt(0), "pos", "neg")`)
	want := reql.Branch(reql.Row().Bracket("x").Gt(0), "pos", "neg")
	assertTermEqual(t, got, want)
}

func TestParse_Error(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.error("msg")`)
	assertTermEqual(t, got, reql.Error("msg"))
}

func TestParse_Args(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.args([r.minval, r.maxval])`)
	want := reql.Args(reql.Array(reql.MinVal(), reql.MaxVal()))
	assertTermEqual(t, got, want)
}
