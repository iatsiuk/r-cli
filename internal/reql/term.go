package reql

import (
	"encoding/json"
	"errors"

	"r-cli/internal/proto"
)

// Term represents a ReQL expression node.
// termType == 0 means the term is a raw datum (string, number, bool, nil).
type Term struct {
	termType proto.TermType
	datum    interface{}
	args     []Term
	opts     map[string]interface{}
	err      error
}

// errTerm returns a Term that serializes as an error.
func errTerm(err error) Term {
	return Term{err: err}
}

// Row creates an IMPLICIT_VAR term ([13,[]]).
// Used as a shorthand for a single-argument function in methods like Filter.
func Row() Term {
	return Term{termType: proto.TermImplicitVar}
}

// wrapImplicitVar detects IMPLICIT_VAR in t and, if found, replaces all occurrences
// with VAR(1) and wraps the term in FUNC([2,[1]], body).
// Returns unchanged term if no IMPLICIT_VAR is present.
// Returns error if IMPLICIT_VAR appears inside a nested FUNC.
func wrapImplicitVar(t Term) (Term, error) {
	replaced, found, err := replaceImplicit(t, false)
	if err != nil {
		return Term{}, err
	}
	if !found {
		return t, nil
	}
	return Func(replaced, 1), nil
}

// replaceImplicit walks t replacing IMPLICIT_VAR with VAR(1).
// inFunc indicates we are inside a FUNC body; IMPLICIT_VAR there is ambiguous.
// Returns the modified term, whether any replacement was made, and any error.
func replaceImplicit(t Term, inFunc bool) (Term, bool, error) {
	if t.termType == proto.TermImplicitVar {
		if inFunc {
			return Term{}, false, errors.New("reql: IMPLICIT_VAR inside nested function is ambiguous")
		}
		return Var(1), true, nil
	}
	if t.termType == 0 {
		return t, false, t.err
	}
	nested := inFunc || t.termType == proto.TermFunc
	newArgs := make([]Term, len(t.args))
	var anyReplaced bool
	for i, a := range t.args {
		rep, did, err := replaceImplicit(a, nested)
		if err != nil {
			return Term{}, false, err
		}
		newArgs[i] = rep
		if did {
			anyReplaced = true
		}
	}
	if !anyReplaced {
		return t, false, nil
	}
	return Term{
		termType: t.termType,
		datum:    t.datum,
		args:     newArgs,
		opts:     t.opts,
		err:      t.err,
	}, true, nil
}

// Datum wraps a raw Go value as a ReQL term.
func Datum(v interface{}) Term {
	return Term{datum: v}
}

// toTerm converts v to a Term: passes through existing Terms, wraps others in Datum.
func toTerm(v interface{}) Term {
	if t, ok := v.(Term); ok {
		return t
	}
	return Datum(v)
}

// Array creates a MAKE_ARRAY term ([2, [items...]]).
func Array(items ...interface{}) Term {
	args := make([]Term, len(items))
	for i, item := range items {
		args[i] = toTerm(item)
	}
	return Term{termType: proto.TermMakeArray, args: args}
}

// DB creates a DB term ([14, [name]]).
func DB(name string) Term {
	return Term{termType: proto.TermDB, args: []Term{Datum(name)}}
}

// Table creates a TABLE term ([15, [name]]) using the connection-default database.
func Table(name string) Term {
	return Term{termType: proto.TermTable, args: []Term{Datum(name)}}
}

// DBCreate creates a DB_CREATE term ([57, [name]]).
func DBCreate(name string) Term {
	return Term{termType: proto.TermDBCreate, args: []Term{Datum(name)}}
}

// DBDrop creates a DB_DROP term ([58, [name]]).
func DBDrop(name string) Term {
	return Term{termType: proto.TermDBDrop, args: []Term{Datum(name)}}
}

// DBList creates a DB_LIST term ([59, []]).
func DBList() Term {
	return Term{termType: proto.TermDBList}
}

