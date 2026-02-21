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
		{"float", Datum(3.14), `3.14`},
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
		{"filter_term", DB("test").Table("users").Filter(DB("test").Table("other").Get("k")), `[39,[[15,[[14,["test"]],"users"]],[16,[[15,[[14,["test"]],"other"]],"k"]]]]`},
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
		{"insert_term", table.Insert(DB("other").Table("src")), `[56,[[15,[[14,["test"]],"users"]],[15,[[14,["other"]],"src"]]]]`},
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

func TestReadOperations(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"get", table.Get("alice"), `[16,[[15,[[14,["test"]],"users"]],"alice"]]`},
		{"getall", table.GetAll("alice", "bob"), `[78,[[15,[[14,["test"]],"users"]],"alice","bob"]]`},
		{"getall_index", table.GetAll("alice", OptArgs{"index": "name"}), `[78,[[15,[[14,["test"]],"users"]],"alice"],{"index":"name"}]`},
		{"between", table.Between(10, 20), `[182,[[15,[[14,["test"]],"users"]],10,20]]`},
		{"orderby_field", table.OrderBy("name"), `[41,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"orderby_asc", table.OrderBy(Asc("name")), `[41,[[15,[[14,["test"]],"users"]],[73,["name"]]]]`},
		{"orderby_desc", table.OrderBy(Desc("age")), `[41,[[15,[[14,["test"]],"users"]],[74,["age"]]]]`},
		{"limit", table.Limit(10), `[71,[[15,[[14,["test"]],"users"]],10]]`},
		{"skip", table.Skip(5), `[70,[[15,[[14,["test"]],"users"]],5]]`},
		{"count", table.Count(), `[43,[[15,[[14,["test"]],"users"]]]]`},
		{"pluck", table.Pluck("name", "age"), `[33,[[15,[[14,["test"]],"users"]],"name","age"]]`},
		{"without", table.Without("password"), `[34,[[15,[[14,["test"]],"users"]],"password"]]`},
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

func TestComparisonOperators(t *testing.T) {
	t.Parallel()
	base := DB("test").Table("users").Get("alice")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"eq", base.Eq("alice"), `[17,[[16,[[15,[[14,["test"]],"users"]],"alice"]],"alice"]]`},
		{"ne", base.Ne("bob"), `[18,[[16,[[15,[[14,["test"]],"users"]],"alice"]],"bob"]]`},
		{"lt", base.Lt(30), `[19,[[16,[[15,[[14,["test"]],"users"]],"alice"]],30]]`},
		{"le", base.Le(30), `[20,[[16,[[15,[[14,["test"]],"users"]],"alice"]],30]]`},
		{"gt", base.Gt(18), `[21,[[16,[[15,[[14,["test"]],"users"]],"alice"]],18]]`},
		{"ge", base.Ge(18), `[22,[[16,[[15,[[14,["test"]],"users"]],"alice"]],18]]`},
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

func TestLogicOperators(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	a := table.Get("alice")
	b := table.Get("bob")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"not", a.Not(), `[23,[[16,[[15,[[14,["test"]],"users"]],"alice"]]]]`},
		{"and", a.And(b), `[67,[[16,[[15,[[14,["test"]],"users"]],"alice"]],[16,[[15,[[14,["test"]],"users"]],"bob"]]]]`},
		{"or", a.Or(b), `[66,[[16,[[15,[[14,["test"]],"users"]],"alice"]],[16,[[15,[[14,["test"]],"users"]],"bob"]]]]`},
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

func TestArithmeticOperators(t *testing.T) {
	t.Parallel()
	base := Datum(10)
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"add", base.Add(5), `[24,[10,5]]`},
		{"sub", base.Sub(3), `[25,[10,3]]`},
		{"mul", base.Mul(2), `[26,[10,2]]`},
		{"div", base.Div(2), `[27,[10,2]]`},
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

func TestObjectOperations(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	doc := map[string]interface{}{"active": true}
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"get_field", table.Get("alice").GetField("name"), `[31,[[16,[[15,[[14,["test"]],"users"]],"alice"]],"name"]]`},
		{"has_fields_none", table.Get("alice").HasFields(), `[32,[[16,[[15,[[14,["test"]],"users"]],"alice"]]]]`},
		{"has_fields_one", table.Get("alice").HasFields("a"), `[32,[[16,[[15,[[14,["test"]],"users"]],"alice"]],"a"]]`},
		{"has_fields_multi", table.Get("alice").HasFields("a", "b"), `[32,[[16,[[15,[[14,["test"]],"users"]],"alice"]],"a","b"]]`},
		{"merge", table.Get("alice").Merge(doc), `[35,[[16,[[15,[[14,["test"]],"users"]],"alice"]],{"active":true}]]`},
		{"distinct", table.Distinct(), `[42,[[15,[[14,["test"]],"users"]]]]`},
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

func TestAggregationOperations(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	fn := DB("test").Table("funcs").Get("f")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"map", table.Map(fn), `[38,[[15,[[14,["test"]],"users"]],[16,[[15,[[14,["test"]],"funcs"]],"f"]]]]`},
		{"reduce", table.Reduce(fn), `[37,[[15,[[14,["test"]],"users"]],[16,[[15,[[14,["test"]],"funcs"]],"f"]]]]`},
		{"group", table.Group("age"), `[144,[[15,[[14,["test"]],"users"]],"age"]]`},
		{"ungroup", table.Group("age").Ungroup(), `[150,[[144,[[15,[[14,["test"]],"users"]],"age"]]]]`},
		{"sum", table.Sum("score"), `[145,[[15,[[14,["test"]],"users"]],"score"]]`},
		{"avg", table.Avg("score"), `[146,[[15,[[14,["test"]],"users"]],"score"]]`},
		{"min", table.Min("age"), `[147,[[15,[[14,["test"]],"users"]],"age"]]`},
		{"max", table.Max("age"), `[148,[[15,[[14,["test"]],"users"]],"age"]]`},
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

func TestIndexOperations(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"index_create", table.IndexCreate("name"), `[75,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"index_drop", table.IndexDrop("name"), `[76,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"index_list", table.IndexList(), `[77,[[15,[[14,["test"]],"users"]]]]`},
		{"index_wait", table.IndexWait("name"), `[140,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"index_status", table.IndexStatus("name"), `[139,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"index_rename", table.IndexRename("old", "new"), `[156,[[15,[[14,["test"]],"users"]],"old","new"]]`},
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

func TestChangefeedAndMiscTerms(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"changes", table.Changes(), `[152,[[15,[[14,["test"]],"users"]]]]`},
		{"changes_include_initial", table.Changes(OptArgs{"include_initial": true}), `[152,[[15,[[14,["test"]],"users"]]],{"include_initial":true}]`},
		{"now", Now(), `[103,[]]`},
		{"uuid", UUID(), `[169,[]]`},
		{"binary", Binary("data"), `[155,["data"]]`},
		{"config", table.Config(), `[174,[[15,[[14,["test"]],"users"]]]]`},
		{"status", table.Status(), `[175,[[15,[[14,["test"]],"users"]]]]`},
		{"grant", table.Grant("alice", map[string]interface{}{"read": true}), `[188,[[15,[[14,["test"]],"users"]],"alice",{"read":true}]]`},
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
