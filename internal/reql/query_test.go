package reql

import (
	"testing"

	"r-cli/internal/proto"
)

func TestBuildQuery(t *testing.T) {
	t.Parallel()
	table := DB("test").Table("users")
	tests := []struct {
		name string
		qt   proto.QueryType
		term Term
		opts OptArgs
		want string
	}{
		{
			"start_no_opts",
			proto.QueryStart,
			table,
			nil,
			`[1,[15,[[14,["test"]],"users"]],{}]`,
		},
		{
			"start_db_opt",
			proto.QueryStart,
			table,
			OptArgs{"db": "mydb"},
			`[1,[15,[[14,["test"]],"users"]],{"db":[14,["mydb"]]}]`,
		},
		{
			"continue",
			proto.QueryContinue,
			Term{},
			nil,
			`[2]`,
		},
		{
			"stop",
			proto.QueryStop,
			Term{},
			nil,
			`[3]`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := BuildQuery(tc.qt, tc.term, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
