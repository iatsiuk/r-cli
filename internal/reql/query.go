package reql

import (
	"encoding/json"
	"fmt"

	"r-cli/internal/proto"
)

// BuildQuery serializes a ReQL query envelope.
// START: [1, term, opts] where "db" opt string is auto-wrapped as DB term.
// CONTINUE: [2]
// STOP: [3]
func BuildQuery(qt proto.QueryType, term Term, opts OptArgs) ([]byte, error) {
	switch qt {
	case proto.QueryContinue, proto.QueryStop:
		return json.Marshal([]interface{}{int(qt)})
	case proto.QueryStart:
		qOpts := make(map[string]interface{}, len(opts))
		for k, v := range opts {
			qOpts[k] = v
		}
		if name, ok := opts["db"].(string); ok {
			qOpts["db"] = DB(name)
		}
		return json.Marshal([]interface{}{int(qt), term, qOpts})
	default:
		return nil, fmt.Errorf("reql: unsupported query type %d", qt)
	}
}
