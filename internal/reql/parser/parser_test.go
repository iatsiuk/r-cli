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
		{"toJSON", db + `.toJSON()`, dbterm.ToJSONString()},
		{"toJsonString", db + `.toJsonString()`, dbterm.ToJSONString()},
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
		// r.row inside arrow
		{`(x) => r.row('f')`, "r.row inside arrow"},
		// r.row inside nested arrow
		{`(x) => (y) => r.row("f")`, "r.row inside arrow"},
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

func TestParseNestedFunctions(t *testing.T) {
	t.Parallel()

	// function(a){ return function(b){ return b } } -> outer FUNC(VAR(1)), inner FUNC(VAR(2))
	t.Run("function_in_function", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `function(a){ return function(b){ return b } }`)
		want := reql.Func(reql.Func(reql.Var(2), 2), 1)
		assertTermEqual(t, got, want)
	})

	// (a) => (b) => b -> outer FUNC(VAR(1)), inner FUNC(VAR(2))
	t.Run("arrow_in_arrow", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(a) => (b) => b`)
		want := reql.Func(reql.Func(reql.Var(2), 2), 1)
		assertTermEqual(t, got, want)
	})

	// function(x){ return (y) => y("f") } -> mixed: outer function, inner arrow
	t.Run("arrow_in_function", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `function(x){ return (y) => y("f") }`)
		want := reql.Func(reql.Func(reql.Var(2).Bracket("f"), 2), 1)
		assertTermEqual(t, got, want)
	})

	// (x) => function(y){ return y("f") } -> mixed: outer arrow, inner function
	t.Run("function_in_arrow", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(x) => function(y){ return y("f") }`)
		want := reql.Func(reql.Func(reql.Var(2).Bracket("f"), 2), 1)
		assertTermEqual(t, got, want)
	})

	// parameter shadowing (x) => (x) => x -> inner x is VAR(2), not VAR(1)
	t.Run("param_shadowing", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(x) => (x) => x`)
		want := reql.Func(reql.Func(reql.Var(2), 2), 1)
		assertTermEqual(t, got, want)
	})

	// outer param accessible in inner body (a) => (b) => a.add(b)
	t.Run("outer_param_in_inner_body", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(a) => (b) => a.add(b)`)
		want := reql.Func(reql.Func(reql.Var(1).Add(reql.Var(2)), 2), 1)
		assertTermEqual(t, got, want)
	})

	// r.row inside nested lambda -> error
	t.Run("row_inside_nested", func(t *testing.T) {
		t.Parallel()
		_, err := Parse(`(x) => (y) => r.row("f")`)
		if err == nil {
			t.Fatal("expected error for r.row inside nested lambda")
		}
		if !strings.Contains(err.Error(), "r.row inside arrow") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// three levels (a) => (b) => (c) => c
	t.Run("three_levels", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(a) => (b) => (c) => c`)
		want := reql.Func(reql.Func(reql.Func(reql.Var(3), 3), 2), 1)
		assertTermEqual(t, got, want)
	})

	// bare arrow nested: (x) => y => y
	t.Run("bare_arrow_nested", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `(x) => y => y`)
		want := reql.Func(reql.Func(reql.Var(2), 2), 1)
		assertTermEqual(t, got, want)
	})
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
		{`function(x, x){ return x }`, "duplicate parameter name"},
		{`function(x){ }`, "unexpected token"},
		{`function(x){ return }`, "unexpected token"},
		{`function(x) x`, "expected '{'"},
		{`function(x){ return x('a')`, "expected '}'"},
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
		{
			"row_nth",
			`r.row(0)`,
			reql.Row().Nth(0),
		},
	})
}

func TestParse_Sample(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"sample_5",
			`r.table("t").sample(5)`,
			reql.Table("t").Sample(5),
		},
		{
			"sample_1",
			`r.table("t").sample(1)`,
			reql.Table("t").Sample(1),
		},
		{
			"sample_0_edge_case",
			`r.table("t").sample(0)`,
			reql.Table("t").Sample(0),
		},
		{
			"sample_chained_pluck",
			`r.table("t").sample(1).pluck("id", "name")`,
			reql.Table("t").Sample(1).Pluck("id", "name"),
		},
	})
}

func TestParse_ParenGrouping(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"arrow_paren_object_one_field",
			`r.table("t").map(row => ({name: row("name")}))`,
			reql.Table("t").Map(reql.Func(
				reql.Datum(map[string]interface{}{"name": reql.Var(1).Bracket("name")}),
				1,
			)),
		},
		{
			"arrow_paren_object_two_fields",
			`r.table("t").map(row => ({a: row("x"), b: row("y")}))`,
			reql.Table("t").Map(reql.Func(
				reql.Datum(map[string]interface{}{"a": reql.Var(1).Bracket("x"), "b": reql.Var(1).Bracket("y")}),
				1,
			)),
		},
		{
			"paren_arrow_object_with_chain",
			`r.table("t").map((x) => ({id: x("id"), n: x("name").upcase()}))`,
			reql.Table("t").Map(reql.Func(
				reql.Datum(map[string]interface{}{"id": reql.Var(1).Bracket("id"), "n": reql.Var(1).Bracket("name").Upcase()}),
				1,
			)),
		},
		{
			"arrow_no_paren_no_regression",
			`r.table("t").map(row => row("name"))`,
			reql.Table("t").Map(reql.Func(reql.Var(1).Bracket("name"), 1)),
		},
		{
			"filter_no_regression",
			`r.table("t").filter(row => row("age").gt(21))`,
			reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("age").Gt(21), 1)),
		},
	})
}

func TestParse_ParenGrouping_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`(`, "unexpected token"},
		{`(r.table("t")`, "expected ')'"},
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

func TestParseNestedFunctionsChain(t *testing.T) {
	t.Parallel()

	// map with function containing nested filter with function
	t.Run("map_function_nested_filter_function", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").map(function(doc){ return doc("items").filter(function(i){ return i("active").eq(true) }) })`)
		want := reql.Table("t").Map(
			reql.Func(
				reql.Var(1).Bracket("items").Filter(
					reql.Func(reql.Var(2).Bracket("active").Eq(true), 2),
				),
				1,
			),
		)
		assertTermEqual(t, got, want)
	})

	// same structure with arrow syntax
	t.Run("map_arrow_nested_filter_arrow", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").map((doc) => doc("items").filter((i) => i("active").eq(true)))`)
		want := reql.Table("t").Map(
			reql.Func(
				reql.Var(1).Bracket("items").Filter(
					reql.Func(reql.Var(2).Bracket("active").Eq(true), 2),
				),
				1,
			),
		)
		assertTermEqual(t, got, want)
	})

	// filter with nested contains and function predicate
	t.Run("filter_function_nested_contains_function", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").filter(function(doc){ return doc("tags").contains(function(tag){ return tag.eq("hot") }) })`)
		want := reql.Table("t").Filter(
			reql.Func(
				reql.Var(1).Bracket("tags").Contains(
					reql.Func(reql.Var(2).Eq("hot"), 2),
				),
				1,
			),
		)
		assertTermEqual(t, got, want)
	})

	// map with function and merge containing no inner function (regression)
	t.Run("map_function_merge_no_inner_function", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").map(function(doc){ return doc.merge({count: doc("items").count()}) })`)
		want := reql.Table("t").Map(
			reql.Func(
				reql.Var(1).Merge(reql.Datum(map[string]interface{}{
					"count": reql.Var(1).Bracket("items").Count(),
				})),
				1,
			),
		)
		assertTermEqual(t, got, want)
	})
}

