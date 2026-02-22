package repl

import (
	"context"
	"strings"
	"testing"
)

// ensure Completer satisfies TabCompleter
var _ TabCompleter = (*Completer)(nil)

func TestCompleterTopLevelMethods(t *testing.T) {
	t.Parallel()
	c := &Completer{}

	// "r." -> all top-level methods, length 0
	line := []rune("r.")
	got, length := c.Do(line, len(line))
	if length != 0 {
		t.Errorf("r.: length = %d, want 0", length)
	}
	if len(got) != len(topLevelMethods) {
		t.Errorf("r.: got %d completions, want %d", len(got), len(topLevelMethods))
	}

	// "r.db" -> methods starting with "db", length 2
	line = []rune("r.db")
	got, length = c.Do(line, len(line))
	if length != 2 {
		t.Errorf("r.db: length = %d, want 2", length)
	}
	wantDB := []string{"db", "dbCreate", "dbDrop", "dbList"}
	for _, w := range wantDB {
		if !containsCompletion(got, w) {
			t.Errorf("r.db: missing %q in %v", w, toStringSlice(got))
		}
	}
	if len(got) != len(wantDB) {
		t.Errorf("r.db: got %d completions, want %d: %v", len(got), len(wantDB), toStringSlice(got))
	}

	// "r.now" -> exactly "now"
	line = []rune("r.now")
	got, length = c.Do(line, len(line))
	if length != 3 {
		t.Errorf("r.now: length = %d, want 3", length)
	}
	if len(got) != 1 || string(got[0]) != "now" {
		t.Errorf("r.now: got %v, want [now]", toStringSlice(got))
	}
}

func TestCompleterChainMethods(t *testing.T) {
	t.Parallel()
	c := &Completer{}

	// "r.table(\"t\")." -> all chain methods, length 0
	line := []rune(`r.table("t").`)
	got, length := c.Do(line, len(line))
	if length != 0 {
		t.Errorf("chain dot: length = %d, want 0", length)
	}
	if len(got) != len(chainMethods) {
		t.Errorf("chain dot: got %d completions, want %d", len(got), len(chainMethods))
	}
	for _, m := range []string{"filter", "get", "insert", "count", "orderBy"} {
		if !containsCompletion(got, m) {
			t.Errorf("chain dot: missing %q", m)
		}
	}

	// "r.table(\"t\").fil" -> methods starting with "fil", length 3
	line = []rune(`r.table("t").fil`)
	got, length = c.Do(line, len(line))
	if length != 3 {
		t.Errorf("chain fil: length = %d, want 3", length)
	}
	if !containsCompletion(got, "filter") {
		t.Errorf("chain fil: missing 'filter' in %v", toStringSlice(got))
	}
	for _, g := range got {
		if !strings.HasPrefix(string(g), "fil") {
			t.Errorf("chain fil: completion %q does not start with 'fil'", string(g))
		}
	}
}

func TestCompleterDBNames(t *testing.T) {
	t.Parallel()
	c := &Completer{
		FetchDBs: func(_ context.Context) ([]string, error) {
			return []string{"test", "rethinkdb", "myapp"}, nil
		},
	}

	// `r.db("` -> all db names, length 0
	line := []rune(`r.db("`)
	got, length := c.Do(line, len(line))
	if length != 0 {
		t.Errorf(`r.db(": length = %d, want 0`, length)
	}
	for _, w := range []string{"test", "rethinkdb", "myapp"} {
		if !containsCompletion(got, w) {
			t.Errorf(`r.db(": missing %q in %v`, w, toStringSlice(got))
		}
	}
	if len(got) != 3 {
		t.Errorf(`r.db(": got %d completions, want 3`, len(got))
	}

	// `r.db("tes` -> only "test", length 3
	line = []rune(`r.db("tes`)
	got, length = c.Do(line, len(line))
	if length != 3 {
		t.Errorf(`r.db("tes: length = %d, want 3`, length)
	}
	if len(got) != 1 || string(got[0]) != "test" {
		t.Errorf(`r.db("tes: got %v, want [test]`, toStringSlice(got))
	}

	// `r.db("test")` -> no completion (string is closed)
	line = []rune(`r.db("test")`)
	got, _ = c.Do(line, len(line))
	if len(got) != 0 {
		t.Errorf(`r.db("test"): expected no completions, got %v`, toStringSlice(got))
	}
}

func TestCompleterTableNames(t *testing.T) {
	t.Parallel()
	c := &Completer{
		CurrentDB: "test",
		FetchTables: func(_ context.Context, db string) ([]string, error) {
			if db == "test" {
				return []string{"heroes", "planets", "users"}, nil
			}
			return nil, nil
		},
	}

	// `.table("` -> all table names, length 0
	line := []rune(`.table("`)
	got, length := c.Do(line, len(line))
	if length != 0 {
		t.Errorf(`.table(": length = %d, want 0`, length)
	}
	if len(got) != 3 {
		t.Errorf(`.table(": got %d completions, want 3: %v`, len(got), toStringSlice(got))
	}

	// `r.db("test").table("her` -> filtered, length 3
	line = []rune(`r.db("test").table("her`)
	got, length = c.Do(line, len(line))
	if length != 3 {
		t.Errorf(`table("her: length = %d, want 3`, length)
	}
	if len(got) != 1 || string(got[0]) != "heroes" {
		t.Errorf(`table("her: got %v, want [heroes]`, toStringSlice(got))
	}

	// `r.table("` -> also matches table string arg
	line = []rune(`r.table("`)
	got, length = c.Do(line, len(line))
	if length != 0 {
		t.Errorf(`r.table(": length = %d, want 0`, length)
	}
	if len(got) != 3 {
		t.Errorf(`r.table(": got %d completions, want 3`, len(got))
	}
}

func TestCompleterNoMatch(t *testing.T) {
	t.Parallel()
	c := &Completer{}

	// no dot: no completion
	line := []rune("hello")
	got, _ := c.Do(line, len(line))
	if len(got) != 0 {
		t.Errorf("no dot: expected no completions, got %v", toStringSlice(got))
	}

	// closed expression: no completion
	line = []rune(`r.now()`)
	got, _ = c.Do(line, len(line))
	if len(got) != 0 {
		t.Errorf("r.now(): expected no completions, got %v", toStringSlice(got))
	}
}

func containsCompletion(completions [][]rune, s string) bool {
	for _, c := range completions {
		if string(c) == s {
			return true
		}
	}
	return false
}

func toStringSlice(rss [][]rune) []string {
	ss := make([]string, len(rss))
	for i, rs := range rss {
		ss[i] = string(rs)
	}
	return ss
}
