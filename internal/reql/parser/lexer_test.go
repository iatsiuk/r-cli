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

func TestLexer_AssignToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []tv
	}{
		{
			"var_binding",
			`var x = 1`,
			[]tv{
				{tokenIdent, "var"},
				{tokenIdent, "x"},
				{tokenAssign, "="},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
		{
			"alone",
			`=`,
			[]tv{{tokenAssign, "="}, {tokenEOF, ""}},
		},
		{
			"double_equal",
			`==`,
			[]tv{{tokenAssign, "="}, {tokenAssign, "="}, {tokenEOF, ""}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			assertTokens(t, got, tc.want)
		})
	}
}

func TestLexer_ArithmeticOperators(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []tv
	}{
		{
			"add",
			`1+2`,
			[]tv{{tokenNumber, "1"}, {tokenPlus, "+"}, {tokenNumber, "2"}, {tokenEOF, ""}},
		},
		{
			"subtract_after_ident",
			`a-2`,
			[]tv{{tokenIdent, "a"}, {tokenMinus, "-"}, {tokenNumber, "2"}, {tokenEOF, ""}},
		},
		{
			"mixed_precedence_operators",
			`60*60*24/2%7`,
			[]tv{
				{tokenNumber, "60"},
				{tokenStar, "*"},
				{tokenNumber, "60"},
				{tokenStar, "*"},
				{tokenNumber, "24"},
				{tokenSlash, "/"},
				{tokenNumber, "2"},
				{tokenPercent, "%"},
				{tokenNumber, "7"},
				{tokenEOF, ""},
			},
		},
		{
			"subtract_after_rparen",
			`x(0)-1`,
			[]tv{
				{tokenIdent, "x"},
				{tokenLParen, "("},
				{tokenNumber, "0"},
				{tokenRParen, ")"},
				{tokenMinus, "-"},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
		{
			"subtract_after_rbracket",
			`x[0]-1`,
			[]tv{
				{tokenIdent, "x"},
				{tokenLBracket, "["},
				{tokenNumber, "0"},
				{tokenRBracket, "]"},
				{tokenMinus, "-"},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
		{
			"subtract_after_rbrace",
			`x}-1`,
			[]tv{
				{tokenIdent, "x"},
				{tokenRBrace, "}"},
				{tokenMinus, "-"},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
		{
			"subtract_after_string",
			`"a"-1`,
			[]tv{{tokenString, "a"}, {tokenMinus, "-"}, {tokenNumber, "1"}, {tokenEOF, ""}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			assertTokens(t, got, tc.want)
		})
	}
}

func TestLexer_NegativeNumberLiterals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  []tv
	}{
		{
			"input_start",
			`-5`,
			[]tv{{tokenNumber, "-5"}, {tokenEOF, ""}},
		},
		{
			"call_argument",
			`f(-2)`,
			[]tv{
				{tokenIdent, "f"},
				{tokenLParen, "("},
				{tokenNumber, "-2"},
				{tokenRParen, ")"},
				{tokenEOF, ""},
			},
		},
		{
			"array_element",
			`[1,-2]`,
			[]tv{
				{tokenLBracket, "["},
				{tokenNumber, "1"},
				{tokenComma, ","},
				{tokenNumber, "-2"},
				{tokenRBracket, "]"},
				{tokenEOF, ""},
			},
		},
		{
			"after_return_keyword",
			`return -1`,
			[]tv{{tokenIdent, "return"}, {tokenNumber, "-1"}, {tokenEOF, ""}},
		},
		{
			"after_assign",
			`var x = -1`,
			[]tv{
				{tokenIdent, "var"},
				{tokenIdent, "x"},
				{tokenAssign, "="},
				{tokenNumber, "-1"},
				{tokenEOF, ""},
			},
		},
		{
			"after_arrow",
			`x => -1`,
			[]tv{
				{tokenIdent, "x"},
				{tokenArrow, "=>"},
				{tokenNumber, "-1"},
				{tokenEOF, ""},
			},
		},
		{
			"after_operator",
			`1--1`,
			[]tv{
				{tokenNumber, "1"},
				{tokenMinus, "-"},
				{tokenNumber, "-1"},
				{tokenEOF, ""},
			},
		},
		{
			"after_bool",
			`true-1`,
			[]tv{
				{tokenBool, "true"},
				{tokenMinus, "-"},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
		{
			"after_null",
			`null-1`,
			[]tv{
				{tokenNull, "null"},
				{tokenMinus, "-"},
				{tokenNumber, "1"},
				{tokenEOF, ""},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tokenizeOrFail(t, tc.input)
			assertTokens(t, got, tc.want)
		})
	}
}

func TestLexer_ArrowNoRegression(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `x => y`)
	want := []tv{
		{tokenIdent, "x"},
		{tokenArrow, "=>"},
		{tokenIdent, "y"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
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

func TestLexer_FunctionWithSemicolon(t *testing.T) {
	t.Parallel()
	got := tokenizeOrFail(t, `function(x){ return x; }`)
	want := []tv{
		{tokenIdent, "function"},
		{tokenLParen, "("},
		{tokenIdent, "x"},
		{tokenRParen, ")"},
		{tokenLBrace, "{"},
		{tokenIdent, "return"},
		{tokenIdent, "x"},
		{tokenSemicolon, ";"},
		{tokenRBrace, "}"},
		{tokenEOF, ""},
	}
	assertTokens(t, got, want)
}
