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

func TestParse_EqJoin(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").eqJoin("id", r.table("other"))`)
	want := reql.DB("test").Table("users").EqJoin("id", reql.Table("other"))
	assertTermEqual(t, got, want)
}

func TestParse_Match(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").match("^foo")`)
	want := reql.DB("test").Table("users").Match("^foo")
	assertTermEqual(t, got, want)
}

func TestParse_Point(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.point(-122.4, 37.7)`)
	assertTermEqual(t, got, reql.Point(-122.4, 37.7))
}

func TestParse_EpochTime(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.epochTime(1234567890)`)
	assertTermEqual(t, got, reql.EpochTime(1234567890))
}

func TestParse_CoerceTo(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").coerceTo("string")`)
	want := reql.DB("test").Table("users").CoerceTo("string")
	assertTermEqual(t, got, want)
}

func TestParse_Default(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").default(0)`)
	want := reql.DB("test").Table("users").Default(0)
	assertTermEqual(t, got, want)
}

func TestParse_MethodMapping(t *testing.T) {
	t.Parallel()
	db := `r.db("test").table("users")`
	dbterm := reql.DB("test").Table("users")
	tests := []struct {
		name  string
		input string
		want  reql.Term
	}{
		// core sequence ops
		{"update", db + `.update({a: 1})`, dbterm.Update(reql.Datum(map[string]interface{}{"a": 1}))},
		{"delete", db + `.delete()`, dbterm.Delete()},
		{"skip", db + `.skip(5)`, dbterm.Skip(5)},
		{"count", db + `.count()`, dbterm.Count()},
		{"distinct", db + `.distinct()`, dbterm.Distinct()},
		{"replace", db + `.replace({a: 2})`, dbterm.Replace(reql.Datum(map[string]interface{}{"a": 2}))},
		// field ops
		{"group", db + `.group("age")`, dbterm.Group("age")},
		{"keys", db + `.keys()`, dbterm.Keys()},
		{"values", db + `.values()`, dbterm.Values()},
		{"sum", db + `.sum("score")`, dbterm.Sum("score")},
		{"avg", db + `.avg("score")`, dbterm.Avg("score")},
		{"typeOf", db + `.typeOf()`, dbterm.TypeOf()},
		{"map", db + `.map(r.row)`, dbterm.Map(reql.Row())},
		// comparisons
		{"eq", `r.row("x").eq(1)`, reql.Row().Bracket("x").Eq(1)},
		{"ne", `r.row("x").ne(0)`, reql.Row().Bracket("x").Ne(0)},
		{"lt", `r.row("x").lt(10)`, reql.Row().Bracket("x").Lt(10)},
		{"le", `r.row("x").le(10)`, reql.Row().Bracket("x").Le(10)},
		{"ge", `r.row("x").ge(0)`, reql.Row().Bracket("x").Ge(0)},
		{"not", `r.row("x").not()`, reql.Row().Bracket("x").Not()},
		{"and", `r.row("x").gt(0).and(r.row("x").lt(10))`, reql.Row().Bracket("x").Gt(0).And(reql.Row().Bracket("x").Lt(10))},
		{"or", `r.row("x").lt(0).or(r.row("x").gt(10))`, reql.Row().Bracket("x").Lt(0).Or(reql.Row().Bracket("x").Gt(10))},
		// arithmetic
		{"add", `r.row("x").add(1)`, reql.Row().Bracket("x").Add(1)},
		{"sub", `r.row("x").sub(1)`, reql.Row().Bracket("x").Sub(1)},
		{"mul", `r.row("x").mul(2)`, reql.Row().Bracket("x").Mul(2)},
		{"div", `r.row("x").div(2)`, reql.Row().Bracket("x").Div(2)},
		{"mod", `r.row("x").mod(3)`, reql.Row().Bracket("x").Mod(3)},
		{"floor", `r.row("x").floor()`, reql.Row().Bracket("x").Floor()},
		{"ceil", `r.row("x").ceil()`, reql.Row().Bracket("x").Ceil()},
		{"round", `r.row("x").round()`, reql.Row().Bracket("x").Round()},
		// strings
		{"upcase", `r.row("name").upcase()`, reql.Row().Bracket("name").Upcase()},
		{"downcase", `r.row("name").downcase()`, reql.Row().Bracket("name").Downcase()},
		// time
		{"date", `r.now().date()`, reql.Now().Date()},
		{"year", `r.now().year()`, reql.Now().Year()},
		{"inTimezone", `r.now().inTimezone("UTC")`, reql.Now().InTimezone("UTC")},
		// arrays
		{"append", db + `.append(1)`, dbterm.Append(1)},
		{"prepend", db + `.prepend(1)`, dbterm.Prepend(1)},
		{"setInsert", db + `.setInsert("x")`, dbterm.SetInsert("x")},
		// admin
		{"changes", db + `.changes()`, dbterm.Changes()},
		{"config", db + `.config()`, dbterm.Config()},
		{"tableList", `r.db("test").tableList()`, reql.DB("test").TableList()},
		{"tableCreate", `r.db("test").tableCreate("new")`, reql.DB("test").TableCreate("new")},
		{"tableDrop", `r.db("test").tableDrop("old")`, reql.DB("test").TableDrop("old")},
		{"indexCreate", db + `.indexCreate("idx")`, dbterm.IndexCreate("idx")},
		{"indexDrop", db + `.indexDrop("idx")`, dbterm.IndexDrop("idx")},
		{"indexList", db + `.indexList()`, dbterm.IndexList()},
		// r.* builders
		{"r.now", `r.now()`, reql.Now()},
		{"r.uuid", `r.uuid()`, reql.UUID()},
		{"r.dbCreate", `r.dbCreate("newdb")`, reql.DBCreate("newdb")},
		{"r.dbDrop", `r.dbDrop("olddb")`, reql.DBDrop("olddb")},
		{"r.dbList", `r.dbList()`, reql.DBList()},
		{"r.table", `r.table("users")`, reql.Table("users")},
		{"r.epochTime", `r.epochTime(1000)`, reql.EpochTime(1000)},
		{"r.literal", `r.literal(42)`, reql.Literal(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, tt.input)
			assertTermEqual(t, got, tt.want)
		})
	}
}
