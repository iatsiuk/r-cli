package reql

import (
	"encoding/json"

	"r-cli/internal/proto"
)

// Term represents a ReQL expression node.
// termType == 0 means the term is a raw datum (string, number, bool, nil).
type Term struct {
	termType proto.TermType
	datum    interface{}
	args     []Term
	opts     map[string]interface{}
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
		if t, ok := item.(Term); ok {
			args[i] = t
		} else {
			args[i] = Datum(item)
		}
	}
	return Term{termType: proto.TermMakeArray, args: args}
}

// DB creates a DB term ([14, [name]]).
func DB(name string) Term {
	return Term{termType: proto.TermDB, args: []Term{Datum(name)}}
}

// Table creates a TABLE term chained on a DB term ([15, [db, name]]).
func (t Term) Table(name string) Term {
	return Term{termType: proto.TermTable, args: []Term{t, Datum(name)}}
}

// Filter creates a FILTER term ([39, [seq, predicate]]).
// predicate can be a Term or any value that marshals to a JSON document.
func (t Term) Filter(predicate interface{}) Term {
	return Term{termType: proto.TermFilter, args: []Term{t, toTerm(predicate)}}
}

// Insert creates an INSERT term ([56, [table, doc]]).
func (t Term) Insert(doc interface{}) Term {
	return Term{termType: proto.TermInsert, args: []Term{t, toTerm(doc)}}
}

// Update creates an UPDATE term ([53, [table, doc]]).
func (t Term) Update(doc interface{}) Term {
	return Term{termType: proto.TermUpdate, args: []Term{t, toTerm(doc)}}
}

// Delete creates a DELETE term ([54, [table]]).
func (t Term) Delete() Term {
	return Term{termType: proto.TermDelete, args: []Term{t}}
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

// Between creates a BETWEEN term ([182, [term, lower, upper]]).
func (t Term) Between(lower, upper interface{}) Term {
	return Term{termType: proto.TermBetween, args: []Term{t, toTerm(lower), toTerm(upper)}}
}

// Asc creates an ASC term ([73, [field]]) for use with OrderBy.
func Asc(field string) Term {
	return Term{termType: proto.TermAsc, args: []Term{Datum(field)}}
}

// Desc creates a DESC term ([74, [field]]) for use with OrderBy.
func Desc(field string) Term {
	return Term{termType: proto.TermDesc, args: []Term{Datum(field)}}
}

// OrderBy creates an ORDERBY term ([41, [term, fields...]]).
func (t Term) OrderBy(fields ...interface{}) Term {
	args := []Term{t}
	for _, f := range fields {
		if ft, ok := f.(Term); ok {
			args = append(args, ft)
		} else {
			args = append(args, Datum(f))
		}
	}
	return Term{termType: proto.TermOrderBy, args: args}
}

// Limit creates a LIMIT term ([71, [term, n]]).
func (t Term) Limit(n int) Term {
	return Term{termType: proto.TermLimit, args: []Term{t, Datum(n)}}
}

// Skip creates a SKIP term ([70, [term, n]]).
func (t Term) Skip(n int) Term {
	return Term{termType: proto.TermSkip, args: []Term{t, Datum(n)}}
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

// binop builds a binary term [type, [t, value]].
func (t Term) binop(tt proto.TermType, value interface{}) Term {
	return Term{termType: tt, args: []Term{t, toTerm(value)}}
}

// MarshalJSON serializes the term to ReQL wire format.
// Datum terms serialize as their raw value; compound terms as [type, [args...], opts?].
func (t Term) MarshalJSON() ([]byte, error) {
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
