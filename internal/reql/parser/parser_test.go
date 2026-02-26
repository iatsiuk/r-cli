package parser

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"r-cli/internal/reql"
)

type parseTest struct {
	name  string
	input string
	want  reql.Term
}

func runParseTests(t *testing.T, tests []parseTest) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertTermEqual(t, mustParse(t, tt.input), tt.want)
		})
	}
}

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

func TestParse_SyntaxError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.unknownThing()`, "unknown r.unknownThing"},
		{`r.db(`, "expected string literal"},
		{`r.db("test"`, "expected ')'"},
		{`r.db("test").unknownMethod()`, "unknown method .unknownMethod"},
		{`42 extra`, "unexpected token"},
		// comma required in arg list
		{`r.db("test").table("users").getAll("a" "b")`, "expected ','"},
		// branch requires odd arg count >= 3
		{`r.branch(true, "x")`, "r.branch requires"},
		{`r.branch(true)`, "r.branch requires"},
		// comma required in string list
		{`r.db("test").table("users").pluck("a" "b")`, "expected ','"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestParse_MaxDepth(t *testing.T) {
	t.Parallel()
	// build 257 levels deep: r.expr(r.expr(r.expr(...)))
	inner := `42`
	for range 257 {
		inner = `r.expr(` + inner + `)`
	}
	_, err := Parse(inner)
	if err == nil {
		t.Fatal("expected depth error, got nil")
	}
	if !strings.Contains(err.Error(), "deeply nested") {
		t.Errorf("expected 'deeply nested' error, got: %v", err)
	}
}

func TestParse_CoreMethodMapping(t *testing.T) {
	t.Parallel()
	db := `r.db("test").table("users")`
	dbterm := reql.DB("test").Table("users")
	runParseTests(t, []parseTest{
		{"update", db + `.update({a: 1})`, dbterm.Update(reql.Datum(map[string]interface{}{"a": 1}))},
		{"delete", db + `.delete()`, dbterm.Delete()},
		{"skip", db + `.skip(5)`, dbterm.Skip(5)},
		{"count", db + `.count()`, dbterm.Count()},
		{"distinct", db + `.distinct()`, dbterm.Distinct()},
		{"replace", db + `.replace({a: 2})`, dbterm.Replace(reql.Datum(map[string]interface{}{"a": 2}))},
		{"group", db + `.group("age")`, dbterm.Group("age")},
		{"keys", db + `.keys()`, dbterm.Keys()},
		{"values", db + `.values()`, dbterm.Values()},
		{"sum", db + `.sum("score")`, dbterm.Sum("score")},
		{"avg", db + `.avg("score")`, dbterm.Avg("score")},
		{"min", db + `.min("score")`, dbterm.Min("score")},
		{"max", db + `.max("score")`, dbterm.Max("score")},
		{"typeOf", db + `.typeOf()`, dbterm.TypeOf()},
		{"map", db + `.map(r.row)`, dbterm.Map(reql.Row())},
		{"eq", `r.row("x").eq(1)`, reql.Row().Bracket("x").Eq(1)},
		{"ne", `r.row("x").ne(0)`, reql.Row().Bracket("x").Ne(0)},
		{"lt", `r.row("x").lt(10)`, reql.Row().Bracket("x").Lt(10)},
		{"le", `r.row("x").le(10)`, reql.Row().Bracket("x").Le(10)},
		{"ge", `r.row("x").ge(0)`, reql.Row().Bracket("x").Ge(0)},
		{"not", `r.row("x").not()`, reql.Row().Bracket("x").Not()},
		{"and", `r.row("x").gt(0).and(r.row("x").lt(10))`, reql.Row().Bracket("x").Gt(0).And(reql.Row().Bracket("x").Lt(10))},
		{"or", `r.row("x").lt(0).or(r.row("x").gt(10))`, reql.Row().Bracket("x").Lt(0).Or(reql.Row().Bracket("x").Gt(10))},
		{"add", `r.row("x").add(1)`, reql.Row().Bracket("x").Add(1)},
		{"sub", `r.row("x").sub(1)`, reql.Row().Bracket("x").Sub(1)},
		{"mul", `r.row("x").mul(2)`, reql.Row().Bracket("x").Mul(2)},
		{"div", `r.row("x").div(2)`, reql.Row().Bracket("x").Div(2)},
		{"mod", `r.row("x").mod(3)`, reql.Row().Bracket("x").Mod(3)},
		{"floor", `r.row("x").floor()`, reql.Row().Bracket("x").Floor()},
		{"ceil", `r.row("x").ceil()`, reql.Row().Bracket("x").Ceil()},
		{"round", `r.row("x").round()`, reql.Row().Bracket("x").Round()},
	})
}

func TestParse_ChainMethodMapping(t *testing.T) {
	t.Parallel()
	db := `r.db("test").table("users")`
	dbterm := reql.DB("test").Table("users")
	runParseTests(t, []parseTest{
		{"pluck", db + `.pluck("a", "b")`, dbterm.Pluck("a", "b")},
		{"without", db + `.without("x")`, dbterm.Without("x")},
		{"hasFields", db + `.hasFields("a")`, dbterm.HasFields("a")},
		{"withFields", db + `.withFields("a")`, dbterm.WithFields("a")},
		{"getAll", db + `.getAll("a", "b")`, dbterm.GetAll("a", "b")},
		{"contains", db + `.contains("val")`, dbterm.Contains("val")},
		{"between", db + `.between(r.minval, r.maxval)`, dbterm.Between(reql.MinVal(), reql.MaxVal())},
		{"union", db + `.union(r.db("test").table("z"))`, dbterm.Union(reql.DB("test").Table("z"))},
		{"split_noarg", `r.row("s").split()`, reql.Row().Bracket("s").Split()},
		{"split_sep", `r.row("s").split(",")`, reql.Row().Bracket("s").Split(",")},
		{"insertAt", db + `.insertAt(0, "val")`, dbterm.InsertAt(0, reql.Datum("val"))},
		{"deleteAt", db + `.deleteAt(2)`, dbterm.DeleteAt(2)},
		{"changeAt", db + `.changeAt(1, "new")`, dbterm.ChangeAt(1, reql.Datum("new"))},
		{"spliceAt", db + `.spliceAt(0, [1, 2])`, dbterm.SpliceAt(0, reql.Array(1, 2))},
		{"slice", db + `.slice(0, 5)`, dbterm.Slice(0, 5)},
		{"indexWait", db + `.indexWait("idx")`, dbterm.IndexWait("idx")},
		{"indexStatus", db + `.indexStatus("idx")`, dbterm.IndexStatus("idx")},
		{"indexRename", db + `.indexRename("old", "new")`, dbterm.IndexRename("old", "new")},
		{"innerJoin", db + `.innerJoin(r.db("test").table("z"), r.row)`, dbterm.InnerJoin(reql.DB("test").Table("z"), reql.Row())},
		{"outerJoin", db + `.outerJoin(r.db("test").table("z"), r.row)`, dbterm.OuterJoin(reql.DB("test").Table("z"), reql.Row())},
		{"during", `r.now().during(r.epochTime(0), r.epochTime(1))`, reql.Now().During(reql.EpochTime(0), reql.EpochTime(1))},
		{"grant", db + `.grant("user", {read: true})`, dbterm.Grant("user", reql.Datum(map[string]interface{}{"read": true}))},
		{"upcase", `r.row("name").upcase()`, reql.Row().Bracket("name").Upcase()},
		{"downcase", `r.row("name").downcase()`, reql.Row().Bracket("name").Downcase()},
		{"date", `r.now().date()`, reql.Now().Date()},
		{"year", `r.now().year()`, reql.Now().Year()},
		{"inTimezone", `r.now().inTimezone("UTC")`, reql.Now().InTimezone("UTC")},
		{"append", db + `.append(1)`, dbterm.Append(1)},
		{"prepend", db + `.prepend(1)`, dbterm.Prepend(1)},
		{"setInsert", db + `.setInsert("x")`, dbterm.SetInsert("x")},
	})
}

func TestParse_AdminMethodMapping(t *testing.T) {
	t.Parallel()
	db := `r.db("test").table("users")`
	dbterm := reql.DB("test").Table("users")
	runParseTests(t, []parseTest{
		{"changes", db + `.changes()`, dbterm.Changes()},
		{"config", db + `.config()`, dbterm.Config()},
		{"tableList", `r.db("test").tableList()`, reql.DB("test").TableList()},
		{"tableCreate", `r.db("test").tableCreate("new")`, reql.DB("test").TableCreate("new")},
		{"tableDrop", `r.db("test").tableDrop("old")`, reql.DB("test").TableDrop("old")},
		{"indexCreate", db + `.indexCreate("idx")`, dbterm.IndexCreate("idx")},
		{"indexDrop", db + `.indexDrop("idx")`, dbterm.IndexDrop("idx")},
		{"indexList", db + `.indexList()`, dbterm.IndexList()},
		{"r.asc", `r.asc("name")`, reql.Asc("name")},
		{"r.now", `r.now()`, reql.Now()},
		{"r.uuid", `r.uuid()`, reql.UUID()},
		{"r.dbCreate", `r.dbCreate("newdb")`, reql.DBCreate("newdb")},
		{"r.dbDrop", `r.dbDrop("olddb")`, reql.DBDrop("olddb")},
		{"r.dbList", `r.dbList()`, reql.DBList()},
		{"r.table", `r.table("users")`, reql.Table("users")},
		{"r.epochTime", `r.epochTime(1000)`, reql.EpochTime(1000)},
		{"r.literal", `r.literal(42)`, reql.Literal(42)},
		{"r.json", `r.json("{\"a\":1}")`, reql.JSON(`{"a":1}`)},
		{"r.iso8601", `r.iso8601("2015-01-01T12:00:00+00:00")`, reql.ISO8601("2015-01-01T12:00:00+00:00")},
		{"r.geoJSON", `r.geoJSON({type: "Point"})`, reql.GeoJSON(reql.Datum(map[string]interface{}{"type": "Point"}))},
	})
}

func TestParse_DatumBoolNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  reql.Term
	}{
		{"true", reql.Datum(true)},
		{"false", reql.Datum(false)},
		{"null", reql.Datum(nil)},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := mustParse(t, tc.input)
			assertTermEqual(t, got, tc.want)
		})
	}
}

func TestParse_StringKeyedObject(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").filter({"name": "foo"})`)
	want := reql.DB("test").Table("users").Filter(reql.Datum(map[string]interface{}{"name": "foo"}))
	assertTermEqual(t, got, want)
}

