package reql

import (
	"encoding/json"
	"testing"
)

func TestDatumEncoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"string", Datum("foo"), `"foo"`},
		{"number", Datum(42), `42`},
		{"bool", Datum(true), `true`},
		{"nil", Datum(nil), `null`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCoreTermBuilder(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"db", DB("test"), `[14,["test"]]`},
		{"table", DB("test").Table("users"), `[15,[[14,["test"]],"users"]]`},
		{"filter", DB("test").Table("users").Filter(map[string]interface{}{"age": 30}), `[39,[[15,[[14,["test"]],"users"]],{"age":30}]]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestWriteOperations(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	doc := map[string]interface{}{"name": "alice"}
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"insert", table.Insert(doc), `[56,[[15,[[14,["test"]],"users"]],{"name":"alice"}]]`},
		{"update", table.Update(doc), `[53,[[15,[[14,["test"]],"users"]],{"name":"alice"}]]`},
		{"delete", table.Delete(), `[54,[[15,[[14,["test"]],"users"]]]]`},
		{"replace", table.Replace(doc), `[55,[[15,[[14,["test"]],"users"]],{"name":"alice"}]]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestArray(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"simple", Array(10, 20, 30), `[2,[10,20,30]]`},
		{"empty", Array(), `[2,[]]`},
		{"nested", Array(Array(1, 2), 3), `[2,[[2,[1,2]],3]]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
