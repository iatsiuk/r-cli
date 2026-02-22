package repl

import (
	"context"
	"strings"
	"sync"
	"time"
)

// TabCompleter is implemented by types that provide readline tab completion.
type TabCompleter interface {
	Do(line []rune, pos int) (newLine [][]rune, length int)
}

// Completer provides ReQL tab completion for the REPL.
// FetchDBs and FetchTables are optional; if nil, dynamic completion is disabled.
type Completer struct {
	FetchDBs    func(ctx context.Context) ([]string, error)
	FetchTables func(ctx context.Context, db string) ([]string, error)
	mu          sync.RWMutex
	currentDB   string
}

// SetCurrentDB updates the current database used for table name completion.
// Safe to call concurrently with tab completion.
func (c *Completer) SetCurrentDB(db string) {
	c.mu.Lock()
	c.currentDB = db
	c.mu.Unlock()
}

func (c *Completer) getDB() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentDB
}

// top-level r.* method names (from parser rBuilders map, sorted)
var topLevelMethods = []string{
	"args", "asc", "branch", "db", "dbCreate", "dbDrop", "dbList",
	"desc", "epochTime", "error", "expr", "geoJSON", "iso8601",
	"json", "literal", "maxval", "minval", "now", "point", "row",
	"table", "uuid",
}

// chainable method names (from parser chainBuilders map, sorted)
var chainMethods = []string{
	"add", "and", "append", "avg", "between", "ceil", "changeAt",
	"changes", "coerceTo", "concatMap", "config", "contains", "count",
	"date", "day", "dayOfWeek", "dayOfYear", "default", "delete",
	"deleteAt", "difference", "distinct", "distance", "div", "downcase",
	"during", "eq", "eqJoin", "fill", "filter", "floor", "forEach",
	"ge", "get", "getAll", "getField", "getIntersecting", "getNearest",
	"grant", "group", "gt", "hasFields", "hours", "includes",
	"indexCreate", "indexDrop", "indexList", "indexRename", "indexStatus",
	"indexWait", "innerJoin", "insert", "insertAt", "intersects",
	"inTimezone", "isEmpty", "keys", "le", "limit", "lt", "map",
	"match", "max", "merge", "min", "minutes", "mod", "month", "mul",
	"ne", "not", "nth", "or", "orderBy", "outerJoin", "pluck",
	"polygonSub", "prepend", "rebalance", "reconfigure", "reduce",
	"replace", "round", "seconds", "setDifference", "setInsert",
	"setIntersection", "setUnion", "skip", "slice", "spliceAt", "split",
	"status", "sub", "sum", "sync", "table", "tableCreate", "tableDrop",
	"tableList", "timeOfDay", "timezone", "toEpochTime", "toGeoJSON",
	"toISO8601", "toJSONString", "typeOf", "ungroup", "union", "update",
	"upcase", "values", "wait", "withFields", "without", "year", "zip",
}

// Do implements TabCompleter and readline.AutoCompleter.
// Returns completion candidates and how many chars to remove before the cursor.
func (c *Completer) Do(line []rune, pos int) (newLine [][]rune, length int) {
	s := string(line[:pos])

	// string-literal completions (checked before identifier completions)
	if partial, ok := stringArg(s, "db"); ok {
		return filterCompletions(c.fetchDBNames(), partial), len(partial)
	}
	if partial, ok := stringArg(s, "table"); ok {
		return filterCompletions(c.fetchTableNames(), partial), len(partial)
	}

	// identifier completions
	before, word := identEnd(s)
	if isTopLevelPrefix(before) {
		return filterCompletions(topLevelMethods, word), len(word)
	}
	if strings.HasSuffix(before, ".") {
		return filterCompletions(chainMethods, word), len(word)
	}

	return nil, 0
}

// isTopLevelPrefix reports whether before ends with "r." not preceded by an ident char.
func isTopLevelPrefix(before string) bool {
	if !strings.HasSuffix(before, "r.") {
		return false
	}
	if len(before) > 2 && isIdentByte(before[len(before)-3]) {
		return false
	}
	return true
}

// stringArg reports whether s ends with method(["']partial where partial has no closing quote.
func stringArg(s, method string) (string, bool) {
	for _, q := range []string{`"`, `'`} {
		tok := method + "(" + q
		idx := strings.LastIndex(s, tok)
		if idx < 0 {
			continue
		}
		rest := s[idx+len(tok):]
		if !strings.ContainsAny(rest, `"'`) {
			return rest, true
		}
	}
	return "", false
}

// identEnd splits s into (before, word) where word is the trailing identifier.
func identEnd(s string) (before, word string) {
	i := len(s)
	for i > 0 && isIdentByte(s[i-1]) {
		i--
	}
	return s[:i], s[i:]
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// filterCompletions returns suffix completions (readline appends them to what's already typed).
func filterCompletions(candidates []string, prefix string) [][]rune {
	var result [][]rune
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			result = append(result, []rune(c[len(prefix):]))
		}
	}
	return result
}

func (c *Completer) fetchDBNames() []string {
	if c.FetchDBs == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	names, _ := c.FetchDBs(ctx)
	return names
}

func (c *Completer) fetchTableNames() []string {
	if c.FetchTables == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	names, _ := c.FetchTables(ctx, c.getDB())
	return names
}