func TestParseNestedFunctionsChain_SiblingLambdaVarIDs(t *testing.T) {
	t.Parallel()
	// sibling lambdas must independently use VAR(1) to preserve backward compat
	// (each top-level lambda resets nextVarID to 0 after the previous one pops)
	got := mustParse(t, `r.table("t").map(x => x).filter(y => y)`)
	want := reql.Table("t").
		Map(reql.Func(reql.Var(1), 1)).
		Filter(reql.Func(reql.Var(1), 1))
	assertTermEqual(t, got, want)
}

func TestParse_InsertUpdateDeleteOptArgs(t *testing.T) {
	t.Parallel()
	tbl := reql.Table("t")
	doc1 := reql.Datum(map[string]interface{}{"a": int64(1)})
	cases := []parseTest{
		{
			"insert_return_changes",
			`r.table("t").insert({a: 1}, {return_changes: true})`,
			tbl.Insert(doc1, reql.OptArgs{"return_changes": true}),
		},
		{
			"insert_conflict_replace",
			`r.table("t").insert({a: 1}, {conflict: "replace"})`,
			tbl.Insert(doc1, reql.OptArgs{"conflict": "replace"}),
		},
		{
			"insert_multi_optargs",
			`r.table("t").insert({a: 1}, {durability: "soft", return_changes: true})`,
			tbl.Insert(doc1, reql.OptArgs{"durability": "soft", "return_changes": true}),
		},
		{
			"insert_no_optargs",
			`r.table("t").insert({a: 1})`,
			tbl.Insert(doc1),
		},
		{
			"update_optargs",
			`r.table("t").update({x: 1}, {durability: "soft"})`,
			tbl.Update(reql.Datum(map[string]interface{}{"x": int64(1)}), reql.OptArgs{"durability": "soft"}),
		},
		{
			"delete_optargs",
			`r.table("t").delete({durability: "soft"})`,
			tbl.Delete(reql.OptArgs{"durability": "soft"}),
		},
		{
			"delete_no_optargs",
			`r.table("t").delete()`,
			tbl.Delete(),
		},
		{
			"insert_optargs_chained",
			`r.table("t").insert({a: 1}, {return_changes: true})("changes")(0)("new_val")`,
			tbl.Insert(doc1, reql.OptArgs{"return_changes": true}).Bracket("changes").Nth(0).Bracket("new_val"),
		},
	}
	runParseTests(t, cases)
}

func TestParse_InsertUpdateDeleteOptArgs_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.table("t").insert({a: 1}, "bad")`, "insert: second argument must be an optargs object"},
		{`r.table("t").update({x: 1}, "bad")`, "update: second argument must be an optargs object"},
		{`r.table("t").delete("bad")`, "delete: argument must be an optargs object"},
		{`r.table("t").insert({a: 1}, {return_changes: true,})`, "trailing comma in opts"},
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

func TestParse_BracketNumericIndex_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.table("t")(0.5)`, "bracket index must be an integer"},
		{`r.table("t")(true)`, "expected string, integer or expression in bracket notation"},
		{`r.table("t")(null)`, "expected string, integer or expression in bracket notation"},
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

func TestParse_GeoConstructors(t *testing.T) {
	t.Parallel()
	p1 := reql.Point(-73.9857, 40.7484)
	p2 := reql.Point(-73.9712, 40.7614)
	p3 := reql.Point(-73.9442, 40.6782)
	runParseTests(t, []parseTest{
		{
			"line_two_points",
			`r.line(r.point(-73.9857, 40.7484), r.point(-73.9712, 40.7614))`,
			reql.Line(p1, p2),
		},
		{
			"line_three_points",
			`r.line(r.point(-73.9857, 40.7484), r.point(-73.9712, 40.7614), r.point(-73.9442, 40.6782))`,
			reql.Line(p1, p2, p3),
		},
		{
			"polygon_three_points",
			`r.polygon(r.point(-73.9857, 40.7484), r.point(-73.9712, 40.7614), r.point(-73.9442, 40.6782))`,
			reql.Polygon(p1, p2, p3),
		},
		{
			"circle_no_opts",
			`r.circle(r.point(-73.9857, 40.7484), 1000)`,
			reql.Circle(p1, 1000),
		},
		{
			"circle_with_opts",
			`r.circle(r.point(-73.9857, 40.7484), 1000, {num_vertices: 32})`,
			reql.Circle(p1, 1000, reql.OptArgs{"num_vertices": int64(32)}),
		},
	})
}

func TestParse_Time(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"4arg_form",
			`r.time(2024, 1, 15, "+00:00")`,
			reql.Time(2024, 1, 15, "+00:00"),
		},
		{
			"7arg_form",
			`r.time(2024, 1, 15, 10, 30, 0, "+00:00")`,
			reql.TimeAt(2024, 1, 15, 10, 30, 0, "+00:00"),
		},
		{
			"7arg_negative_seconds",
			`r.time(2024, 6, 1, 23, 59, 45, "+05:30")`,
			reql.TimeAt(2024, 6, 1, 23, 59, 45, "+05:30"),
		},
	})
}

