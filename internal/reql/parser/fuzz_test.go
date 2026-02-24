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
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		// must not panic; errors are fine
		_, _ = Parse(input)
	})
}
