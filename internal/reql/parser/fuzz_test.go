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
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// must not panic; errors are fine
		_, _ = Parse(input)
	})
}