func TestParseLambda_ChainMethods(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"filter_compound_predicate",
			`r.table('t').filter((doc) => doc('status').eq('active').and(doc('age').gt(18)))`,
			reql.Table("t").Filter(reql.Func(
				reql.Var(1).Bracket("status").Eq("active").And(reql.Var(1).Bracket("age").Gt(18)),
				1,
			)),
		},
		{
			"reduce_two_param",
			`r.table('t').reduce((a, b) => a.add(b))`,
			reql.Table("t").Reduce(reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2)),
		},
		{
			"concatMap_one_param",
			`r.table('t').concatMap((x) => x('items'))`,
			reql.Table("t").ConcatMap(reql.Func(reql.Var(1).Bracket("items"), 1)),
		},
		{
			"forEach_one_param",
			`r.table('t').forEach((x) => x('src').add('_copy'))`,
			reql.Table("t").ForEach(reql.Func(reql.Var(1).Bracket("src").Add("_copy"), 1)),
		},
		{
			"innerJoin_two_param",
			`r.table('a').innerJoin(r.table('b'), (left, right) => left('id').eq(right('id')))`,
			reql.Table("a").InnerJoin(reql.Table("b"), reql.Func(
				reql.Var(1).Bracket("id").Eq(reql.Var(2).Bracket("id")),
				1, 2,
			)),
		},
		{
			"outerJoin_two_param",
			`r.table('a').outerJoin(r.table('b'), (a, b) => a('k').eq(b('k')))`,
			reql.Table("a").OuterJoin(reql.Table("b"), reql.Func(
				reql.Var(1).Bracket("k").Eq(reql.Var(2).Bracket("k")),
				1, 2,
			)),
		},
	})
}

