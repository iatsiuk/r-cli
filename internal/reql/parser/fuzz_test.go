package parser

import "testing"

func FuzzLex(f *testing.F) {
	// seed corpus from representative test cases
	seeds := []string{
		`r.db("test")`,
		`r.db("test").table("users")`,
		`r.db("test").table("users").filter({name: "foo"})`,
		`r.row("field").gt(21)`,
		`r.minval`,
		`r.maxval`,
		`r.branch(r.row("x").gt(0), "pos", "neg")`,
		`[1, 2, 3]`,
		`{name: "foo", age: 42}`,
		`r.expr([1,2,3])`,
		`r.epochTime(1234567890)`,
		`r.point(-122.4, 37.7)`,
		``,
		`!!!`,
		`r.db(`,
		`=>`,
		`(x) => x`,
		`= `,
		`a=>b`,
		`===`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// must not panic; errors are fine
		l := newLexer(input)
		_, _ = l.tokenize()
	})
}

func FuzzParse(f *testing.F) {
	seeds := []string{
		`r.db("test")`,
		`r.db("test").table("users")`,
		`r.db("test").table("users").filter({name: "foo"})`,
		`r.row("field").gt(21)`,
		`r.minval`,
		`r.maxval`,
		`r.branch(r.row("x").gt(0), "pos", "neg")`,
		`r.error("msg")`,
		`r.args([r.minval, r.maxval])`,
		`r.expr([1, 2, 3])`,
		`r.point(-122.4, 37.7)`,
		`r.epochTime(1234567890)`,
		`r.db("test").table("users").limit(10)`,
		`r.db("test").table("users").orderBy(r.desc("name"))`,
		`r.db("test").table("users").eqJoin("id", r.table("other"))`,
		``,
		`r.db(`,
		`r.unknownThing()`,
		`42 extra`,
		`(x) => x`,
		`(a,b) => a.add(b)`,
		`x => x`,
		`() => 1`,
		`(x) => (y) => y`,
		`=> x`,
		`(x) =>`,
		`function(x){ return x }`,
		`function(a,b){ return a.add(b) }`,
		`function(r){ return r('f') }`,
		`function(){ return 1 }`,
		`function(x){}`,
		`function(x){ return }`,
		`function x`,
		`function`,
		`(r) => r('f')`,
		// bracket numeric index seeds
		`r.table("t")(0)`,
		`r.table("t")(0)("f")`,
		`r.table("t")(-1)`,
		`r.table("t")(0.5)`,
		// sample seeds
		`r.table("t").sample(1)`,
		`r.table("t").sample(5).pluck("id")`,
		// insert/update/delete optargs seeds
		`r.table("t").insert({a:1},{return_changes:true})`,
		`r.table("t").insert({a:1},{conflict:"replace"})`,
		`r.table("t").update({x:1},{durability:"soft"})`,
		`r.table("t").delete({durability:"soft"})`,
		`r.table("t").insert({a:1},)`,
		// parenthesized expression seeds
		`row => ({a: row("b")})`,
		`(x) => ({id: x("id")})`,
		`(x) => ({a: x("x"), b: x("y")})`,
		`=> ({})`,
		`(()`,
		// nested function seeds
		`function(a){ return function(b){ return b } }`,
		`(x) => (y) => y("f")`,
		`function(x){ return (y) => y("f") }`,
		`(x) => function(y){ return y("f") }`,
		`function(x){ return function(y){ return function(z){ return z } } }`,
		// group and aggregation seeds
		`r.table("t").group("a","b")`,
		`r.table("t").group(x => x("a"))`,
		`r.table("t").group([r.row("a"), r.row("b")])`,
		`r.table("t").group("a",{index:"i"})`,
		`r.table("t").group()`,
		`r.table("t").group(function(){ return true })`,
		`r.table("t").min()`,
		`r.table("t").max({index:"d"})`,
		`r.table("t").sum(x => x("v"))`,
		`r.table("t").avg("a","b")`,
		// small-addition seeds
		`r.tableList()`,
		`r.tableList("x")`,
		`r.table("t").slice(-2)`,
		`r.table("t").slice()`,
		`r.row("a").branch(1,2)`,
		`r.row("a").branch("only")`,
		`r.table("t")(0)(0)`,
		// arithmetic seeds
		`r.expr(1+2*3)`,
		`r.expr(60*60*24*30)`,
		`r.expr((1+2)*3)`,
		`r.expr(10/2%3)`,
		`r.expr(1 - -2)`,
		`r.expr(1+)`,
		`r.expr(-)`,
		`r.expr(*2)`,
		`1--1`,
		`r.expr(1==2)`,
		// local variable and block body seeds
		`function(a){ var b = 1; return b }`,
		`function(a){ var b = a("x"); return b.add(b) }`,
		`function(a){ let b = 1; const c = 2; return b.add(c) }`,
		`function(a){ var b = ; return b }`,
		`function(a){ var b = 1 return b }`,
		`function(a){ var true = 1; return a }`,
		`g => { return {a: 1} }`,
		`g => { var x = g("b"); return {a: x} }`,
		`g => {a: g("b")}`,
		`g => { return }`,
		`g => { var x = 1; }`,
		// term-valued optargs and field selector seeds
		`r.table("t").orderBy({index: r.desc("d")})`,
		`r.table("t").orderBy({index: })`,
		`r.table("t").between(1, 2, {index: r.desc("d")})`,
		`r.table("t").getAll("a",{index: r.row("i")})`,
		`r.table("t").filter(f => f.hasFields(["a"]))`,
		`r.table("t").map(function(f){ return f(f) })`,
		// unsupported-input hint seeds
		`new Date("2026-06-19T07:40:13.981Z").getTime()`,
		`table("x").count()`,
		`r.table("x").count(); r.table("y").count()`,
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// must not panic; errors are fine
		_, _ = Parse(input)
	})
}
