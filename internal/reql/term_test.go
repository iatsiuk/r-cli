package reql

import (
	"encoding/json"
	"strings"
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
		{"index_wait_all", table.IndexWait(), `[140,[[15,[[14,["test"]],"users"]]]]`},
		{"index_status", table.IndexStatus("name"), `[139,[[15,[[14,["test"]],"users"]],"name"]]`},
		{"index_status_all", table.IndexStatus(), `[139,[[15,[[14,["test"]],"users"]]]]`},
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
		{"changes_empty_opts", table.Changes(OptArgs{}), `[152,[[15,[[14,["test"]],"users"]]]]`},
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

func TestFuncSerialization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"var", Var(1), `[10,[1]]`},
		{"var_id", Var(42), `[10,[42]]`},
		{"zero_params_func", Func(Datum(42)), `[69,[[2,[]],42]]`},
		{"single_arg_func", Func(Datum(42), 1), `[69,[[2,[1]],42]]`},
		{"multi_arg_func", Func(Var(1).Add(Var(2)), 1, 2), `[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]]`},
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

func TestImplicitVarWrapping(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	tests := []struct {
		name    string
		term    Term
		want    string
		wantErr bool
	}{
		{
			// IMPLICIT_VAR in predicate -> auto-wrapped in FUNC
			"wrap_simple",
			table.Filter(Row().GetField("age").Gt(21)),
			`[39,[[15,[[14,["test"]],"users"]],[69,[[2,[1]],[21,[[31,[[10,[1]],"age"]],21]]]]]]`,
			false,
		},
		{
			// multiple IMPLICIT_VAR at different positions -> all replaced
			"wrap_multiple",
			table.Filter(Row().GetField("x").Eq(Row().GetField("y"))),
			`[39,[[15,[[14,["test"]],"users"]],[69,[[2,[1]],[17,[[31,[[10,[1]],"x"]],[31,[[10,[1]],"y"]]]]]]]]`,
			false,
		},
		{
			// explicit FUNC predicate passes through unchanged
			"explicit_func",
			table.Filter(Func(Var(1).GetField("age").Gt(21), 1)),
			`[39,[[15,[[14,["test"]],"users"]],[69,[[2,[1]],[21,[[31,[[10,[1]],"age"]],21]]]]]]`,
			false,
		},
		{
			// compound predicate with And - all IMPLICIT_VAR nodes replaced
			"wrap_compound",
			table.Filter(Row().GetField("a").Gt(0).And(Row().GetField("b").Lt(10))),
			`[39,[[15,[[14,["test"]],"users"]],[69,[[2,[1]],[67,[[21,[[31,[[10,[1]],"a"]],0]],[19,[[31,[[10,[1]],"b"]],10]]]]]]]]`,
			false,
		},
		{
			// IMPLICIT_VAR inside explicit FUNC -> error (ambiguous)
			"nested_func_error",
			table.Filter(Func(Row().Gt(0), 1)),
			"",
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tc.term)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "IMPLICIT_VAR") {
					t.Errorf("expected IMPLICIT_VAR error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFuncCall(t *testing.T) {
	t.Parallel()
	fn := Func(Var(1).Add(Var(2)), 1, 2)
	tests := []struct {
		name string
		term Term
		want string
	}{
		{
			// Do(arg1, arg2, fn) -> [64,[fn,arg1,arg2]]
			"two_args",
			Do(10, 20, fn),
			`[64,[[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]],10,20]]`,
		},
		{
			// Do(fn) with no extra args -> [64,[fn]]
			"no_args",
			Do(fn),
			`[64,[[69,[[2,[1,2]],[24,[[10,[1]],[10,[2]]]]]]]]`,
		},
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

func TestDoError(t *testing.T) {
	t.Parallel()
	_, err := json.Marshal(Do())
	if err == nil {
		t.Fatal("expected error for Do() with no args, got nil")
	}
}

func TestAdminTerms(t *testing.T) {
	t.Parallel()
	db := DB("test")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{"db_create", DBCreate("mydb"), `[57,["mydb"]]`},
		{"db_drop", DBDrop("mydb"), `[58,["mydb"]]`},
		{"db_list", DBList(), `[59,[]]`},
		{"table_create", db.TableCreate("users"), `[60,[[14,["test"]],"users"]]`},
		{"table_drop", db.TableDrop("users"), `[61,[[14,["test"]],"users"]]`},
		{"table_list", db.TableList(), `[62,[[14,["test"]]]]`},
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

func TestTermOptargs(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	db := DB("test")
	doc := map[string]interface{}{"name": "alice"}
	tests := []struct {
		name string
		term Term
		want string
	}{
		{
			"insert_conflict",
			table.Insert(doc, OptArgs{"conflict": "replace"}),
			`[56,[[15,[[14,["test"]],"users"]],{"name":"alice"}],{"conflict":"replace"}]`,
		},
		{
			"insert_return_changes",
			table.Insert(doc, OptArgs{"return_changes": true}),
			`[56,[[15,[[14,["test"]],"users"]],{"name":"alice"}],{"return_changes":true}]`,
		},
		{
			"changes_include_initial",
			table.Changes(OptArgs{"include_initial": true}),
			`[152,[[15,[[14,["test"]],"users"]]],{"include_initial":true}]`,
		},
		{
			"table_create_primary_key",
			db.TableCreate("users", OptArgs{"primary_key": "user_id"}),
			`[60,[[14,["test"]],"users"],{"primary_key":"user_id"}]`,
		},
		{
			"orderby_index",
			table.OrderBy(OptArgs{"index": "name"}),
			`[41,[[15,[[14,["test"]],"users"]]],{"index":"name"}]`,
		},
		{
			"orderby_field_and_index",
			table.OrderBy("age", OptArgs{"index": "name"}),
			`[41,[[15,[[14,["test"]],"users"]],"age"],{"index":"name"}]`,
		},
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

func TestJoinOperations(t *testing.T) {
	t.Parallel()
	users := DB("test").Table("users")
	posts := DB("test").Table("posts")
	fn := Func(Var(1).GetField("id").Eq(Var(2).GetField("user_id")), 1, 2)
	tests := []struct {
		name string
		term Term
		want string
	}{
		{
			"inner_join",
			users.InnerJoin(posts, fn),
			`[48,[[15,[[14,["test"]],"users"]],[15,[[14,["test"]],"posts"]],[69,[[2,[1,2]],[17,[[31,[[10,[1]],"id"]],[31,[[10,[2]],"user_id"]]]]]]]]`,
		},
		{
			"outer_join",
			users.OuterJoin(posts, fn),
			`[49,[[15,[[14,["test"]],"users"]],[15,[[14,["test"]],"posts"]],[69,[[2,[1,2]],[17,[[31,[[10,[1]],"id"]],[31,[[10,[2]],"user_id"]]]]]]]]`,
		},
		{
			"eq_join",
			users.EqJoin("user_id", posts),
			`[50,[[15,[[14,["test"]],"users"]],"user_id",[15,[[14,["test"]],"posts"]]]]`,
		},
		{
			"eq_join_index",
			users.EqJoin("user_id", posts, OptArgs{"index": "name"}),
			`[50,[[15,[[14,["test"]],"users"]],"user_id",[15,[[14,["test"]],"posts"]]],{"index":"name"}]`,
		},
		{
			"zip",
			users.Zip(),
			`[72,[[15,[[14,["test"]],"users"]]]]`,
		},
		{
			"zip_after_join",
			users.InnerJoin(posts, fn).Zip(),
			`[72,[[48,[[15,[[14,["test"]],"users"]],[15,[[14,["test"]],"posts"]],[69,[[2,[1,2]],[17,[[31,[[10,[1]],"id"]],[31,[[10,[2]],"user_id"]]]]]]]]]]`,
		},
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

func TestStringOperations(t *testing.T) {
	t.Parallel()
	str := Datum("hello world")
	tests := []struct {
		name string
		term Term
		want string
	}{
		{
			"match",
			str.Match(`\w+`),
			`[97,["hello world","\\w+"]]`,
		},
		{
			"split_with_delim",
			str.Split(" "),
			`[149,["hello world"," "]]`,
		},
		{
			"split_no_delim",
			str.Split(),
			`[149,["hello world"]]`,
		},
		{
			"upcase",
			str.Upcase(),
			`[141,["hello world"]]`,
		},
		{
			"downcase",
			str.Downcase(),
			`[142,["hello world"]]`,
		},
		{
			"to_json_string",
			str.ToJsonString(),
			`[172,["hello world"]]`,
		},
		{
			"json",
			Json(`{"a":1}`),
			`[98,["{\"a\":1}"]]`,
		},
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

func TestTimeOperations(t *testing.T) {
	t.Parallel()
	ts := Now()
	start := EpochTime(0)
	end := EpochTime(1000000)
	tests := []struct {
		name string
		term Term
		want string
	}{
		// construction
		{"iso8601", ISO8601("2024-01-01T00:00:00Z"), `[99,["2024-01-01T00:00:00Z"]]`},
		{"epoch_time", EpochTime(1234567890), `[101,[1234567890]]`},
		{"time", Time(2024, 1, 1, "Z"), `[136,[2024,1,1,"Z"]]`},
		{"now", Now(), `[103,[]]`},
		// extraction
		{"to_iso8601", ts.ToISO8601(), `[100,[[103,[]]]]`},
		{"to_epoch_time", ts.ToEpochTime(), `[102,[[103,[]]]]`},
		{"date", ts.Date(), `[106,[[103,[]]]]`},
		{"time_of_day", ts.TimeOfDay(), `[126,[[103,[]]]]`},
		{"timezone", ts.Timezone(), `[127,[[103,[]]]]`},
		{"year", ts.Year(), `[128,[[103,[]]]]`},
		{"month", ts.Month(), `[129,[[103,[]]]]`},
		{"day", ts.Day(), `[130,[[103,[]]]]`},
		{"day_of_week", ts.DayOfWeek(), `[131,[[103,[]]]]`},
		{"day_of_year", ts.DayOfYear(), `[132,[[103,[]]]]`},
		{"hours", ts.Hours(), `[133,[[103,[]]]]`},
		{"minutes", ts.Minutes(), `[134,[[103,[]]]]`},
		{"seconds", ts.Seconds(), `[135,[[103,[]]]]`},
		// operations
		{"in_timezone", ts.InTimezone("+02:00"), `[104,[[103,[]],"+02:00"]]`},
		{"during", ts.During(start, end), `[105,[[103,[]],[101,[0]],[101,[1000000]]]]`},
		// day-of-week constants
		{"monday", Monday(), `[107,[]]`},
		{"tuesday", Tuesday(), `[108,[]]`},
		{"wednesday", Wednesday(), `[109,[]]`},
		{"thursday", Thursday(), `[110,[]]`},
		{"friday", Friday(), `[111,[]]`},
		{"saturday", Saturday(), `[112,[]]`},
		{"sunday", Sunday(), `[113,[]]`},
		// month constants
		{"january", January(), `[114,[]]`},
		{"february", February(), `[115,[]]`},
		{"march", March(), `[116,[]]`},
		{"april", April(), `[117,[]]`},
		{"may", May(), `[118,[]]`},
		{"june", June(), `[119,[]]`},
		{"july", July(), `[120,[]]`},
		{"august", August(), `[121,[]]`},
		{"september", September(), `[122,[]]`},
		{"october", October(), `[123,[]]`},
		{"november", November(), `[124,[]]`},
		{"december", December(), `[125,[]]`},
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

func runTermTests(t *testing.T, tests []struct {
	name string
	term Term
	want string
}) {
	t.Helper()
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

func TestArrayMutations(t *testing.T) {
	t.Parallel()
	arr := Array(1, 2, 3)
	other := Array(3, 4, 5)
	runTermTests(t, []struct {
		name string
		term Term
		want string
	}{
		{"append", arr.Append(4), `[29,[[2,[1,2,3]],4]]`},
		{"prepend", arr.Prepend(0), `[80,[[2,[1,2,3]],0]]`},
		{"slice", arr.Slice(1, 3), `[30,[[2,[1,2,3]],1,3]]`},
		{"difference", arr.Difference(other), `[95,[[2,[1,2,3]],[2,[3,4,5]]]]`},
		{"insert_at", arr.InsertAt(1, 99), `[82,[[2,[1,2,3]],1,99]]`},
		{"delete_at", arr.DeleteAt(1), `[83,[[2,[1,2,3]],1]]`},
		{"change_at", arr.ChangeAt(1, 99), `[84,[[2,[1,2,3]],1,99]]`},
		{"splice_at", arr.SpliceAt(1, Array(10, 11)), `[85,[[2,[1,2,3]],1,[2,[10,11]]]]`},
	})
}

func TestSetOperations(t *testing.T) {
	t.Parallel()
	arr := Array(1, 2, 3)
	other := Array(3, 4, 5)
	runTermTests(t, []struct {
		name string
		term Term
		want string
	}{
		{"set_insert", arr.SetInsert(4), `[88,[[2,[1,2,3]],4]]`},
		{"set_intersection", arr.SetIntersection(other), `[89,[[2,[1,2,3]],[2,[3,4,5]]]]`},
		{"set_union", arr.SetUnion(other), `[90,[[2,[1,2,3]],[2,[3,4,5]]]]`},
		{"set_difference", arr.SetDifference(other), `[91,[[2,[1,2,3]],[2,[3,4,5]]]]`},
	})
}

func TestControlFlow(t *testing.T) {
	t.Parallel()
	cond := DB("test").Table("users").Count().Gt(0)
	seq := DB("test").Table("users")
	fn := Func(Var(1).GetField("active"), 1)
	runTermTests(t, []struct {
		name string
		term Term
		want string
	}{
		{
			"branch_simple",
			Branch(cond, Datum("yes"), Datum("no")),
			`[65,[[21,[[43,[[15,[[14,["test"]],"users"]]]],0]],"yes","no"]]`,
		},
		{
			"branch_multi",
			Branch(Datum(true), Datum(1), Datum(false), Datum(2), Datum(3)),
			`[65,[true,1,false,2,3]]`,
		},
		{
			"for_each",
			seq.ForEach(fn),
			`[68,[[15,[[14,["test"]],"users"]],[69,[[2,[1]],[31,[[10,[1]],"active"]]]]]]`,
		},
		{
			"default",
			seq.Count().Default(0),
			`[92,[[43,[[15,[[14,["test"]],"users"]]]],0]]`,
		},
		{
			"error",
			Error("something went wrong"),
			`[12,["something went wrong"]]`,
		},
		{
			"coerce_to",
			seq.Count().CoerceTo("string"),
			`[51,[[43,[[15,[[14,["test"]],"users"]]]],"string"]]`,
		},
		{
			"type_of",
			seq.TypeOf(),
			`[52,[[15,[[14,["test"]],"users"]]]]`,
		},
	})
}