func TestParse_IntArgError(t *testing.T) {
	t.Parallel()
	_, err := Parse(`r.db("test").table("users").limit(3.14)`)
	if err == nil {
		t.Fatal("expected error for float arg to limit(), got nil")
	}
}

func TestParseLambda_SingleParamParen(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"field_gt",
			`(x) => x('age').gt(21)`,
			reql.Func(reql.Var(1).Bracket("age").Gt(21), 1),
		},
		{
			"eq",
			`(x) => x.eq(5)`,
			reql.Func(reql.Var(1).Eq(5), 1),
		},
		{
			"datum_bool",
			`(x) => true`,
			reql.Func(reql.Datum(true), 1),
		},
	})
}

func TestParseLambda_MultiParam(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"two_params",
			`(a, b) => a.add(b)`,
			reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2),
		},
		{
			"three_params",
			`(a, b, c) => a.add(b).add(c)`,
			reql.Func(reql.Var(1).Add(reql.Var(2)).Add(reql.Var(3)), 1, 2, 3),
		},
	})
}

func TestParseLambda_MultiParam_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`(x, x) => x`, "duplicate parameter name"},
		{`() => 1`, "at least one parameter"},
		{`(a,) => a`, "trailing comma"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestParseLambda_BareArrow(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"bare_field_gt",
			`x => x('field').gt(0)`,
			reql.Func(reql.Var(1).Bracket("field").Gt(0), 1),
		},
		{
			"same_as_paren_form",
			`x => x('age').gt(21)`,
			reql.Func(reql.Var(1).Bracket("age").Gt(21), 1),
		},
		{
			"inside_filter",
			`r.table('t').filter(x => x('age').gt(21))`,
			reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("age").Gt(21), 1)),
		},
		{
			"multi_var_refs",
			`x => x('a').add(x('b'))`,
			reql.Func(reql.Var(1).Bracket("a").Add(reql.Var(1).Bracket("b")), 1),
		},
	})
}