// TableCreate creates a TABLE_CREATE term ([60, [db, name]], opts?) chained on a DB term.
// Optional OptArgs can specify options like {"primary_key": "id"}.
func (t Term) TableCreate(name string, opts ...OptArgs) Term {
	term := Term{termType: proto.TermTableCreate, args: []Term{t, Datum(name)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// TableDrop creates a TABLE_DROP term ([61, [db, name]]) chained on a DB term.
func (t Term) TableDrop(name string) Term {
	return Term{termType: proto.TermTableDrop, args: []Term{t, Datum(name)}}
}

// TableList creates a TABLE_LIST term ([62, [db]]) chained on a DB term.
func (t Term) TableList() Term {
	return Term{termType: proto.TermTableList, args: []Term{t}}
}

// Table creates a TABLE term chained on a DB term ([15, [db, name]]).
func (t Term) Table(name string) Term {
	return Term{termType: proto.TermTable, args: []Term{t, Datum(name)}}
}

// Filter creates a FILTER term ([39, [seq, predicate]]).
// If predicate contains IMPLICIT_VAR (Row()), it is auto-wrapped in FUNC.
func (t Term) Filter(predicate interface{}) Term {
	pt := toTerm(predicate)
	wrapped, err := wrapImplicitVar(pt)
	if err != nil {
		return errTerm(err)
	}
	return Term{termType: proto.TermFilter, args: []Term{t, wrapped}}
}

// Insert creates an INSERT term ([56, [table, doc]], opts?).
// Optional OptArgs can specify options like {"conflict": "replace"}.
func (t Term) Insert(doc interface{}, opts ...OptArgs) Term {
	term := Term{termType: proto.TermInsert, args: []Term{t, toTerm(doc)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Update creates an UPDATE term ([53, [table, doc]]).
func (t Term) Update(doc interface{}, opts ...OptArgs) Term {
	term := Term{termType: proto.TermUpdate, args: []Term{t, toTerm(doc)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Delete creates a DELETE term ([54, [table]], opts?).
func (t Term) Delete(opts ...OptArgs) Term {
	term := Term{termType: proto.TermDelete, args: []Term{t}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Replace creates a REPLACE term ([55, [table, doc]]).
func (t Term) Replace(doc interface{}) Term {
	return Term{termType: proto.TermReplace, args: []Term{t, toTerm(doc)}}
}

// OptArgs is a map of optional arguments passed as the last element to terms like GetAll.
type OptArgs map[string]interface{}

// Get creates a GET term ([16, [table, key]]).
func (t Term) Get(key interface{}) Term {
	return Term{termType: proto.TermGet, args: []Term{t, toTerm(key)}}
}

// GetAll creates a GETALL term ([78, [table, keys...], opts?]).
// The last argument may be an OptArgs to specify options (e.g. {"index": "field"}).
func (t Term) GetAll(args ...interface{}) Term {
	var keys []interface{}
	var opts map[string]interface{}

	if len(args) > 0 {
		if o, ok := args[len(args)-1].(OptArgs); ok {
			opts = map[string]interface{}(o)
			keys = args[:len(args)-1]
		} else {
			keys = args
		}
	}

	if len(keys) == 0 {
		return errTerm(errors.New("reql: GetAll requires at least one key"))
	}
	termArgs := []Term{t}
	for _, k := range keys {
		if kt, ok := k.(Term); ok {
			termArgs = append(termArgs, kt)
		} else {
			termArgs = append(termArgs, Datum(k))
		}
	}
	return Term{termType: proto.TermGetAll, args: termArgs, opts: opts}
}

// Between creates a BETWEEN term ([182, [term, lower, upper]], opts?).
func (t Term) Between(lower, upper interface{}, opts ...OptArgs) Term {
	term := Term{termType: proto.TermBetween, args: []Term{t, toTerm(lower), toTerm(upper)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Asc creates an ASC term ([73, [field]]) for use with OrderBy.
func Asc(field string) Term {
	return Term{termType: proto.TermAsc, args: []Term{Datum(field)}}
}

// Desc creates a DESC term ([74, [field]]) for use with OrderBy.
func Desc(field string) Term {
	return Term{termType: proto.TermDesc, args: []Term{Datum(field)}}
}

// OrderBy creates an ORDERBY term ([41, [term, fields...]], opts?).
// The last argument may be an OptArgs to specify options (e.g. {"index": "field"}).
func (t Term) OrderBy(fields ...interface{}) Term {
	var opts map[string]interface{}
	termFields := fields
	if len(fields) > 0 {
		if o, ok := fields[len(fields)-1].(OptArgs); ok {
			opts = map[string]interface{}(o)
			termFields = fields[:len(fields)-1]
		}
	}
	args := []Term{t}
	for _, f := range termFields {
		if ft, ok := f.(Term); ok {
			args = append(args, ft)
		} else {
			args = append(args, Datum(f))
		}
	}
	return Term{termType: proto.TermOrderBy, args: args, opts: opts}
}

// Limit creates a LIMIT term ([71, [term, n]]).
func (t Term) Limit(n int) Term {
	return Term{termType: proto.TermLimit, args: []Term{t, Datum(n)}}
}

// Skip creates a SKIP term ([70, [term, n]]).
func (t Term) Skip(n int) Term {
	return Term{termType: proto.TermSkip, args: []Term{t, Datum(n)}}
}

// Sample creates a SAMPLE term ([81, [term, n]]).
func (t Term) Sample(n int) Term {
	return Term{termType: proto.TermSample, args: []Term{t, Datum(n)}}
}

// Count creates a COUNT term ([43, [term]]).
func (t Term) Count() Term {
	return Term{termType: proto.TermCount, args: []Term{t}}
}

// Pluck creates a PLUCK term ([33, [term, fields...]]).
func (t Term) Pluck(fields ...string) Term {
	args := make([]Term, 0, 1+len(fields))
	args = append(args, t)
	for _, f := range fields {
		args = append(args, Datum(f))
	}
	return Term{termType: proto.TermPluck, args: args}
}

// Without creates a WITHOUT term ([34, [term, fields...]]).
func (t Term) Without(fields ...string) Term {
	args := make([]Term, 0, 1+len(fields))
	args = append(args, t)
	for _, f := range fields {
		args = append(args, Datum(f))
	}
	return Term{termType: proto.TermWithout, args: args}
}

// GetField creates a GET_FIELD term ([31, [term, field]]).
func (t Term) GetField(field string) Term {
	return Term{termType: proto.TermGetField, args: []Term{t, Datum(field)}}
}

// HasFields creates a HAS_FIELDS term ([32, [term, fields...]]).
func (t Term) HasFields(fields ...string) Term {
	args := make([]Term, 0, 1+len(fields))
	args = append(args, t)
	for _, f := range fields {
		args = append(args, Datum(f))
	}
	return Term{termType: proto.TermHasFields, args: args}
}

// Merge creates a MERGE term ([35, [term, obj]]).
func (t Term) Merge(obj interface{}) Term {
	return Term{termType: proto.TermMerge, args: []Term{t, toTerm(obj)}}
}

// Distinct creates a DISTINCT term ([42, [term]]).
func (t Term) Distinct() Term {
	return Term{termType: proto.TermDistinct, args: []Term{t}}
}

// Map creates a MAP term ([38, [term, func]]).
func (t Term) Map(fn Term) Term {
	return Term{termType: proto.TermMap, args: []Term{t, fn}}
}

// Reduce creates a REDUCE term ([37, [term, func]]).
func (t Term) Reduce(fn Term) Term {
	return Term{termType: proto.TermReduce, args: []Term{t, fn}}
}

// Group creates a GROUP term ([144, [term, field]]).
func (t Term) Group(field string) Term {
	return Term{termType: proto.TermGroup, args: []Term{t, Datum(field)}}
}

// Ungroup creates an UNGROUP term ([150, [term]]).
func (t Term) Ungroup() Term {
	return Term{termType: proto.TermUngroup, args: []Term{t}}
}

// Sum creates a SUM term ([145, [term, field]]).
func (t Term) Sum(field string) Term {
	return Term{termType: proto.TermSum, args: []Term{t, Datum(field)}}
}

// Avg creates an AVG term ([146, [term, field]]).
func (t Term) Avg(field string) Term {
	return Term{termType: proto.TermAvg, args: []Term{t, Datum(field)}}
}

// Min creates a MIN term ([147, [term, field]]).
func (t Term) Min(field string) Term {
	return Term{termType: proto.TermMin, args: []Term{t, Datum(field)}}
}

// Max creates a MAX term ([148, [term, field]]).
func (t Term) Max(field string) Term {
	return Term{termType: proto.TermMax, args: []Term{t, Datum(field)}}
}

// Eq creates an EQ term ([17, [term, value]]).
func (t Term) Eq(value interface{}) Term {
	return t.binop(proto.TermEq, value)
}

// Ne creates a NE term ([18, [term, value]]).
func (t Term) Ne(value interface{}) Term {
	return t.binop(proto.TermNe, value)
}

// Lt creates a LT term ([19, [term, value]]).
func (t Term) Lt(value interface{}) Term {
	return t.binop(proto.TermLt, value)
}

// Le creates a LE term ([20, [term, value]]).
func (t Term) Le(value interface{}) Term {
	return t.binop(proto.TermLe, value)
}

// Gt creates a GT term ([21, [term, value]]).
func (t Term) Gt(value interface{}) Term {
	return t.binop(proto.TermGt, value)
}

// Ge creates a GE term ([22, [term, value]]).
func (t Term) Ge(value interface{}) Term {
	return t.binop(proto.TermGe, value)
}

// Not creates a NOT term ([23, [term]]).
func (t Term) Not() Term {
	return Term{termType: proto.TermNot, args: []Term{t}}
}

// And creates an AND term ([67, [term, other]]).
func (t Term) And(other Term) Term {
	return Term{termType: proto.TermAnd, args: []Term{t, other}}
}

// Or creates an OR term ([66, [term, other]]).
func (t Term) Or(other Term) Term {
	return Term{termType: proto.TermOr, args: []Term{t, other}}
}

// Add creates an ADD term ([24, [term, value]]).
func (t Term) Add(value interface{}) Term {
	return t.binop(proto.TermAdd, value)
}

// Sub creates a SUB term ([25, [term, value]]).
func (t Term) Sub(value interface{}) Term {
	return t.binop(proto.TermSub, value)
}

// Mul creates a MUL term ([26, [term, value]]).
func (t Term) Mul(value interface{}) Term {
	return t.binop(proto.TermMul, value)
}

// Div creates a DIV term ([27, [term, value]]).
func (t Term) Div(value interface{}) Term {
	return t.binop(proto.TermDiv, value)
}

// Mod creates a MOD term ([28, [term, value]]).
func (t Term) Mod(value interface{}) Term {
	return t.binop(proto.TermMod, value)
}

// Floor creates a FLOOR term ([183, [term]]).
func (t Term) Floor() Term {
	return Term{termType: proto.TermFloor, args: []Term{t}}
}

// Ceil creates a CEIL term ([184, [term]]).
func (t Term) Ceil() Term {
	return Term{termType: proto.TermCeil, args: []Term{t}}
}

// Round creates a ROUND term ([185, [term]]).
func (t Term) Round() Term {
	return Term{termType: proto.TermRound, args: []Term{t}}
}

// IndexCreate creates an INDEX_CREATE term ([75, [table, name]], opts?).
// Optional OptArgs can specify options like {"geo": true, "multi": true}.
func (t Term) IndexCreate(name string, opts ...OptArgs) Term {
	term := Term{termType: proto.TermIndexCreate, args: []Term{t, Datum(name)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// IndexDrop creates an INDEX_DROP term ([76, [table, name]]).
func (t Term) IndexDrop(name string) Term {
	return Term{termType: proto.TermIndexDrop, args: []Term{t, Datum(name)}}
}

// IndexList creates an INDEX_LIST term ([77, [table]]).
func (t Term) IndexList() Term {
	return Term{termType: proto.TermIndexList, args: []Term{t}}
}

// indexOp builds an index operation term with optional index names.
func (t Term) indexOp(tt proto.TermType, names []string) Term {
	args := make([]Term, 1, 1+len(names))
	args[0] = t
	for _, n := range names {
		args = append(args, Datum(n))
	}
	return Term{termType: tt, args: args}
}

// IndexWait creates an INDEX_WAIT term ([140, [table, names...]]).
func (t Term) IndexWait(names ...string) Term { return t.indexOp(proto.TermIndexWait, names) }

// IndexStatus creates an INDEX_STATUS term ([139, [table, names...]]).
func (t Term) IndexStatus(names ...string) Term { return t.indexOp(proto.TermIndexStatus, names) }

// IndexRename creates an INDEX_RENAME term ([156, [table, old, new]]).
func (t Term) IndexRename(oldName, newName string) Term {
	return Term{termType: proto.TermIndexRename, args: []Term{t, Datum(oldName), Datum(newName)}}
}

// Var creates a VAR term ([10, [id]]) referencing a function parameter.
func Var(id int) Term {
	return Term{termType: proto.TermVar, args: []Term{Datum(id)}}
}

// Func creates a FUNC term ([69, [[2, [param_ids...]], body]]).
// params are the integer parameter IDs; body is the function body term.
func Func(body Term, params ...int) Term {
	paramTerms := make([]Term, len(params))
	for i, p := range params {
		paramTerms[i] = Datum(p)
	}
	paramArray := Term{termType: proto.TermMakeArray, args: paramTerms}
	return Term{termType: proto.TermFunc, args: []Term{paramArray, body}}
}

// Changes creates a CHANGES term ([152, [term]], opts?).
// Optional OptArgs can specify options like {"include_initial": true}.
func (t Term) Changes(opts ...OptArgs) Term {
	term := Term{termType: proto.TermChanges, args: []Term{t}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Now creates a NOW term ([103, []]).
func Now() Term {
	return Term{termType: proto.TermNow}
}

// UUID creates a UUID term ([169, []]).
func UUID() Term {
	return Term{termType: proto.TermUUID}
}

// Binary creates a BINARY term ([155, [data]]).
func Binary(data interface{}) Term {
	return Term{termType: proto.TermBinary, args: []Term{toTerm(data)}}
}

// Config creates a CONFIG term ([174, [term]]).
func (t Term) Config() Term {
	return Term{termType: proto.TermConfig, args: []Term{t}}
}

// Status creates a STATUS term ([175, [term]]).
func (t Term) Status() Term {
	return Term{termType: proto.TermStatus, args: []Term{t}}
}

// Grant creates a GRANT term ([188, [scope, user, perms]]).
func (t Term) Grant(user string, perms interface{}) Term {
	return Term{termType: proto.TermGrant, args: []Term{t, Datum(user), toTerm(perms)}}
}

// Grant creates a global GRANT term ([188, [user, perms]]) with no scope.
func Grant(user string, perms interface{}) Term {
	return Term{termType: proto.TermGrant, args: []Term{Datum(user), toTerm(perms)}}
}

// Do creates a FUNCALL term ([64, [fn, t]]) -- chain form.
// Equivalent to reql.Do(t, fn): applies fn to the current term.
func (t Term) Do(fn Term) Term {
	return Term{termType: proto.TermFuncCall, args: []Term{fn, t}}
}

// Do creates a FUNCALL term ([64, [fn, args...]]).
// API order: Do(arg1, arg2, ..., fn) - function is the last argument.
// Wire order: [64, [fn, arg1, arg2, ...]] - function goes first on the wire.
func Do(args ...interface{}) Term {
	if len(args) == 0 {
		return errTerm(errors.New("reql: Do requires at least a function argument"))
	}
	fn := toTerm(args[len(args)-1])
	wireArgs := make([]Term, 1, len(args))
	wireArgs[0] = fn
	for _, a := range args[:len(args)-1] {
		wireArgs = append(wireArgs, toTerm(a))
	}
	return Term{termType: proto.TermFuncCall, args: wireArgs}
}

// InnerJoin creates an INNER_JOIN term ([48, [seq, other, fn]]).
func (t Term) InnerJoin(other, fn Term) Term {
	return Term{termType: proto.TermInnerJoin, args: []Term{t, other, fn}}
}

// OuterJoin creates an OUTER_JOIN term ([49, [seq, other, fn]]).
func (t Term) OuterJoin(other, fn Term) Term {
	return Term{termType: proto.TermOuterJoin, args: []Term{t, other, fn}}
}

// EqJoin creates an EQ_JOIN term ([50, [seq, field, table]], opts?).
// Optional OptArgs can specify options like {"index": "name"}.
func (t Term) EqJoin(field string, table Term, opts ...OptArgs) Term {
	term := Term{termType: proto.TermEqJoin, args: []Term{t, Datum(field), table}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Zip creates a ZIP term ([72, [term]]).
func (t Term) Zip() Term {
	return Term{termType: proto.TermZip, args: []Term{t}}
}

// Match creates a MATCH term ([97, [term, "pattern"]]).
func (t Term) Match(pattern string) Term {
	return Term{termType: proto.TermMatch, args: []Term{t, Datum(pattern)}}
}

// Split creates a SPLIT term ([149, [term]] or [149, [term, "delim"]]).
func (t Term) Split(delim ...string) Term {
	if len(delim) == 0 {
		return Term{termType: proto.TermSplit, args: []Term{t}}
	}
	return Term{termType: proto.TermSplit, args: []Term{t, Datum(delim[0])}}
}

// Upcase creates an UPCASE term ([141, [term]]).
func (t Term) Upcase() Term {
	return Term{termType: proto.TermUpcase, args: []Term{t}}
}

// Downcase creates a DOWNCASE term ([142, [term]]).
func (t Term) Downcase() Term {
	return Term{termType: proto.TermDowncase, args: []Term{t}}
}

// ToJSONString creates a TO_JSON_STRING term ([172, [term]]).
func (t Term) ToJSONString() Term {
	return Term{termType: proto.TermToJSONString, args: []Term{t}}
}

// JSON creates a JSON term ([98, ["json_string"]]).
func JSON(s string) Term {
	return Term{termType: proto.TermJSON, args: []Term{Datum(s)}}
}

// ISO8601 creates an ISO8601 term ([99, [<iso_string>]]).
func ISO8601(s string) Term {
	return Term{termType: proto.TermISO8601, args: []Term{Datum(s)}}
}

// EpochTime creates an EPOCH_TIME term ([101, [<epoch>]]).
func EpochTime(epoch interface{}) Term {
	return Term{termType: proto.TermEpochTime, args: []Term{toTerm(epoch)}}
}

// Time creates a TIME term ([136, [year, month, day, timezone]]).
func Time(year, month, day int, timezone string) Term {
	return Term{
		termType: proto.TermTime,
		args:     []Term{Datum(year), Datum(month), Datum(day), Datum(timezone)},
	}
}

// TimeAt creates a TIME term with time-of-day ([136, [year, month, day, hour, minute, second, timezone]]).
func TimeAt(year, month, day, hour, minute, second int, timezone string) Term {
	return Term{
		termType: proto.TermTime,
		args: []Term{
			Datum(year), Datum(month), Datum(day),
			Datum(hour), Datum(minute), Datum(second),
			Datum(timezone),
		},
	}
}

// ToISO8601 creates a TO_ISO8601 term ([100, [<time_term>]]).
func (t Term) ToISO8601() Term {
	return Term{termType: proto.TermToISO8601, args: []Term{t}}
}

// ToEpochTime creates a TO_EPOCH_TIME term ([102, [<time_term>]]).
func (t Term) ToEpochTime() Term {
	return Term{termType: proto.TermToEpochTime, args: []Term{t}}
}

// Date creates a DATE term ([106, [<time_term>]]).
func (t Term) Date() Term {
	return Term{termType: proto.TermDate, args: []Term{t}}
}

// TimeOfDay creates a TIME_OF_DAY term ([126, [<time_term>]]).
func (t Term) TimeOfDay() Term {
	return Term{termType: proto.TermTimeOfDay, args: []Term{t}}
}

// Timezone creates a TIMEZONE term ([127, [<time_term>]]).
func (t Term) Timezone() Term {
	return Term{termType: proto.TermTimezone, args: []Term{t}}
}

// Year creates a YEAR term ([128, [<time_term>]]).
func (t Term) Year() Term {
	return Term{termType: proto.TermYear, args: []Term{t}}
}

// Month creates a MONTH term ([129, [<time_term>]]).
func (t Term) Month() Term {
	return Term{termType: proto.TermMonth, args: []Term{t}}
}

// Day creates a DAY term ([130, [<time_term>]]).
func (t Term) Day() Term {
	return Term{termType: proto.TermDay, args: []Term{t}}
}

// DayOfWeek creates a DAY_OF_WEEK term ([131, [<time_term>]]).
func (t Term) DayOfWeek() Term {
	return Term{termType: proto.TermDayOfWeek, args: []Term{t}}
}

// DayOfYear creates a DAY_OF_YEAR term ([132, [<time_term>]]).
func (t Term) DayOfYear() Term {
	return Term{termType: proto.TermDayOfYear, args: []Term{t}}
}

// Hours creates an HOURS term ([133, [<time_term>]]).
func (t Term) Hours() Term {
	return Term{termType: proto.TermHours, args: []Term{t}}
}

// Minutes creates a MINUTES term ([134, [<time_term>]]).
func (t Term) Minutes() Term {
	return Term{termType: proto.TermMinutes, args: []Term{t}}
}

// Seconds creates a SECONDS term ([135, [<time_term>]]).
func (t Term) Seconds() Term {
	return Term{termType: proto.TermSeconds, args: []Term{t}}
}

// InTimezone creates an IN_TIMEZONE term ([104, [<time_term>, <tz>]]).
func (t Term) InTimezone(tz string) Term {
	return Term{termType: proto.TermInTimezone, args: []Term{t, Datum(tz)}}
}

// During creates a DURING term ([105, [<time_term>, <start>, <end>]]).
func (t Term) During(start, end Term) Term {
	return Term{termType: proto.TermDuring, args: []Term{t, start, end}}
}

// Monday creates a MONDAY constant term ([107, []]).
func Monday() Term { return Term{termType: proto.TermMonday} }

// Tuesday creates a TUESDAY constant term ([108, []]).
func Tuesday() Term { return Term{termType: proto.TermTuesday} }

// Wednesday creates a WEDNESDAY constant term ([109, []]).
func Wednesday() Term { return Term{termType: proto.TermWednesday} }

// Thursday creates a THURSDAY constant term ([110, []]).
func Thursday() Term { return Term{termType: proto.TermThursday} }

// Friday creates a FRIDAY constant term ([111, []]).
func Friday() Term { return Term{termType: proto.TermFriday} }

// Saturday creates a SATURDAY constant term ([112, []]).
func Saturday() Term { return Term{termType: proto.TermSaturday} }

// Sunday creates a SUNDAY constant term ([113, []]).
func Sunday() Term { return Term{termType: proto.TermSunday} }

// January creates a JANUARY constant term ([114, []]).
func January() Term { return Term{termType: proto.TermJanuary} }

// February creates a FEBRUARY constant term ([115, []]).
func February() Term { return Term{termType: proto.TermFebruary} }

// March creates a MARCH constant term ([116, []]).
func March() Term { return Term{termType: proto.TermMarch} }

// April creates an APRIL constant term ([117, []]).
func April() Term { return Term{termType: proto.TermApril} }

// May creates a MAY constant term ([118, []]).
func May() Term { return Term{termType: proto.TermMay} }

// June creates a JUNE constant term ([119, []]).
func June() Term { return Term{termType: proto.TermJune} }

// July creates a JULY constant term ([120, []]).
func July() Term { return Term{termType: proto.TermJuly} }

// August creates an AUGUST constant term ([121, []]).
func August() Term { return Term{termType: proto.TermAugust} }

// September creates a SEPTEMBER constant term ([122, []]).
func September() Term { return Term{termType: proto.TermSeptember} }

// October creates an OCTOBER constant term ([123, []]).
func October() Term { return Term{termType: proto.TermOctober} }

// November creates a NOVEMBER constant term ([124, []]).
func November() Term { return Term{termType: proto.TermNovember} }

// December creates a DECEMBER constant term ([125, []]).
func December() Term { return Term{termType: proto.TermDecember} }

// Append creates an APPEND term ([29, [term, value]]).
func (t Term) Append(value interface{}) Term {
	return Term{termType: proto.TermAppend, args: []Term{t, toTerm(value)}}
}

// Prepend creates a PREPEND term ([80, [term, value]]).
func (t Term) Prepend(value interface{}) Term {
	return Term{termType: proto.TermPrepend, args: []Term{t, toTerm(value)}}
}

// Slice creates a SLICE term ([30, [term, start, end]]).
func (t Term) Slice(start, end int) Term {
	return Term{termType: proto.TermSlice, args: []Term{t, Datum(start), Datum(end)}}
}

// Difference creates a DIFFERENCE term ([95, [term, array]]).
func (t Term) Difference(other Term) Term {
	return Term{termType: proto.TermDifference, args: []Term{t, other}}
}

// InsertAt creates an INSERT_AT term ([82, [term, index, value]]).
func (t Term) InsertAt(index int, value interface{}) Term {
	return Term{termType: proto.TermInsertAt, args: []Term{t, Datum(index), toTerm(value)}}
}

// DeleteAt creates a DELETE_AT term ([83, [term, index]]).
func (t Term) DeleteAt(index int) Term {
	return Term{termType: proto.TermDeleteAt, args: []Term{t, Datum(index)}}
}

// ChangeAt creates a CHANGE_AT term ([84, [term, index, value]]).
func (t Term) ChangeAt(index int, value interface{}) Term {
	return Term{termType: proto.TermChangeAt, args: []Term{t, Datum(index), toTerm(value)}}
}

// SpliceAt creates a SPLICE_AT term ([85, [term, index, array]]).
func (t Term) SpliceAt(index int, array Term) Term {
	return Term{termType: proto.TermSpliceAt, args: []Term{t, Datum(index), array}}
}

// SetInsert creates a SET_INSERT term ([88, [term, value]]).
func (t Term) SetInsert(value interface{}) Term {
	return Term{termType: proto.TermSetInsert, args: []Term{t, toTerm(value)}}
}

// SetIntersection creates a SET_INTERSECTION term ([89, [term, array]]).
func (t Term) SetIntersection(other Term) Term {
	return Term{termType: proto.TermSetIntersection, args: []Term{t, other}}
}

// SetUnion creates a SET_UNION term ([90, [term, array]]).
func (t Term) SetUnion(other Term) Term {
	return Term{termType: proto.TermSetUnion, args: []Term{t, other}}
}

// SetDifference creates a SET_DIFFERENCE term ([91, [term, array]]).
func (t Term) SetDifference(other Term) Term {
	return Term{termType: proto.TermSetDifference, args: []Term{t, other}}
}

// Info creates an INFO term ([79, [term]]).
func (t Term) Info() Term {
	return Term{termType: proto.TermInfo, args: []Term{t}}
}

// OffsetsOf creates an OFFSETS_OF term ([87, [seq, pred]]).
func (t Term) OffsetsOf(predicate interface{}) Term {
	return Term{termType: proto.TermOffsetsOf, args: []Term{t, toTerm(predicate)}}
}

// Fold creates a FOLD term ([187, [seq, base, fn], opts?]).
func (t Term) Fold(base, fn Term, opts ...OptArgs) Term {
	term := Term{termType: proto.TermFold, args: []Term{t, base, fn}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Branch creates a BRANCH term ([65, [cond, true_val, false_val, ...]]).
// Accepts 3+ arguments: cond1, val1, ..., else_val (supports multi-condition form).
func Branch(args ...interface{}) Term {
	if len(args) < 3 {
		return errTerm(errors.New("reql: Branch requires at least 3 arguments"))
	}
	if len(args)%2 == 0 {
		return errTerm(errors.New("reql: Branch requires an odd number of arguments"))
	}
	termArgs := make([]Term, len(args))
	for i, a := range args {
		termArgs[i] = toTerm(a)
	}
	return Term{termType: proto.TermBranch, args: termArgs}
}

// ForEach creates a FOR_EACH term ([68, [seq, fn]]).
func (t Term) ForEach(fn Term) Term {
	return Term{termType: proto.TermForEach, args: []Term{t, fn}}
}

// Default creates a DEFAULT term ([92, [term, default_val]]).
func (t Term) Default(val interface{}) Term {
	return Term{termType: proto.TermDefault, args: []Term{t, toTerm(val)}}
}

// Error creates an ERROR term ([12, [message]]).
func Error(msg string) Term {
	return Term{termType: proto.TermError, args: []Term{Datum(msg)}}
}

// CoerceTo creates a COERCE_TO term ([51, [term, type_name]]).
func (t Term) CoerceTo(typeName string) Term {
	return Term{termType: proto.TermCoerceTo, args: []Term{t, Datum(typeName)}}
}

// TypeOf creates a TYPE_OF term ([52, [term]]).
func (t Term) TypeOf() Term {
	return Term{termType: proto.TermTypeOf, args: []Term{t}}
}

// ConcatMap creates a CONCAT_MAP term ([40, [seq, fn]]).
func (t Term) ConcatMap(fn Term) Term {
	return Term{termType: proto.TermConcatMap, args: []Term{t, fn}}
}

// Nth creates an NTH term ([45, [seq, index]]).
func (t Term) Nth(index int) Term {
	return Term{termType: proto.TermNth, args: []Term{t, Datum(index)}}
}

// Union creates a UNION term ([44, [seq, seqs...]]).
func (t Term) Union(seqs ...Term) Term {
	args := make([]Term, 1, 1+len(seqs))
	args[0] = t
	args = append(args, seqs...)
	return Term{termType: proto.TermUnion, args: args}
}

// IsEmpty creates an IS_EMPTY term ([86, [seq]]).
func (t Term) IsEmpty() Term {
	return Term{termType: proto.TermIsEmpty, args: []Term{t}}
}

// Contains creates a CONTAINS term ([93, [seq, values...]]).
func (t Term) Contains(values ...interface{}) Term {
	if len(values) == 0 {
		return errTerm(errors.New("reql: Contains requires at least one value"))
	}
	args := make([]Term, 1, 1+len(values))
	args[0] = t
	for _, v := range values {
		args = append(args, toTerm(v))
	}
	return Term{termType: proto.TermContains, args: args}
}

// Bracket creates a BRACKET term ([170, [term, field]]).
func (t Term) Bracket(field string) Term {
	return Term{termType: proto.TermBracket, args: []Term{t, Datum(field)}}
}

// WithFields creates a WITH_FIELDS term ([96, [seq, fields...]]).
func (t Term) WithFields(fields ...string) Term {
	args := make([]Term, 1, 1+len(fields))
	args[0] = t
	for _, f := range fields {
		args = append(args, Datum(f))
	}
	return Term{termType: proto.TermWithFields, args: args}
}

// Keys creates a KEYS term ([94, [term]]).
func (t Term) Keys() Term {
	return Term{termType: proto.TermKeys, args: []Term{t}}
}

// Values creates a VALUES term ([186, [term]]).
func (t Term) Values() Term {
	return Term{termType: proto.TermValues, args: []Term{t}}
}

// Literal creates a LITERAL term ([137, [value]]).
func Literal(value interface{}) Term {
	return Term{termType: proto.TermLiteral, args: []Term{toTerm(value)}}
}

// Sync creates a SYNC term ([138, [table_term]]).
func (t Term) Sync() Term {
	return Term{termType: proto.TermSync, args: []Term{t}}
}

// Reconfigure creates a RECONFIGURE term ([176, [table_term]], opts?).
func (t Term) Reconfigure(opts ...OptArgs) Term {
	term := Term{termType: proto.TermReconfigure, args: []Term{t}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Rebalance creates a REBALANCE term ([179, [table_term]]).
func (t Term) Rebalance() Term {
	return Term{termType: proto.TermRebalance, args: []Term{t}}
}

// Wait creates a WAIT term ([177, [table_term]]).
func (t Term) Wait() Term {
	return Term{termType: proto.TermWait, args: []Term{t}}
}

// Args creates an ARGS term ([154, [array]]).
func Args(array Term) Term {
	return Term{termType: proto.TermArgs, args: []Term{array}}
}

// MinVal creates a MINVAL term ([180, []]).
func MinVal() Term {
	return Term{termType: proto.TermMinVal}
}

// MaxVal creates a MAXVAL term ([181, []]).
func MaxVal() Term {
	return Term{termType: proto.TermMaxVal}
}

// GeoJSON creates a GEOJSON term ([157, [obj]]).
func GeoJSON(obj interface{}) Term {
	return Term{termType: proto.TermGeoJSON, args: []Term{toTerm(obj)}}
}

// ToGeoJSON creates a TO_GEOJSON term ([158, [geo_term]]).
func (t Term) ToGeoJSON() Term {
	return Term{termType: proto.TermToGeoJSON, args: []Term{t}}
}

// Point creates a POINT term ([159, [lon, lat]]).
func Point(lon, lat float64) Term {
	return Term{termType: proto.TermPoint, args: []Term{Datum(lon), Datum(lat)}}
}

// Line creates a LINE term ([160, [point1, point2, ...]]).
// Requires at least 2 points.
func Line(points ...Term) Term {
	if len(points) < 2 {
		return errTerm(errors.New("reql: Line requires at least 2 points"))
	}
	args := make([]Term, len(points))
	copy(args, points)
	return Term{termType: proto.TermLine, args: args}
}

// Polygon creates a POLYGON term ([161, [point1, point2, ...]]).
// Requires at least 3 points.
func Polygon(points ...Term) Term {
	if len(points) < 3 {
		return errTerm(errors.New("reql: Polygon requires at least 3 points"))
	}
	args := make([]Term, len(points))
	copy(args, points)
	return Term{termType: proto.TermPolygon, args: args}
}

// Circle creates a CIRCLE term ([165, [center, radius]], opts?).
func Circle(center Term, radius float64, opts ...OptArgs) Term {
	term := Term{termType: proto.TermCircle, args: []Term{center, Datum(radius)}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Distance creates a DISTANCE term ([162, [geo1, geo2]], opts?).
func (t Term) Distance(other Term, opts ...OptArgs) Term {
	term := Term{termType: proto.TermDistance, args: []Term{t, other}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Intersects creates an INTERSECTS term ([163, [geo1, geo2]]).
func (t Term) Intersects(other Term) Term {
	return Term{termType: proto.TermIntersects, args: []Term{t, other}}
}

// Includes creates an INCLUDES term ([164, [geo, point]]).
func (t Term) Includes(point Term) Term {
	return Term{termType: proto.TermIncludes, args: []Term{t, point}}
}

// GetIntersecting creates a GET_INTERSECTING term ([166, [table, geo]], opts?).
func (t Term) GetIntersecting(geo Term, opts ...OptArgs) Term {
	term := Term{termType: proto.TermGetIntersecting, args: []Term{t, geo}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// GetNearest creates a GET_NEAREST term ([168, [table, point]], opts?).
func (t Term) GetNearest(point Term, opts ...OptArgs) Term {
	term := Term{termType: proto.TermGetNearest, args: []Term{t, point}}
	if len(opts) > 0 {
		term.opts = opts[0]
	}
	return term
}

// Fill creates a FILL term ([167, [line_term]]).
func (t Term) Fill() Term {
	return Term{termType: proto.TermFill, args: []Term{t}}
}

// PolygonSub creates a POLYGON_SUB term ([171, [polygon1, polygon2]]).
func (t Term) PolygonSub(other Term) Term {
	return Term{termType: proto.TermPolygonSub, args: []Term{t, other}}
}

// Object creates an OBJECT term ([143, [k, v, ...]]).
// Requires an even number of arguments (key-value pairs).
func Object(pairs ...interface{}) Term {
	if len(pairs)%2 != 0 {
		return errTerm(errors.New("reql: Object requires an even number of arguments (key-value pairs)"))
	}
	args := make([]Term, len(pairs))
	for i, p := range pairs {
		args[i] = toTerm(p)
	}
	return Term{termType: proto.TermObject, args: args}
}

// Range creates a RANGE term ([173, [start?, end?]]).
// Accepts 0, 1, or 2 arguments.
func Range(args ...interface{}) Term {
	if len(args) > 2 {
		return errTerm(errors.New("reql: Range accepts 0, 1, or 2 arguments"))
	}
	termArgs := make([]Term, len(args))
	for i, a := range args {
		termArgs[i] = toTerm(a)
	}
	return Term{termType: proto.TermRange, args: termArgs}
}

// Random creates a RANDOM term ([151, [...], opts?]).
// Accepts 0, 1, or 2 numeric arguments plus optional OptArgs.
func Random(args ...interface{}) Term {
	var opts map[string]interface{}
	termArgs := args
	if len(args) > 0 {
		if o, ok := args[len(args)-1].(OptArgs); ok {
			opts = map[string]interface{}(o)
			termArgs = args[:len(args)-1]
		}
	}
	if len(termArgs) > 2 {
		return errTerm(errors.New("reql: Random accepts 0, 1, or 2 numeric arguments"))
	}
	argTerms := make([]Term, len(termArgs))
	for i, a := range termArgs {
		argTerms[i] = toTerm(a)
	}
	return Term{termType: proto.TermRandom, args: argTerms, opts: opts}
}

// BitAnd creates a BIT_AND term ([191, [val, n]]).
func (t Term) BitAnd(n interface{}) Term {
	return t.binop(proto.TermBitAnd, n)
}

// BitOr creates a BIT_OR term ([192, [val, n]]).
func (t Term) BitOr(n interface{}) Term {
	return t.binop(proto.TermBitOr, n)
}

// BitXor creates a BIT_XOR term ([193, [val, n]]).
func (t Term) BitXor(n interface{}) Term {
	return t.binop(proto.TermBitXor, n)
}

// BitNot creates a BIT_NOT term ([194, [val]]).
func (t Term) BitNot() Term {
	return Term{termType: proto.TermBitNot, args: []Term{t}}
}

// BitSal creates a BIT_SAL term ([195, [val, n]]).
func (t Term) BitSal(n interface{}) Term {
	return t.binop(proto.TermBitSal, n)
}

// BitSar creates a BIT_SAR term ([196, [val, n]]).
func (t Term) BitSar(n interface{}) Term {
	return t.binop(proto.TermBitSar, n)
}

// binop builds a binary term [type, [t, value]].
func (t Term) binop(tt proto.TermType, value interface{}) Term {
	return Term{termType: tt, args: []Term{t, toTerm(value)}}
}

// MarshalJSON serializes the term to ReQL wire format.
// Datum terms serialize as their raw value; compound terms as [type, [args...], opts?].
func (t Term) MarshalJSON() ([]byte, error) {
	if t.err != nil {
		return nil, t.err
	}
	if t.termType == 0 {
		return json.Marshal(t.datum)
	}
	args := t.args
	if args == nil {
		args = []Term{}
	}
	parts := []interface{}{int(t.termType), args}
	if len(t.opts) > 0 {
		parts = append(parts, t.opts)
	}
	return json.Marshal(parts)
}
