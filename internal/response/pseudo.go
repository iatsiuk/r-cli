package response

import (
	"encoding/base64"
	"fmt"
	"time"
)

const reqlTypeKey = "$reql_type$"

// ConvertPseudoTypes recursively converts RethinkDB pseudo-types to native Go types:
//   - TIME -> time.Time (epoch_time + timezone)
//   - BINARY -> []byte (base64-decoded data)
//   - GEOMETRY -> pass-through (no conversion needed)
//
// Plain values and maps without $reql_type$ are returned unchanged.
func ConvertPseudoTypes(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return convertMap(val)
	case []interface{}:
		return convertSlice(val)
	default:
		return v
	}
}

func convertMap(m map[string]interface{}) interface{} {
	rt, ok := m[reqlTypeKey]
	if !ok {
		result := make(map[string]interface{}, len(m))
		for k, v := range m {
			result[k] = ConvertPseudoTypes(v)
		}
		return result
	}

	switch rt {
	case "TIME":
		return convertTime(m)
	case "BINARY":
		return convertBinary(m)
	default:
		// GEOMETRY and unknown pseudo-types: pass through as-is
		return m
	}
}

func convertTime(m map[string]interface{}) interface{} {
	epochTime, ok := m["epoch_time"].(float64)
	if !ok {
		return m
	}
	tz, _ := m["timezone"].(string)
	loc, err := parseTimezone(tz)
	if err != nil {
		loc = time.UTC
	}
	sec := int64(epochTime)
	nsec := int64((epochTime - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).In(loc)
}

func parseTimezone(tz string) (*time.Location, error) {
	if tz == "" || tz == "+00:00" || tz == "-00:00" || tz == "Z" {
		return time.UTC, nil
	}
	if len(tz) != 6 {
		return nil, fmt.Errorf("invalid timezone: %q", tz)
	}
	sign := 1
	if tz[0] == '-' {
		sign = -1
	} else if tz[0] != '+' {
		return nil, fmt.Errorf("invalid timezone: %q", tz)
	}
	var hours, minutes int
	if _, err := fmt.Sscanf(tz[1:], "%d:%d", &hours, &minutes); err != nil {
		return nil, fmt.Errorf("invalid timezone: %q", tz)
	}
	offset := sign * (hours*3600 + minutes*60)
	return time.FixedZone(tz, offset), nil
}

func convertBinary(m map[string]interface{}) interface{} {
	data, ok := m["data"].(string)
	if !ok {
		return m
	}
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return m
	}
	return b
}

func convertSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = ConvertPseudoTypes(v)
	}
	return result
}