func TestParseLambda_BareArrow_FallThrough(t *testing.T) {
	t.Parallel()
	// bare ident without => falls through to datum error (unknown identifier)
	_, err := Parse(`z`)
	if err == nil {
		t.Fatal("Parse(\"z\"): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected token") {
		t.Errorf("Parse(\"z\"): error %q does not contain \"unexpected token\"", err.Error())
	}
}

func TestParseLambda_SingleParamParen_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`(false) => 1`, "expected identifier"},
		{`(null) => 1`, "expected identifier"},
		{`(x) =>`, "unexpected token"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestParseLambda_ScopingRules(t *testing.T) {
	t.Parallel()

	// multiple VAR refs: same param ID used in multiple places
	t.Run("multi_var_refs", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(x) => x('a').add(x('b')).mul(2)`)
		want := reql.Func(reql.Var(1).Bracket("a").Add(reql.Var(1).Bracket("b")).Mul(2), 1)
		assertTermEqual(t, got, want)
	})

	// chain methods on param
	t.Run("chain_on_param", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(doc) => doc('name').upcase().match('^A')`)
		want := reql.Func(reql.Var(1).Bracket("name").Upcase().Match("^A"), 1)
		assertTermEqual(t, got, want)
	})
}

func TestParseLambda_ScopingErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		// nested arrow: paren form inside paren form
		{`(x) => (y) => y`, "nested arrow functions"},
		// nested arrow: bare form inside paren form
		{`(x) => y => y`, "nested arrow functions"},
		// function expr inside arrow body
		{`(x) => function(y){ return y }`, "nested functions"},
		// r.row inside arrow
		{`(x) => r.row('f')`, "r.row inside arrow"},
		// unknown identifier in body (scope isolation)
		{`(x) => y`, "unexpected token"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestParseLambda_BodyBoundaries(t *testing.T) {
	t.Parallel()

	// arrow body is entire x('a').gt(1); outer filter paren closes after lambda
	t.Run("filter_body_gt", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table('t').filter((x) => x('a').gt(1))`)
		want := reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("a").Gt(1), 1))
		assertTermEqual(t, got, want)
	})

	// arrow body is x('ok'); remaining args "yes","no" are branch args
	t.Run("branch_arrow_first_arg", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.branch((x) => x('ok'), "yes", "no")`)
		want := reql.Branch(reql.Func(reql.Var(1).Bracket("ok"), 1), "yes", "no")
		assertTermEqual(t, got, want)
	})

	// filter with arrow must not double-wrap: exactly one FUNC(69) in wire output
	t.Run("filter_no_double_wrap", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table('t').filter((x) => x('a').gt(1))`)
		b, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		count := strings.Count(string(b), "[69,")
		if count != 1 {
			t.Errorf("expected exactly 1 FUNC(69) in wire JSON, got %d: %s", count, b)
		}
	})

	// map with arrow: no wrapImplicitVar needed, FUNC passed directly
	t.Run("map_arrow", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table('t').map((x) => x('price').mul(x('qty')))`)
		want := reql.Table("t").Map(reql.Func(reql.Var(1).Bracket("price").Mul(reql.Var(1).Bracket("qty")), 1))
		assertTermEqual(t, got, want)
	})
}

func TestParseFunctionExpr_SingleParam(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"with_return",
			`function(x){ return x('age').gt(21) }`,
			reql.Func(reql.Var(1).Bracket("age").Gt(21), 1),
		},
		{
			"without_return",
			`function(x){ x('age').gt(21) }`,
			reql.Func(reql.Var(1).Bracket("age").Gt(21), 1),
		},
		{
			"trailing_semicolon",
			`function(x){ return x('age').gt(21); }`,
			reql.Func(reql.Var(1).Bracket("age").Gt(21), 1),
		},
		{
			"in_filter",
			`r.table('t').filter(function(x){ return x('age').gt(21) })`,
			reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("age").Gt(21), 1)),
		},
	})
}

