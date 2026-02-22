package response

import (
	"testing"
	"time"
)

func TestConvertPseudoTypes_Time(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"$reql_type$": "TIME",
		"epoch_time":  float64(1376436985),
		"timezone":    "+00:00",
	}
	result := ConvertPseudoTypes(v)
	ts, ok := result.(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", result)
	}
	if ts.Unix() != 1376436985 {
		t.Errorf("got unix %d, want 1376436985", ts.Unix())
	}
	if ts.Location() != time.UTC {
		t.Errorf("expected UTC location, got %v", ts.Location())
	}
}

func TestConvertPseudoTypes_TimeWithOffset(t *testing.T) {
	t.Parallel()
	// epoch_time with fractional seconds and non-UTC timezone
	v := map[string]interface{}{
		"$reql_type$": "TIME",
		"epoch_time":  float64(1376436985) + 0.298,
		"timezone":    "+05:30",
	}
	result := ConvertPseudoTypes(v)
	ts, ok := result.(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", result)
	}
	if ts.Unix() != 1376436985 {
		t.Errorf("got unix %d, want 1376436985", ts.Unix())
	}
	_, offset := ts.Zone()
	if offset != 5*3600+30*60 {
		t.Errorf("got offset %d, want %d", offset, 5*3600+30*60)
	}
}

func TestConvertPseudoTypes_Binary(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"$reql_type$": "BINARY",
		"data":        "aGVsbG8=",
	}
	result := ConvertPseudoTypes(v)
	b, ok := result.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", result)
	}
	if string(b) != "hello" {
		t.Errorf("got %q, want %q", string(b), "hello")
	}
}

func TestConvertPseudoTypes_Nested(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"name": "doc",
		"created": map[string]interface{}{
			"$reql_type$": "TIME",
			"epoch_time":  float64(1000000),
			"timezone":    "+00:00",
		},
		"data": map[string]interface{}{
			"$reql_type$": "BINARY",
			"data":        "aGVsbG8=",
		},
	}
	result := ConvertPseudoTypes(v)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if _, ok := m["created"].(time.Time); !ok {
		t.Errorf("expected time.Time for 'created', got %T", m["created"])
	}
	if _, ok := m["data"].([]byte); !ok {
		t.Errorf("expected []byte for 'data', got %T", m["data"])
	}
}

func TestConvertPseudoTypes_Geometry(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"$reql_type$": "GEOMETRY",
		"type":        "Point",
		"coordinates": []interface{}{float64(-122.42), float64(37.78)},
	}
	result := ConvertPseudoTypes(v)
	// geometry passes through as map unchanged
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for GEOMETRY, got %T", result)
	}
	if m["type"] != "Point" {
		t.Errorf("got type %v, want Point", m["type"])
	}
	if m["$reql_type$"] != "GEOMETRY" {
		t.Errorf("expected $reql_type$ preserved, got %v", m["$reql_type$"])
	}
}

func TestConvertPseudoTypes_NestedGeometry(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"name": "location",
		"geo": map[string]interface{}{
			"$reql_type$": "GEOMETRY",
			"type":        "Point",
			"coordinates": []interface{}{float64(0), float64(0)},
		},
	}
	result := ConvertPseudoTypes(v)
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	geo, ok := m["geo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map for 'geo', got %T", m["geo"])
	}
	if geo["type"] != "Point" {
		t.Errorf("got type %v, want Point", geo["type"])
	}
}

func TestConvertPseudoTypes_PassThrough(t *testing.T) {
	t.Parallel()
	// plain values without $reql_type$ pass through unchanged
	cases := []interface{}{
		"hello",
		float64(42),
		true,
		nil,
		map[string]interface{}{"name": "Alice", "age": float64(30)},
	}
	for _, c := range cases {
		result := ConvertPseudoTypes(c)
		switch expected := c.(type) {
		case map[string]interface{}:
			m, ok := result.(map[string]interface{})
			if !ok {
				t.Errorf("expected map, got %T", result)
				continue
			}
			for k, v := range expected {
				if m[k] != v {
					t.Errorf("key %q: got %v, want %v", k, m[k], v)
				}
			}
		default:
			// primitives and nil: result must equal input
			if result != c {
				t.Errorf("got %v, want %v", result, c)
			}
		}
	}
}

func TestConvertPseudoTypes_TimeNoEpoch(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"$reql_type$": "TIME",
		"timezone":    "+00:00",
		// no epoch_time field
	}
	result := ConvertPseudoTypes(v)
	// missing epoch_time -> pass-through as original map
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map passthrough, got %T", result)
	}
	if m["$reql_type$"] != "TIME" {
		t.Errorf("expected original map preserved, got %v", m)
	}
}

func TestConvertPseudoTypes_BinaryInvalidBase64(t *testing.T) {
	t.Parallel()
	v := map[string]interface{}{
		"$reql_type$": "BINARY",
		"data":        "!!!not valid base64!!!",
	}
	result := ConvertPseudoTypes(v)
	// invalid base64 -> pass-through as original map
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map passthrough, got %T", result)
	}
	if m["$reql_type$"] != "BINARY" {
		t.Errorf("expected original map preserved, got %v", m)
	}
}

func TestParseTimezone_Invalid(t *testing.T) {
	t.Parallel()
	cases := []string{
		"invalid", // non-numeric, wrong length
		"05:30",   // missing sign prefix
		"+5:30",   // wrong length (5 chars instead of 6)
	}
	for _, tz := range cases {
		_, err := parseTimezone(tz)
		if err == nil {
			t.Errorf("expected error for timezone %q, got nil", tz)
		}
	}
}

func TestConvertPseudoTypes_SliceNested(t *testing.T) {
	t.Parallel()
	v := []interface{}{
		map[string]interface{}{
			"$reql_type$": "TIME",
			"epoch_time":  float64(500),
			"timezone":    "+00:00",
		},
		"plain",
	}
	result := ConvertPseudoTypes(v)
	s, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result)
	}
	if _, ok := s[0].(time.Time); !ok {
		t.Errorf("expected time.Time at index 0, got %T", s[0])
	}
	if s[1] != "plain" {
		t.Errorf("got %v, want plain", s[1])
	}
}