func TestParse_Time_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		// too few args: missing timezone
		{`r.time(2024, 1, 15)`, "expected ','"},
		// 5-arg form: disambiguated as 7-arg (hour=10 is a number), then fails at minute
		{`r.time(2024, 1, 15, 10, "+00:00")`, "r.time minute"},
		// 6-arg form: disambiguated as 7-arg, then fails at second
		{`r.time(2024, 1, 15, 10, 30, "+00:00")`, "r.time second"},
		// 7-arg form missing second and timezone
		{`r.time(2024, 1, 15, 10, 30)`, "expected ','"},
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

func TestParse_Binary(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"string_arg",
			`r.binary("hello")`,
			reql.Binary(reql.Datum("hello")),
		},
	})
}

func TestParse_Object(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"two_pairs",
			`r.object("a", 1, "b", 2)`,
			reql.Object("a", int64(1), "b", int64(2)),
		},
		{
			"empty",
			`r.object()`,
			reql.Object(),
		},
	})
}

func TestParse_Range(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"no_args",
			`r.range()`,
			reql.Range(),
		},
		{
			"one_arg",
			`r.range(10)`,
			reql.Range(int64(10)),
		},
		{
			"two_args",
			`r.range(1, 10)`,
			reql.Range(int64(1), int64(10)),
		},
	})
}

func TestParse_Random(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"no_args",
			`r.random()`,
			reql.Random(),
		},
		{
			"one_arg",
			`r.random(100)`,
			reql.Random(int64(100)),
		},
		{
			"two_args",
			`r.random(1, 10)`,
			reql.Random(int64(1), int64(10)),
		},
		{
			"with_float_opt",
			`r.random(1, 10, {float: true})`,
			reql.Random(int64(1), int64(10), reql.OptArgs{"float": true}),
		},
	})
}

func TestParse_ObjectRange_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.object("a", 1, "b")`, "even number"},
		{`r.range(1, 2, 3)`, "0, 1, or 2 arguments"},
		{`r.random(1, 2, 3)`, "r.random accepts 0, 1, or 2"},
		{`r.random(1,)`, "trailing comma"},
		{`r.random(1, 2,)`, "trailing comma"},
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

func TestParse_GeoConstructors_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.line(r.point(-73.9857, 40.7484))`, "r.line requires at least 2"},
		{`r.polygon(r.point(-73.9857, 40.7484), r.point(-73.9712, 40.7614))`, "r.polygon requires at least 3"},
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

func TestParse_InfoOffsetsOf(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"info_on_table",
			`r.db("test").table("users").info()`,
			reql.DB("test").Table("users").Info(),
		},
		{
			"offsetsOf_value",
			`r.expr(["a","b","a"]).offsetsOf("a")`,
			reql.Array(reql.Datum("a"), reql.Datum("b"), reql.Datum("a")).OffsetsOf(reql.Datum("a")),
		},
		{
			"offsetsOf_lambda",
			`r.expr([1,2,3]).offsetsOf(x => x.gt(1))`,
			reql.Array(reql.Datum(int64(1)), reql.Datum(int64(2)), reql.Datum(int64(3))).OffsetsOf(
				reql.Func(reql.Var(1).Gt(reql.Datum(int64(1))), 1),
			),
		},
	})
}

func TestParse_Bitwise(t *testing.T) {
	t.Parallel()
	base := `r.expr(5)`
	baseTerm := reql.Datum(int64(5))
	runParseTests(t, []parseTest{
		{
			"bitAnd",
			base + `.bitAnd(3)`,
			baseTerm.BitAnd(reql.Datum(int64(3))),
		},
		{
			"bitOr",
			base + `.bitOr(3)`,
			baseTerm.BitOr(reql.Datum(int64(3))),
		},
		{
			"bitXor",
			base + `.bitXor(3)`,
			baseTerm.BitXor(reql.Datum(int64(3))),
		},
		{
			"bitNot",
			base + `.bitNot()`,
			baseTerm.BitNot(),
		},
		{
			"bitSal",
			base + `.bitSal(2)`,
			baseTerm.BitSal(reql.Datum(int64(2))),
		},
		{
			"bitSar",
			base + `.bitSar(1)`,
			baseTerm.BitSar(reql.Datum(int64(1))),
		},
	})
}

func TestParse_Fold(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"fold_basic",
			`r.expr([1,2,3]).fold(0, (acc, x) => acc.add(x))`,
			reql.Array(reql.Datum(int64(1)), reql.Datum(int64(2)), reql.Datum(int64(3))).Fold(
				reql.Datum(int64(0)),
				reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2),
			),
		},
		{
			"fold_with_emit_opt",
			`r.expr([1,2,3]).fold(0, (acc, x) => acc.add(x), {emit: new_acc => [new_acc]})`,
			reql.Array(reql.Datum(int64(1)), reql.Datum(int64(2)), reql.Datum(int64(3))).Fold(
				reql.Datum(int64(0)),
				reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2),
				reql.OptArgs{"emit": reql.Func(reql.Array(reql.Var(1)), 1)},
			),
		},
	})
}

func TestParse_Do(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"toplevel_single_arg",
			`r.do(r.table("t"), t => t.count())`,
			reql.Do(reql.Table("t"), reql.Func(reql.Var(1).Count(), 1)),
		},
		{
			"chain_form",
			`r.table("t").do(t => t.count())`,
			reql.Table("t").Do(reql.Func(reql.Var(1).Count(), 1)),
		},
		{
			"toplevel_multi_arg",
			`r.do(r.expr(1), r.expr(2), (a, b) => a.add(b))`,
			reql.Do(reql.Datum(int64(1)), reql.Datum(int64(2)), reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2)),
		},
	})
}

