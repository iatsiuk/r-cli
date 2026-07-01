package reql

import (
	"encoding/json"

	"r-cli/internal/proto"
)

// ContainsWrite reports whether the term tree contains any write operation
// (data write, DDL, index DDL, admin write, permission grant, write hook).
// It walks nested terms so writes hidden inside forEach/do/function bodies,
// optarg values (e.g. Fold's emit/finalEmit lambdas), or object/array literals
// (e.g. filter({a: r.table(...).insert(...)}), which the parser stores as a
// native Datum) are detected.
func (t Term) ContainsWrite() bool {
	if t.err != nil {
		return false
	}
	if t.termType == 0 {
		return datumContainsWrite(t.datum)
	}
	if proto.IsWriteTerm(t.termType) {
		return true
	}
	for _, a := range t.args {
		if a.ContainsWrite() {
			return true
		}
	}
	for _, v := range t.opts {
		if datumContainsWrite(v) {
			return true
		}
	}
	return false
}

// datumContainsWrite scans a datum value for embedded write terms. Datums are
// usually plain data (json.RawMessage/[]byte from the run path, or scalar Go
// values from the builder path), but object/array literals and Fold's
// emit/finalEmit accept full expressions, so a value can be a Term (or a
// slice/map containing one).
func datumContainsWrite(datum interface{}) bool {
	switch v := datum.(type) {
	case json.RawMessage:
		return rawJSONContainsWrite(v)
	case []byte:
		return rawJSONContainsWrite(v)
	case Term:
		return v.ContainsWrite()
	case []interface{}:
		for _, item := range v {
			if datumContainsWrite(item) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range v {
			if datumContainsWrite(item) {
				return true
			}
		}
	}
	return false
}

// rawJSONContainsWrite decodes a wire-format ReQL term and looks for write
// term types. It is reliable because ReQL data arrays are always encoded as
// MAKE_ARRAY [2,[...]], so a bare [writeType,[...]] is always a term.
func rawJSONContainsWrite(data []byte) bool {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return false
	}
	return jsonTermContainsWrite(v)
}

// jsonTermContainsWrite walks a decoded wire-format value at a term position:
// arrays are [type, [args...], opts?] terms, objects are literals whose values
// are terms. Only element[1] entries are recursed as terms, so numeric datums
// sitting in a MAKE_ARRAY arg list are not mistaken for term heads.
func jsonTermContainsWrite(v interface{}) bool {
	switch val := v.(type) {
	case []interface{}:
		return jsonArrayContainsWrite(val)
	case map[string]interface{}:
		for _, item := range val {
			if jsonTermContainsWrite(item) {
				return true
			}
		}
	}
	return false
}

// jsonArrayContainsWrite inspects a term array [type, [args...], opts?].
func jsonArrayContainsWrite(val []interface{}) bool {
	if len(val) == 0 {
		return false
	}
	tt, ok := val[0].(float64)
	if !ok {
		return false
	}
	if proto.IsWriteTerm(proto.TermType(int(tt))) {
		return true
	}
	if len(val) >= 2 {
		if args, ok := val[1].([]interface{}); ok {
			for _, a := range args {
				if jsonTermContainsWrite(a) {
					return true
				}
			}
		}
	}
	if len(val) >= 3 {
		if opts, ok := val[2].(map[string]interface{}); ok {
			return jsonTermContainsWrite(opts)
		}
	}
	return false
}
