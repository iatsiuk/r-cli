package reql

import (
	"encoding/json"

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
	default:
		qOpts := make(map[string]interface{}, len(opts))
		for k, v := range opts {
			if k == "db" {
				if name, ok := v.(string); ok {
					qOpts[k] = DB(name)
				} else {
					qOpts[k] = v
				}
			} else {
				qOpts[k] = v
			}
		}
		return json.Marshal([]interface{}{int(qt), term, qOpts})
	}
}