func TestParseLambda_RAsParam(t *testing.T) {
	t.Parallel()

	t.Run("paren_r_eq", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(r) => r('enabled').eq(false)`)
		want := reql.Func(reql.Var(1).Bracket("enabled").Eq(false), 1)
		assertTermEqual(t, got, want)
	})

	t.Run("filter_r_gt", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table('t').filter((r) => r('age').gt(21))`)
		want := reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("age").Gt(21), 1))
		assertTermEqual(t, got, want)
	})

	t.Run("multi_var_refs_r", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(r) => r('a').add(r('b'))`)
		want := reql.Func(reql.Var(1).Bracket("a").Add(reql.Var(1).Bracket("b")), 1)
		assertTermEqual(t, got, want)
	})

	t.Run("bare_arrow_r", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r => r('field')`)
		want := reql.Func(reql.Var(1).Bracket("field"), 1)
		assertTermEqual(t, got, want)
	})

	t.Run("r_db_regression", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db('test')`)
		assertTermEqual(t, got, reql.DB("test"))
	})

	t.Run("r_param_row_chain_error", func(t *testing.T) {
		t.Parallel()
		_, err := Parse(`r.table('t').filter((r) => r.row('f'))`)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unknown method") {
			t.Errorf("expected 'unknown method' error, got: %v", err)
		}
	})
}

func TestParseFunctionExpr_RAsParam(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"function_r_eq_false",
			`function(r){ return r('enabled').eq(false) }`,
			reql.Func(reql.Var(1).Bracket("enabled").Eq(false), 1),
		},
		{
			"full_chain_function_r",
			`r.db('restored').table('routes').filter(function(r){ return r('enabled').eq(false) })`,
			reql.DB("restored").Table("routes").Filter(reql.Func(reql.Var(1).Bracket("enabled").Eq(false), 1)),
		},
	})

	t.Run("arrow_r_same_as_function_r", func(t *testing.T) {
		t.Parallel()
		arrow := mustParse(t, `r.table('t').filter((r) => r('enabled').eq(false))`)
		fn := mustParse(t, `r.table('t').filter(function(r){ return r('enabled').eq(false) })`)
		assertTermEqual(t, arrow, fn)
	})
}

func TestParseFunctionExpr_MultiParam(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"two_params",
			`function(a, b){ return a.add(b) }`,
			reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2),
		},
		{
			"three_params",
			`function(a, b, c){ return a.add(b).add(c) }`,
			reql.Func(reql.Var(1).Add(reql.Var(2)).Add(reql.Var(3)), 1, 2, 3),
		},
	})
}

func TestParseFunctionExpr_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`function(){ return 1 }`, "at least one parameter"},
		{`function(x, x){ return x }`, "duplicate parameter name"},
		{`function(x){ }`, "unexpected token"},
		{`function(x){ return }`, "unexpected token"},
		{`function(x) x`, "expected '{'"},
		{`function(x){ return x('a')`, "expected '}'"},
		{`function(x){ return function(y){ return y } }`, "nested functions"},
		{`function(x){ return (y) => y }`, "nested arrow functions"},
		{`function(x){ return y => y }`, "nested arrow functions"},
		{`function(x){ return r.row('f') }`, "r.row inside arrow"},
		{`function(return){ return return }`, "reserved word"}, //nolint:dupword
		{`function(function){ return function }`, "reserved word"},
		{`(return) => return`, "reserved word"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestParse_BracketNumericIndex(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"limit_then_nth",
			`r.table("t").limit(1)(0)`,
			reql.Table("t").Limit(1).Nth(0),
		},
		{
			"row_bracket_then_nth",
			`r.row("items")(0)`,
			reql.Row().Bracket("items").Nth(0),
		},
		{
			"insert_bracket_nth_bracket",
			`r.table("t").insert({a: 1})("changes")(0)("new_val")`,
			reql.Table("t").Insert(reql.Datum(map[string]interface{}{"a": 1})).Bracket("changes").Nth(0).Bracket("new_val"),
		},
		{
			"table_nth_then_bracket",
			`r.table("t")(0)("name")`,
			reql.Table("t").Nth(0).Bracket("name"),
		},
		{
			"table_bracket_string_no_regression",
			`r.table("t")("field")`,
			reql.Table("t").Bracket("field"),
		},
		{
			"negative_index",
			`r.table("t")(-1)`,
			reql.Table("t").Nth(-1),
		},
	})
}

func TestParse_BracketNumericIndex_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.table("t")(0.5)`, "bracket index must be an integer"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
		})
	}
}
