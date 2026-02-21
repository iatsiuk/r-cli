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
	var pred Term
	if pt, ok := predicate.(Term); ok {
		pred = pt
	} else {
		pred = Datum(predicate)
	}
	return Term{termType: proto.TermFilter, args: []Term{t, pred}}
}

// Insert creates an INSERT term ([56, [table, doc]]).
func (t Term) Insert(doc interface{}) Term {
	var d Term
	if dt, ok := doc.(Term); ok {
		d = dt
	} else {
		d = Datum(doc)
	}
	return Term{termType: proto.TermInsert, args: []Term{t, d}}
}

// Update creates an UPDATE term ([53, [table, doc]]).
func (t Term) Update(doc interface{}) Term {
	var d Term
	if dt, ok := doc.(Term); ok {
		d = dt
	} else {
		d = Datum(doc)
	}
	return Term{termType: proto.TermUpdate, args: []Term{t, d}}
}

// Delete creates a DELETE term ([54, [table]]).
func (t Term) Delete() Term {
	return Term{termType: proto.TermDelete, args: []Term{t}}
}

// Replace creates a REPLACE term ([55, [table, doc]]).
func (t Term) Replace(doc interface{}) Term {
	var d Term
	if dt, ok := doc.(Term); ok {
		d = dt
	} else {
		d = Datum(doc)
	}
	return Term{termType: proto.TermReplace, args: []Term{t, d}}
}

// MarshalJSON serializes the term to ReQL wire format.
// Datum terms serialize as their raw value; compound terms as [type, [args...], opts?].
func (t Term) MarshalJSON() ([]byte, error) {
	if t.termType == 0 {
		return json.Marshal(t.datum)
	}
	parts := []interface{}{int(t.termType), t.args}
	if len(t.opts) > 0 {
		parts = append(parts, t.opts)
	}
	return json.Marshal(parts)
}
