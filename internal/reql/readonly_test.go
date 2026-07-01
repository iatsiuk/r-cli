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
		{"table_create", DB("d").TableCreate("t")},
		{"index_create", DB("d").Table("t").IndexCreate("idx")},
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
