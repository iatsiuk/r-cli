package parser

import (
	"testing"
)

// tv is a compact token descriptor for test assertions.
type tv struct {
	t tokenType
	v string
}

func tokenizeOrFail(t *testing.T, input string) []tv {
	t.Helper()
	l := newLexer(input)
	toks, err := l.tokenize()
	if err != nil {
		t.Fatalf("tokenize(%q) error: %v", input, err)
	}
	out := make([]tv, len(toks))
	for i, tok := range toks {
		out[i] = tv{tok.Type, tok.Value}
	}
	return out
}

func assertTokens(t *testing.T, got, want []tv) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("token count: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i, w := range want {
		g := got[i]
		if g.t != w.t || g.v != w.v {
			t.Errorf("token[%d]: got {%d %q}, want {%d %q}", i, g.t, g.v, w.t, w.v)
		}
	}
}

func TestLexer_RDbCall(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `r.db("test")`)
	want := []tv{
		{tokenIdent, "r"},
		{tokenDot, "."},
		{tokenIdent, "db"},
		{tokenLParen, "("},
		{tokenString, "test"},
		{tokenRParen, ")"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_NumbersBoolsNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  []tv
	}{
		{"42", []tv{{tokenNumber, "42"}, {tokenEOF, ""}}},
		{"3.14", []tv{{tokenNumber, "3.14"}, {tokenEOF, ""}}},
		{"-7", []tv{{tokenNumber, "-7"}, {tokenEOF, ""}}},
		{"-122.4", []tv{{tokenNumber, "-122.4"}, {tokenEOF, ""}}},
		{"1e10", []tv{{tokenNumber, "1e10"}, {tokenEOF, ""}}},
		{"true", []tv{{tokenBool, "true"}, {tokenEOF, ""}}},
		{"false", []tv{{tokenBool, "false"}, {tokenEOF, ""}}},
		{"null", []tv{{tokenNull, "null"}, {tokenEOF, ""}}},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			assertTokens(t, got, tc.want)
		})
	}
}

func TestLexer_ObjectLiteral(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `{name: "foo", age: 42}`)
	want := []tv{
		{tokenLBrace, "{"},
		{tokenIdent, "name"},
		{tokenColon, ":"},
		{tokenString, "foo"},
		{tokenComma, ","},
		{tokenIdent, "age"},
		{tokenColon, ":"},
		{tokenNumber, "42"},
		{tokenRBrace, "}"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_ArrayLiteral(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `[1, 2, 3]`)
	want := []tv{
		{tokenLBracket, "["},
		{tokenNumber, "1"},
		{tokenComma, ","},
		{tokenNumber, "2"},
		{tokenComma, ","},
		{tokenNumber, "3"},
		{tokenRBracket, "]"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_ChainedMethods(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `.table("x").filter({})`)
	want := []tv{
		{tokenDot, "."},
		{tokenIdent, "table"},
		{tokenLParen, "("},
		{tokenString, "x"},
		{tokenRParen, ")"},
		{tokenDot, "."},
		{tokenIdent, "filter"},
		{tokenLParen, "("},
		{tokenLBrace, "{"},
		{tokenRBrace, "}"},
		{tokenRParen, ")"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_SingleQuotedString(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `'foo'`)
	want := []tv{
		{tokenString, "foo"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_SingleQuotedStringWithEscape(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `'it\'s'`)
	want := []tv{
		{tokenString, "it's"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_MinvalMaxval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  []tv
	}{
		{
			`r.minval`,
			[]tv{
				{tokenIdent, "r"},
				{tokenDot, "."},
				{tokenIdent, "minval"},
				{tokenEOF, ""},
			},
		},
		{
			`r.maxval`,
			[]tv{
				{tokenIdent, "r"},
				{tokenDot, "."},
				{tokenIdent, "maxval"},
				{tokenEOF, ""},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			assertTokens(t, got, tc.want)
		})
	}
}

func TestLexer_SignedExponents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"1e+10", "1e+10"},
		{"1e-10", "1e-10"},
		{"2.5e+3", "2.5e+3"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			if len(got) != 2 || got[0].t != tokenNumber || got[0].v != tc.want {
				t.Errorf("tokenize(%q): got %v, want first token {number %q}", tc.input, got, tc.want)
			}
		})
	}
}

func TestLexer_EscapeSequences(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{`"\n"`, "\n"},
		{`"\t"`, "\t"},
		{`"\r"`, "\r"},
		{`"\\"`, "\\"},
		{`"\""`, `"`},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			if len(got) != 2 || got[0].t != tokenString || got[0].v != tc.want {
				t.Errorf("tokenize(%q): got %v, want string %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLexer_UnknownEscapeError(t *testing.T) {
	t.Parallel()
	l := newLexer(`"\q"`)
	_, err := l.tokenize()
	if err == nil {
		t.Fatal("expected error for unknown escape '\\q', got nil")
	}
}

func TestLexer_UnexpectedCharError(t *testing.T) {
	t.Parallel()
	l := newLexer("@foo")
	_, err := l.tokenize()
	if err == nil {
		t.Fatal("expected error for '@', got nil")
	}
}

func TestLexer_UnterminatedStringError(t *testing.T) {
	t.Parallel()
	l := newLexer(`"unterminated`)
	_, err := l.tokenize()
	if err == nil {
		t.Fatal("expected error for unterminated string, got nil")
	}
}

func TestLexer_ArrowToken(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `=>`)
	want := []tv{
		{tokenArrow, "=>"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_ArrowInLambda(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `(x) => x`)
	want := []tv{
		{tokenLParen, "("},
		{tokenIdent, "x"},
		{tokenRParen, ")"},
		{tokenArrow, "=>"},
		{tokenIdent, "x"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_EqualAloneError(t *testing.T) {
	t.Parallel()
	l := newLexer(`=`)
	_, err := l.tokenize()
	if err == nil {
		t.Fatal("expected error for '=' alone, got nil")
	}
}

func TestLexer_DoubleEqualError(t *testing.T) {
	t.Parallel()
	l := newLexer(`==`)
	_, err := l.tokenize()
	if err == nil {
		t.Fatal("expected error for '==', got nil")
	}
}

func TestLexer_FunctionKeyword(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `function(x){ return x }`)
	want := []tv{
		{tokenIdent, "function"},
		{tokenLParen, "("},
		{tokenIdent, "x"},
		{tokenRParen, ")"},
		{tokenLBrace, "{"},
		{tokenIdent, "return"},
		{tokenIdent, "x"},
		{tokenRBrace, "}"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}

func TestLexer_FunctionEmpty(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `function(){}`)
	want := []tv{
		{tokenIdent, "function"},
		{tokenLParen, "("},
		{tokenRParen, ")"},
		{tokenLBrace, "{"},
		{tokenRBrace, "}"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}
