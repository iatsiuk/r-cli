package reql

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestContainsWriteBuilderWrites(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
	}{
		{"insert", DB("d").Table("t").Insert(map[string]interface{}{"a": 1})},
		{"update", DB("d").Table("t").Update(map[string]interface{}{"a": 1})},
		{"delete", DB("d").Table("t").Delete()},
		{"replace", DB("d").Table("t").Replace(map[string]interface{}{"a": 1})},
		{"db_create", DBCreate("d")},
		{"db_drop", DBDrop("d")},
		{"table_create", DB("d").TableCreate("t")},
		{"table_drop", DB("d").TableDrop("t")},
		{"index_create", DB("d").Table("t").IndexCreate("idx")},
		{"index_drop", DB("d").Table("t").IndexDrop("idx")},
		{"index_rename", DB("d").Table("t").IndexRename("old", "new")},
		{"reconfigure", DB("d").Table("t").Reconfigure(OptArgs{"shards": 1, "replicas": 1})},
		{"rebalance", DB("d").Table("t").Rebalance()},
		{"sync", DB("d").Table("t").Sync()},
		{"grant", Grant("bob", map[string]interface{}{"read": true})},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !tc.term.ContainsWrite() {
				t.Errorf("ContainsWrite() = false, want true")
			}
		})
	}
}

func TestContainsWriteReads(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
	}{
		{"filter", DB("d").Table("t").Filter(map[string]interface{}{"age": 30})},
		{"get", DB("d").Table("t").Get("k")},
		{"db_list", DBList()},
		{"table_list", DB("d").TableList()},
		{"wait", DB("d").Table("t").Wait()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.term.ContainsWrite() {
				t.Errorf("ContainsWrite() = true, want false")
			}
		})
	}
}

func TestContainsWriteNested(t *testing.T) {
	t.Parallel()
	term := DB("d").Table("t").ForEach(Func(DB("d").Table("t2").Insert(Var(1)), 1))
	if !term.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in ForEach, want true")
	}
}

func TestContainsWriteInOptArgs(t *testing.T) {
	t.Parallel()
	write := DB("d").Table("t").Fold(Datum(0), Func(Var(1), 1), OptArgs{
		"emit": Func(DB("d").Table("t2").Insert(Var(2)), 2),
	})
	if !write.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in Fold emit optarg, want true")
	}
	read := DB("d").Table("t").Fold(Datum(0), Func(Var(1), 1), OptArgs{
		"emit": Func(Var(2), 2),
	})
	if read.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for read-only Fold optargs, want false")
	}
	writeInSlice := DB("d").Table("t").Fold(Datum(0), Func(Var(1), 1), OptArgs{
		"emit": []interface{}{Func(DB("d").Table("t2").Insert(Var(2)), 2)},
	})
	if !writeInSlice.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in optarg slice, want true")
	}
	writeInMap := DB("d").Table("t").Fold(Datum(0), Func(Var(1), 1), OptArgs{
		"emit": map[string]interface{}{"fn": Func(DB("d").Table("t2").Insert(Var(2)), 2)},
	})
	if !writeInMap.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in optarg map, want true")
	}
}

func TestContainsWriteRawJSON(t *testing.T) {
	t.Parallel()
	write := Datum(json.RawMessage(`[56,[[15,["t"]],{"a":1}]]`))
	if !write.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for raw-JSON insert, want true")
	}
	read := Datum(json.RawMessage(`[16,[[15,["t"]],"k"]]`))
	if read.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for raw-JSON get, want false")
	}
	// data array [56,[1,2]] encoded as MAKE_ARRAY is data, not a term
	dataArray := Datum(json.RawMessage(`[2,[56,[2,[1,2]]]]`))
	if dataArray.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for MAKE_ARRAY data, want false")
	}
	// bare data array with no numeric term-type head
	bareArray := Datum(json.RawMessage(`["not","a","term"]`))
	if bareArray.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for bare data array, want false")
	}
	// object literal (e.g. r.object result) with a nested write
	nestedObject := Datum(json.RawMessage(`{"nested":[56,[[15,["t"]],{"a":1}]]}`))
	if !nestedObject.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in object literal, want true")
	}
	// write hidden in the opts position (element[2]) of a term array
	writeInOpts := Datum(json.RawMessage(`[39,[[15,["t"]]],{"index":[56,[[15,["t2"]],{"a":1}]]}]`))
	if !writeInOpts.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for write nested in opts position, want true")
	}
	// malformed JSON must not be treated as a write
	malformed := Datum(json.RawMessage(`not json`))
	if malformed.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for malformed JSON, want false")
	}
	// raw []byte datum takes the same path as json.RawMessage
	rawBytes := Datum([]byte(`[56,[[15,["t"]],{"a":1}]]`))
	if !rawBytes.ContainsWrite() {
		t.Errorf("ContainsWrite() = false for []byte insert, want true")
	}
}

func TestContainsWriteNoFalsePositive(t *testing.T) {
	t.Parallel()
	doc := Datum(map[string]interface{}{
		"nums": []interface{}{1, 2, 3},
		"code": []interface{}{56, 78},
	})
	if doc.ContainsWrite() {
		t.Errorf("ContainsWrite() = true for native datum with array fields, want false")
	}
	if errTerm(errors.New("boom")).ContainsWrite() {
		t.Errorf("ContainsWrite() = true for deferred-error term, want false")
	}
}