func TestParse_Do_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input   string
		wantMsg string
	}{
		{`r.do()`, "r.do requires at least a function argument"},
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

func TestParse_OptArgs_GetAll(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").getAll("a", "b", {index: "idx"})`)
	want := reql.DB("d").Table("t").GetAll("a", "b", reql.OptArgs{"index": "idx"})
	assertTermEqual(t, got, want)
}

func TestParse_GetAll_ObjectKey(t *testing.T) {
	t.Parallel()
	// object literal as positional key must not be misclassified as OptArgs
	got := mustParse(t, `r.db("d").table("t").getAll({id: "x"})`)
	want := reql.DB("d").Table("t").GetAll(reql.Datum(map[string]interface{}{"id": "x"}))
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_OrderBy(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").orderBy("name", {index: "idx"})`)
	want := reql.DB("d").Table("t").OrderBy("name", reql.OptArgs{"index": "idx"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_OrderByOnlyIndex(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").orderBy({index: "idx"})`)
	want := reql.DB("d").Table("t").OrderBy(reql.OptArgs{"index": "idx"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_Between(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").between(1, 10, {index: "score", left_bound: "closed"})`)
	want := reql.DB("d").Table("t").Between(reql.Datum(int64(1)), reql.Datum(int64(10)), reql.OptArgs{"index": "score", "left_bound": "closed"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_EqJoin(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.table("t").eqJoin("field", r.table("t2"), {index: "idx"})`)
	want := reql.Table("t").EqJoin("field", reql.Table("t2"), reql.OptArgs{"index": "idx"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_Distance(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.point(0, 0).distance(r.point(1, 1), {unit: "km"})`)
	want := reql.Point(0, 0).Distance(reql.Point(1, 1), reql.OptArgs{"unit": "km"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_GetIntersecting(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.table("t").getIntersecting(r.point(0, 0), {index: "geo"})`)
	want := reql.Table("t").GetIntersecting(reql.Point(0, 0), reql.OptArgs{"index": "geo"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_GetNearest(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.table("t").getNearest(r.point(0, 0), {index: "geo", max_dist: 1000})`)
	want := reql.Table("t").GetNearest(reql.Point(0, 0), reql.OptArgs{"index": "geo", "max_dist": int64(1000)})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_TableCreate(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").tableCreate("t", {primary_key: "uid"})`)
	want := reql.DB("d").TableCreate("t", reql.OptArgs{"primary_key": "uid"})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_IndexCreate(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").indexCreate("idx", {multi: true})`)
	want := reql.DB("d").Table("t").IndexCreate("idx", reql.OptArgs{"multi": true})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_Changes(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").changes({include_initial: true})`)
	want := reql.DB("d").Table("t").Changes(reql.OptArgs{"include_initial": true})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_Reconfigure(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("d").table("t").reconfigure({shards: 2, replicas: 1})`)
	want := reql.DB("d").Table("t").Reconfigure(reql.OptArgs{"shards": int64(2), "replicas": int64(1)})
	assertTermEqual(t, got, want)
}

func TestParse_OptArgs_CamelCaseConversion(t *testing.T) {
	t.Parallel()
	t.Run("between_leftBound", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db("d").table("t").between(1, 10, {index: "x", leftBound: "closed"})`)
		want := reql.DB("d").Table("t").Between(reql.Datum(int64(1)), reql.Datum(int64(10)), reql.OptArgs{"index": "x", "left_bound": "closed"})
		assertTermEqual(t, got, want)
	})
	t.Run("insert_returnChanges", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db("d").table("t").insert({a: 1}, {returnChanges: true})`)
		want := reql.DB("d").Table("t").Insert(reql.Datum(map[string]interface{}{"a": int64(1)}), reql.OptArgs{"return_changes": true})
		assertTermEqual(t, got, want)
	})
	t.Run("getAll_already_snake", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db("d").table("t").getAll("a", {index: "idx"})`)
		want := reql.DB("d").Table("t").GetAll("a", reql.OptArgs{"index": "idx"})
		assertTermEqual(t, got, want)
	})
	t.Run("changes_includeInitial", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db("d").table("t").changes({includeInitial: true})`)
		want := reql.DB("d").Table("t").Changes(reql.OptArgs{"include_initial": true})
		assertTermEqual(t, got, want)
	})
	t.Run("filter_data_object_preserved", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.db("d").table("t").filter({firstName: "Alice"})`)
		want := reql.DB("d").Table("t").Filter(reql.Datum(map[string]interface{}{"firstName": "Alice"}))
		assertTermEqual(t, got, want)
	})
	t.Run("changes_string_key_includeInitial", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").changes({"includeInitial": true})`)
		want := reql.Table("t").Changes(reql.OptArgs{"include_initial": true})
		assertTermEqual(t, got, want)
	})
	t.Run("fold_finalEmit", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.expr([1]).fold(0, (a, x) => a.add(x), {finalEmit: a => a})`)
		want := reql.Array(reql.Datum(int64(1))).Fold(reql.Datum(int64(0)), reql.Func(reql.Var(1).Add(reql.Var(2)), 1, 2), reql.OptArgs{"final_emit": reql.Func(reql.Var(1), 1)})
		assertTermEqual(t, got, want)
	})
}

// assertWireJSON compares the marshalled term against an exact wire JSON string.
func assertWireJSON(t *testing.T, got reql.Term, want string) {
	t.Helper()
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != want {
		t.Errorf("wire JSON mismatch:\ngot:  %s\nwant: %s", b, want)
	}
}

func TestParse_OptArgs_TermValued(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"orderBy_index_desc_lands_in_optargs_slot",
			`r.table("t").orderBy({index: r.desc("d")})`,
			`[41,[[15,["t"]]],{"index":[74,["d"]]}]`,
		},
		{
			"orderBy_index_string",
			`r.table("t").orderBy({index:"d"})`,
			`[41,[[15,["t"]]],{"index":"d"}]`,
		},
		{
			"between_index_desc",
			`r.table("t").between(1, 2, {index: r.desc("d")})`,
			`[182,[[15,["t"]],1,2],{"index":[74,["d"]]}]`,
		},
		{
			"between_minval_maxval_camel_key",
			`r.table("t").between(r.minval, r.maxval, {index:"d", rightBound:"open"})`,
			`[182,[[15,["t"]],[180,[]],[181,[]]],{"index":"d","right_bound":"open"}]`,
		},
		{
			"changes_datum_optarg",
			`r.table("t").changes({includeInitial: true})`,
			`[152,[[15,["t"]]],{"include_initial":true}]`,
		},
		{
			"getAll_index_expression",
			`r.table("t").getAll("a",{index: r.row("i")})`,
			`[78,[[15,["t"]],"a"],{"index":[170,[[13,[]],"i"]]}]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_OptArgs_MissingValue(t *testing.T) {
	t.Parallel()
	_, err := Parse(`r.table("t").orderBy({index: })`)
	if err == nil {
		t.Fatal("expected error for an optarg without a value")
	}
	if !strings.Contains(err.Error(), "at position") {
		t.Errorf("error %q does not report a byte position", err)
	}
}

func TestParse_OptArgsBacktrack_VarIDs(t *testing.T) {
	t.Parallel()
	t.Run("lambda_before_optargs", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").map(x => x("a")).orderBy({index:"d"})`)
		want := reql.Table("t").Map(reql.Func(reql.Var(1).Bracket("a"), 1)).OrderBy(reql.OptArgs{"index": "d"})
		assertTermEqual(t, got, want)
	})
	t.Run("sibling_lambdas_reuse_var1", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").filter(x => x("a")).orderBy(y => y("b"))`)
		want := reql.Table("t").
			Filter(reql.Func(reql.Var(1).Bracket("a"), 1)).
			OrderBy(reql.Func(reql.Var(1).Bracket("b"), 1))
		assertTermEqual(t, got, want)
	})
	// a failed trailing-optargs attempt parses the inner lambda, then backtracks;
	// the re-parse must allocate the same VAR id, not the next free one
	t.Run("failed_attempt_restores_var_counter", func(t *testing.T) {
		t.Parallel()
		got := mustParse(t, `r.table("t").map(x => x("a").orderBy({index: y => y("b")}, "c"))`)
		inner := reql.Datum(map[string]interface{}{"index": reql.Func(reql.Var(2).Bracket("b"), 2)})
		want := reql.Table("t").Map(reql.Func(reql.Var(1).Bracket("a").OrderBy(inner, "c"), 1))
		assertTermEqual(t, got, want)
	})
}

func TestParse_FieldSelectorChains(t *testing.T) {
	t.Parallel()
	db := `r.db("test").table("users")`
	dbterm := reql.DB("test").Table("users")
	runParseTests(t, []parseTest{
		{
			"pluck_string_only_compat",
			db + `.pluck("a", "b")`,
			dbterm.Pluck("a", "b"),
		},
		{
			"without_nested_object",
			db + `.without({perks: {refill: true}})`,
			dbterm.Without(map[string]interface{}{"perks": map[string]interface{}{"refill": true}}),
		},
		{
			"pluck_mixed_string_object",
			db + `.pluck("name", {address: ["city"]})`,
			dbterm.Pluck("name", map[string]interface{}{"address": reql.Array("city")}),
		},
		{
			"hasFields_object",
			db + `.hasFields({profile: true})`,
			dbterm.HasFields(map[string]interface{}{"profile": true}),
		},
		{
			"withFields_mixed",
			db + `.withFields("id", {stats: true})`,
			dbterm.WithFields("id", map[string]interface{}{"stats": true}),
		},
	})
}

func TestParse_PluckWireJSON(t *testing.T) {
	t.Parallel()
	got := mustParse(t, `r.db("test").table("users").pluck("name", {address: ["city"]})`)
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wire := string(b)
	// array inside field selector object must be MAKE_ARRAY [2,...] so that RethinkDB
	// does not misinterpret ["city"] as a term array (where first element must be a TermType number)
	if !strings.Contains(wire, `{"address":[2,`) {
		t.Errorf("wire JSON must use MAKE_ARRAY {\"address\":[2,...]}: %s", wire)
	}
}

func TestParse_FieldSelectorErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		// number arg rejected
		{"pluck_number_arg", `r.table("t").pluck(123)`, "expected"},
		// bool arg rejected
		{"without_bool_arg", `r.table("t").without(true)`, "expected"},
		// trailing comma rejected
		{"pluck_trailing_comma", `r.table("t").pluck("a",)`, "trailing comma"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestParse_Group(t *testing.T) {
	t.Parallel()
	tbl := reql.Table("t")
	runParseTests(t, []parseTest{
		{
			"single_string_key",
			`r.table("t").group("a")`,
			tbl.Group("a"),
		},
		{
			"multiple_string_keys",
			`r.table("t").group("a","b","c")`,
			tbl.Group("a", "b", "c"),
		},
		{
			"row_key_func_wrapped",
			`r.table("t").group(r.row("a"))`,
			tbl.Group(reql.Func(reql.Var(1).Bracket("a"), 1)),
		},
		{
			"arrow_lambda_key",
			`r.table("t").group(t => t("a"))`,
			tbl.Group(reql.Func(reql.Var(1).Bracket("a"), 1)),
		},
		{
			"function_returning_array",
			`r.table("t").group(function(t){ return [t("a"), t("b")] })`,
			tbl.Group(reql.Func(reql.Array(reql.Var(1).Bracket("a"), reql.Var(1).Bracket("b")), 1)),
		},
		{
			"array_of_row_keys",
			`r.table("t").group([r.row("a"), r.row("b")])`,
			tbl.Group(reql.Func(reql.Array(reql.Var(1).Bracket("a"), reql.Var(1).Bracket("b")), 1)),
		},
		{
			"lambda_and_string_key",
			`r.table("t").group(t => t("a"), "b")`,
			tbl.Group(reql.Func(reql.Var(1).Bracket("a"), 1), "b"),
		},
		{
			"key_with_index_optargs",
			`r.table("t").group("a",{index:"i"})`,
			tbl.Group("a", reql.OptArgs{"index": "i"}),
		},
		{
			"optargs_only",
			`r.table("t").group({multi:true})`,
			tbl.Group(reql.OptArgs{"multi": true}),
		},
		{
			"group_count_ungroup_orderby",
			`r.table("t").group("a").count().ungroup().orderBy(r.desc("reduction"))`,
			tbl.Group("a").Count().Ungroup().OrderBy(reql.Desc("reduction")),
		},
	})
}

func TestParse_Group_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"no_args", `r.table("t").group()`, "group: requires at least one key or an optargs object"},
		{"trailing_comma", `r.table("t").group("a",)`, "trailing comma"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestParse_Aggregate(t *testing.T) {
	t.Parallel()
	tbl := reql.Table("t")
	runParseTests(t, []parseTest{
		{
			"sum_single_string_field",
			`r.table("t").sum("f")`,
			tbl.Sum("f"),
		},
		{
			"avg_single_string_field",
			`r.table("t").avg("f")`,
			tbl.Avg("f"),
		},
		{
			"min_single_string_field",
			`r.table("t").min("f")`,
			tbl.Min("f"),
		},
		{
			"max_single_string_field",
			`r.table("t").max("f")`,
			tbl.Max("f"),
		},
		{
			"min_no_args",
			`r.table("t").min()`,
			tbl.Min(),
		},
		{
			"max_no_args",
			`r.table("t").max()`,
			tbl.Max(),
		},
		{
			"sum_no_args",
			`r.table("t").sum()`,
			tbl.Sum(),
		},
		{
			"avg_no_args",
			`r.table("t").avg()`,
			tbl.Avg(),
		},
		{
			"max_index_optargs",
			`r.table("t").max({index:"createdAt"})`,
			tbl.Max(reql.OptArgs{"index": "createdAt"}),
		},
		{
			"max_index_optargs_bracket_chain",
			`r.table("t").max({index:"createdAt"})("createdAt")`,
			tbl.Max(reql.OptArgs{"index": "createdAt"}).Bracket("createdAt"),
		},
		{
			"min_row_nested_bracket",
			`r.table("t").min(r.row("prices")("USD"))`,
			tbl.Min(reql.Func(reql.Var(1).Bracket("prices").Bracket("USD"), 1)),
		},
		{
			"sum_lambda_in_grouped_stream",
			`r.table("t").group("c").sum(x => x("balance")("amount"))`,
			tbl.Group("c").Sum(reql.Func(reql.Var(1).Bracket("balance").Bracket("amount"), 1)),
		},
		{
			"min_after_map",
			`r.table("t").map(function(t){ return t("date") }).min()`,
			tbl.Map(reql.Func(reql.Var(1).Bracket("date"), 1)).Min(),
		},
		{
			"min_field_with_index_optargs",
			`r.table("t").min("f",{index:"i"})`,
			tbl.Min("f", reql.OptArgs{"index": "i"}),
		},
	})
}

func TestParse_Aggregate_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"avg_two_fields", `r.table("t").avg("a","b")`, "at most one field argument"},
		{"sum_two_fields", `r.table("t").sum("a","b")`, "at most one field argument"},
		{"min_trailing_comma", `r.table("t").min("a",)`, "trailing comma"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestParse_TermValuedArguments(t *testing.T) {
	t.Parallel()
	tbl := reql.Table("t")
	runParseTests(t, []parseTest{
		{
			"bracket_string_key",
			`r.table("t")("field")`,
			tbl.Bracket("field"),
		},
		{
			"bracket_integer_index",
			`r.table("t")(0)`,
			tbl.Nth(0),
		},
		{
			"bracket_lambda_param_key",
			`r.table("t").map(function(f){ return f(f) })`,
			tbl.Map(reql.Func(reql.Var(1).Bracket(reql.Var(1)), 1)),
		},
		{
			"hasFields_lambda_param",
			`r.expr(["a","b"]).map(function(f){ return r.table("t").filter(function(o){ return o.hasFields(f) }).count() })`,
			reql.Array("a", "b").Map(reql.Func(
				tbl.Filter(reql.Func(reql.Var(2).HasFields(reql.Var(1)), 2)).Count(), 1)),
		},
		{
			"getField_lambda_param",
			`r.table("t").map(function(f){ return r.table("u").get(f).getField(f) })`,
			tbl.Map(reql.Func(reql.Table("u").Get(reql.Var(1)).GetField(reql.Var(1)), 1)),
		},
		{
			"match_lambda_param",
			`r.table("t").map(function(f){ return f.match(f) })`,
			tbl.Map(reql.Func(reql.Var(1).Match(reql.Var(1)), 1)),
		},
		{
			"getField_string_key",
			`r.table("t").getField("a")`,
			tbl.GetField("a"),
		},
		{
			"match_string_pattern",
			`r.table("t")("name").match("^a")`,
			tbl.Bracket("name").Match("^a"),
		},
		{
			"hasFields_array_selector",
			`r.table("t").filter(f => f.hasFields(["a","b"]))`,
			tbl.Filter(reql.Func(reql.Var(1).HasFields(reql.Array("a", "b")), 1)),
		},
		{
			"pluck_nested_object_selector",
			`r.table("t").pluck("a",{"p":["b","c"]})`,
			tbl.Pluck("a", map[string]interface{}{"p": reql.Array("b", "c")}),
		},
		{
			"pluck_plain_string",
			`r.table("t").pluck("a")`,
			tbl.Pluck("a"),
		},
		{
			"without_expression_selector",
			`r.table("t").without(r.row("a"))`,
			tbl.Without(reql.Row().Bracket("a")),
		},
		{
			"withFields_array_selector",
			`r.table("t").withFields(["a","b"])`,
			tbl.WithFields(reql.Array("a", "b")),
		},
	})
}

func TestParse_TermValuedArguments_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"bracket_float_index", `r.table("t")(0.5)`, "bracket index must be an integer"},
		{"bracket_empty", `r.table("t")()`, "position"},
		{"getField_no_arg", `r.table("t").getField()`, "position"},
		{"hasFields_trailing_comma", `r.table("t").hasFields("a",)`, "trailing comma"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestCamelToSnake(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"leftBound", "left_bound"},
		{"rightBound", "right_bound"},
		{"returnChanges", "return_changes"},
		{"dryRun", "dry_run"},
		{"includeInitial", "include_initial"},
		{"maxResults", "max_results"},
		{"primaryKey", "primary_key"},
		{"nonVotingReplicaTags", "non_voting_replica_tags"},
		{"left_bound", "left_bound"},
		{"index", "index"},
		{"shards", "shards"},
		{"", ""},
		{"X", "x"},
		{"maxBPS", "max_b_p_s"},
		{"already_snake", "already_snake"},
		{"mixed_Case", "mixed_case"},
	}
	for _, tc := range cases {
		got := camelToSnake(tc.input)
		if got != tc.want {
			t.Errorf("camelToSnake(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestParse_ObjectLiteralWriteDetection guards against a write term hidden
// inside an object-literal value (parseObjectTerm stores parseExpr results,
// including Terms, in a native map wrapped by reql.Datum) evading
// ContainsWrite's read-only gate.
func TestParse_ObjectLiteralWriteDetection(t *testing.T) {
	t.Parallel()
	term, err := Parse(`r.db("d").table("t").filter({a: r.db("d2").table("t2").insert({x: 1})})`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !term.ContainsWrite() {
		t.Fatal("ContainsWrite() = false for write hidden in filter's object-literal predicate, want true")
	}
}

func TestParse_TableListTopLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"top_level_no_args", `r.tableList()`, `[62,[]]`},
		{"db_scoped", `r.db("d").tableList()`, `[62,[[14,["d"]]]]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_BranchChain(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"count_gt_zero",
			`r.table("t").count().gt(0).branch([1],[])`,
			`[65,[[21,[[43,[[15,["t"]]]],0]],[2,[1]],[2,[]]]]`,
		},
		{
			"receiver_is_the_test",
			`r.row("a").branch("yes","no")`,
			`[65,[[170,[[13,[]],"a"]],"yes","no"]]`,
		},
		{
			"multi_condition",
			`r.row("a").branch("x", r.row("b"), "y", "z")`,
			`[65,[[170,[[13,[]],"a"]],"x",[170,[[13,[]],"b"]],"y","z"]]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_SliceVariadic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"one_bound", `r.table("t").slice(-2)`, `[30,[[15,["t"]],-2]]`},
		{"two_bounds", `r.table("t").slice(0,2)`, `[30,[[15,["t"]],0,2]]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_EmptyParamLambda(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"function_no_params",
			`r.table("t").group(function(){ return true })`,
			`[144,[[15,["t"]],[69,[[2,[]],true]]]]`,
		},
		{
			"arrow_no_params",
			`r.table("t").map(() => 1)`,
			`[38,[[15,["t"]],[69,[[2,[]],1]]]]`,
		},
		{
			"single_param_unchanged",
			`r.table("t").filter(x => x("a"))`,
			`[39,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],"a"]]]]]]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_TableListBranchSlice_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"table_list_with_arg", `r.tableList("x")`, "expected ')'"},
		{"branch_even_total", `r.row("a").branch("only")`, "even number of arguments"},
		{"branch_no_args", `r.row("a").branch()`, "even number of arguments"},
		{"slice_no_args", `r.table("t").slice()`, "1 or 2 integer bounds"},
		{"slice_three_bounds", `r.table("t").slice(0,1,2)`, "1 or 2 integer bounds"},
		{"slice_trailing_comma", `r.table("t").slice(0,)`, "trailing comma"},
		{"slice_non_integer", `r.table("t").slice(0.5)`, "expected integer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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

func TestParse_InfixArithmetic(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"add", `r.expr(1+2)`, `[24,[1,2]]`},
		{"sub", `r.expr(1-2)`, `[25,[1,2]]`},
		{"mul_chain_left_associative", `r.expr(60*60*24)`, `[26,[[26,[60,60]],24]]`},
		{"mul_binds_tighter_than_add", `r.expr(1+2*3)`, `[24,[1,[26,[2,3]]]]`},
		{"parentheses_win_over_precedence", `r.expr((1+2)*3)`, `[26,[[24,[1,2]],3]]`},
		{"sub_left_associative", `r.expr(10-2-3)`, `[25,[[25,[10,2]],3]]`},
		{"div_then_mod_left_associative", `r.expr(10/2%3)`, `[28,[[27,[10,2]],3]]`},
		{"negative_literal_after_operator", `r.expr(1 - -2)`, `[25,[1,-2]]`},
		{"folded_mul_chain_as_method_argument", `r.now().sub(60*60*24*30)`, `[25,[[103,[]],[26,[[26,[[26,[60,60]],24]],30]]]]`},
		{
			"between_arithmetic_upper_bound",
			`r.table("t").between(1779222884700, 1779222884700+1, {index:"d"})`,
			`[182,[[15,["t"]],1779222884700,[24,[1779222884700,1]]],{"index":"d"}]`,
		},
		{
			"method_form_add_unchanged",
			`r.table("t").filter(x => x("a").add(1))`,
			`[39,[[15,["t"]],[69,[[2,[1]],[24,[[170,[[10,[1]],"a"]],1]]]]]]`,
		},
		{
			"operands_are_chained_terms",
			`r.table("t").count()+1`,
			`[24,[[43,[[15,["t"]]]],1]]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_InfixArithmetic_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{"missing_right_operand", `r.expr(1+)`},
		{"missing_multiplicative_operand", `r.expr(60*)`},
		{"missing_left_operand", `r.expr(*2)`},
		{"dangling_operator_at_eof", `r.expr(1) +`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("Parse(%q): error %q does not include a byte position", tc.input, err.Error())
			}
		})
	}
}

func TestParse_FunctionLocals(t *testing.T) {
	t.Parallel()
	runParseTests(t, []parseTest{
		{
			"string_local_inlined_into_match",
			`r.table("t").filter(function(p){ var re = "x"; return p("id").match(re) })`,
			reql.Table("t").Filter(reql.Func(reql.Var(1).Bracket("id").Match("x"), 1)),
		},
		{
			"local_bound_to_parameter_expression",
			`function(a){ var b = a("x"); return b.add(1) }`,
			reql.Func(reql.Var(1).Bracket("x").Add(1), 1),
		},
		{
			"local_used_twice_duplicates_subtree",
			`function(a){ var b = a("x"); return b.add(b) }`,
			reql.Func(reql.Var(1).Bracket("x").Add(reql.Var(1).Bracket("x")), 1),
		},
		{
			"multiple_bindings",
			`function(a){ var b = 1; var c = 2; return a("x").add(b).add(c) }`,
			reql.Func(reql.Var(1).Bracket("x").Add(1).Add(2), 1),
		},
		{
			"let_keyword",
			`function(a){ let b = a("x"); return b.add(1) }`,
			reql.Func(reql.Var(1).Bracket("x").Add(1), 1),
		},
		{
			"const_keyword",
			`function(a){ const b = a("x"); return b.add(1) }`,
			reql.Func(reql.Var(1).Bracket("x").Add(1), 1),
		},
		{
			"lambda_bound_to_local",
			`function(a){ var b = function(c){ return c }; return a.map(b) }`,
			reql.Func(reql.Var(1).Map(reql.Func(reql.Var(2), 2)), 1),
		},
		{
			"local_shadows_parameter",
			`function(a){ var a = 1; return a }`,
			reql.Func(reql.Datum(1), 1),
		},
		{
			"local_visible_in_nested_lambda",
			`function(a){ var b = a("x"); return a.map(function(c){ return c.add(b) }) }`,
			reql.Func(reql.Var(1).Map(reql.Func(reql.Var(2).Add(reql.Var(1).Bracket("x")), 2)), 1),
		},
		{
			"local_out_of_scope_after_function",
			`r.table("t").filter(function(p){ var b = 1; return p("a").eq(b) }).map(function(b){ return b("c") })`,
			reql.Table("t").
				Filter(reql.Func(reql.Var(1).Bracket("a").Eq(1), 1)).
				Map(reql.Func(reql.Var(1).Bracket("c"), 1)),
		},
		{
			"body_without_locals_unchanged",
			`function(a){ return a("x") }`,
			reql.Func(reql.Var(1).Bracket("x"), 1),
		},
	})
}

func TestParse_FunctionLocals_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"missing_semicolon", `function(a){ var b = 1 return b }`, "expected ';'"},
		{"reserved_name", `function(a){ var true = 1; return a }`, "reserved word"},
		{"reserved_keyword_name", `function(a){ var const = 1; return a }`, "reserved word"},
		{"missing_assign", `function(a){ var b; return b }`, "expected '='"},
		{"missing_value", `function(a){ var b = ; return b }`, "unexpected token"},
		{"missing_name", `function(a){ var = 1; return a }`, "expected identifier"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("Parse(%q): error %q does not include a byte position", tc.input, err.Error())
			}
		})
	}
}

// TestParse_FunctionLocals_ProductionExpression parses the route-switcher expression
// recorded in the parser error log, whose concatMap body opens with a var binding.
func TestParse_FunctionLocals_ProductionExpression(t *testing.T) {
	t.Parallel()
	const expr = `
r.db("restored").table("routes")
  .getAll("/games/wow/coaching", {index: "url.en"})
  .concatMap(function(route){
    var sw = route("pageConfiguration").default([])
      .filter(function(p){ return p("type").default("").eq("routeSwitcher") })
      .nth(0).default(null);
    return r.branch(
      sw.eq(null),
      [],
      sw("data").default([]).map(function(id){
        return { parentId: route("id"), parentUrl: route("url")("en"), linkedId: id }
      })
    );
  })`
	if _, err := Parse(expr); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestParse_ArrowBlockBody(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			"block_body_returns_object",
			`r.table("t").map(g => { return {a: g("b")} })`,
			`[38,[[15,["t"]],[69,[[2,[1]],{"a":[170,[[10,[1]],"b"]]}]]]]`,
		},
		{
			"block_body_with_local",
			`r.table("t").map(g => { var x = g("b"); return {a: x} })`,
			`[38,[[15,["t"]],[69,[[2,[1]],{"a":[170,[[10,[1]],"b"]]}]]]]`,
		},
		{
			"object_literal_body_unchanged",
			`r.table("t").map(g => {a: g("b")})`,
			`[38,[[15,["t"]],[69,[[2,[1]],{"a":[170,[[10,[1]],"b"]]}]]]]`,
		},
		{
			"parenthesized_object_literal_unchanged",
			`r.table("t").map(g => ({a: g("b")}))`,
			`[38,[[15,["t"]],[69,[[2,[1]],{"a":[170,[[10,[1]],"b"]]}]]]]`,
		},
		{
			"multi_parameter_arrow_with_block_body",
			`r.table("t").map((x, y) => { return x.add(y) })`,
			`[38,[[15,["t"]],[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]]]]`,
		},
		{
			"single_parenthesized_parameter_with_block_body",
			`r.table("t").map((g) => { return g("b") })`,
			`[38,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],"b"]]]]]]`,
		},
		{
			"block_body_without_return_keyword",
			`r.table("t").map(g => { var x = g("b"); x })`,
			`[38,[[15,["t"]],[69,[[2,[1]],[170,[[10,[1]],"b"]]]]]]`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertWireJSON(t, mustParse(t, tc.input), tc.want)
		})
	}
}

func TestParse_ArrowBlockBody_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{"empty_return", `r.table("t").map(g => { return })`},
		{"unterminated_block", `r.table("t").map(g => { return g("b") )`},
		{"local_without_body", `r.table("t").map(g => { var x = 1; })`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("Parse(%q): error %q does not include a byte position", tc.input, err.Error())
			}
		})
	}
}

func TestParse_UnsupportedInputHints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		input    string
		wantMsgs []string
	}{
		{
			"new_date_in_between_bound",
			`r.table("t").between(["A", new Date("2026-06-19T07:40:13.981Z").getTime()], ["A", r.maxval], {index:"i"})`,
			[]string{"new Date()", "r.iso8601", "r.epochTime"},
		},
		{
			"new_date_standalone",
			`new Date()`,
			[]string{"new Date()", "r.iso8601", "r.epochTime"},
		},
		{
			"table_without_r_prefix",
			`table("x").count()`,
			[]string{`unknown identifier "table"`, "r.table(...)"},
		},
		{
			"db_without_r_prefix",
			`db("x").tableList()`,
			[]string{`unknown identifier "db"`, "r.db(...)"},
		},
		{
			"multiple_statements",
			`r.table("x").count(); r.table("y").count()`,
			[]string{"multiple statements", "one query at a time", "--file", "---"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			for _, want := range tc.wantMsgs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), want)
				}
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("Parse(%q): error %q does not include a byte position", tc.input, err.Error())
			}
		})
	}
}

// TestParse_UnknownIdentifier_Generic keeps the generic message for names that are not
// builders, so only a missing r. prefix gets the suggestion.
func TestParse_UnknownIdentifier_Generic(t *testing.T) {
	t.Parallel()
	_, err := Parse(`notAKnownName("x")`)
	if err == nil {
		t.Fatal("Parse: expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unexpected token "notAKnownName"`) {
		t.Errorf("error %q is not the generic unexpected-token error", err.Error())
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("error %q must not suggest an r.* builder", err.Error())
	}
}

// TestParse_ArrowBlockBody_ProductionExpression parses the account-balance report
// recorded in the parser error log, whose map body is an arrow lambda block.
func TestParse_ArrowBlockBody_ProductionExpression(t *testing.T) {
	t.Parallel()
	const expr = `r.table("accounts").filter(t => t("balance")("amount").gt(0)).group("currency").ungroup().` +
		`map(g => {return {currency:g("group"), accounts:g("reduction").count(), ` +
		`total:g("reduction").sum(x=>x("balance")("amount"))}}).orderBy(r.desc("accounts"))`
	if _, err := Parse(expr); err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

func TestParse_AssignToken_Errors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"equality_operator", `r.expr(1==2)`, `expected ')', got "="`},
		{"bare_assign", `=`, `unexpected token "="`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q): expected error, got nil", tc.input)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("Parse(%q): error %q does not contain %q", tc.input, err.Error(), tc.wantMsg)
			}
			if !strings.Contains(err.Error(), "position") {
				t.Errorf("Parse(%q): error %q does not include a byte position", tc.input, err.Error())
			}
		})
	}
}
