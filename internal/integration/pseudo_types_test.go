//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"r-cli/internal/reql"
	"r-cli/internal/response"
)

func TestNowTimestamp(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	before := time.Now().Unix()

	_, cur, err := exec.Run(ctx, reql.Now(), nil)
	if err != nil {
		t.Fatalf("now: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var tm struct {
		ReqlType  string  `json:"$reql_type$"`
		EpochTime float64 `json:"epoch_time"`
	}
	if err := json.Unmarshal(raw, &tm); err != nil {
		t.Fatalf("unmarshal time: %v", err)
	}
	if tm.ReqlType != "TIME" {
		t.Errorf("$reql_type$=%q, want TIME", tm.ReqlType)
	}
	after := time.Now().Unix()
	epochSec := int64(tm.EpochTime)
	if epochSec < before-5 || epochSec > after+5 {
		t.Errorf("epoch_time=%v is not recent (before=%d, after=%d)", tm.EpochTime, before, after)
	}
}

func TestTimePseudoTypeConversion(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	before := time.Now()
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(
		map[string]interface{}{"id": "t1", "ts": reql.Now()},
	), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	after := time.Now()

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("t1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}

	converted := response.ConvertPseudoTypes(doc)
	convDoc, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("converted is not map, got %T", converted)
	}
	ts, ok := convDoc["ts"].(time.Time)
	if !ok {
		t.Fatalf("ts field is not time.Time, got %T: %v", convDoc["ts"], convDoc["ts"])
	}
	slack := 10 * time.Second
	if ts.Before(before.Add(-slack)) || ts.After(after.Add(slack)) {
		t.Errorf("ts=%v is not within expected range [%v, %v]", ts, before, after)
	}
}

func TestTimezoneRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(
		map[string]interface{}{"id": "tz1", "ts": reql.ISO8601("2024-01-15T10:30:00+05:30")},
	), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("tz1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var doc struct {
		Ts struct {
			ReqlType  string  `json:"$reql_type$"`
			Timezone  string  `json:"timezone"`
			EpochTime float64 `json:"epoch_time"`
		} `json:"ts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}
	if doc.Ts.ReqlType != "TIME" {
		t.Errorf("$reql_type$=%q, want TIME", doc.Ts.ReqlType)
	}
	if doc.Ts.Timezone != "+05:30" {
		t.Errorf("timezone=%q, want +05:30", doc.Ts.Timezone)
	}

	// verify ConvertPseudoTypes preserves the UTC offset
	converted := response.ConvertPseudoTypes(map[string]interface{}{
		"$reql_type$": "TIME",
		"epoch_time":  doc.Ts.EpochTime,
		"timezone":    doc.Ts.Timezone,
	})
	ts, ok := converted.(time.Time)
	if !ok {
		t.Fatalf("converted is not time.Time, got %T", converted)
	}
	_, offset := ts.Zone()
	const wantOffset = 5*3600 + 30*60
	if offset != wantOffset {
		t.Errorf("timezone offset=%d seconds, want %d (+05:30)", offset, wantOffset)
	}
}

func TestBinaryRoundtrip(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	dbName := sanitizeID(t.Name())
	setupTestDB(t, exec, dbName)
	createTestTable(t, exec, dbName, "docs")

	original := []byte("Hello, RethinkDB!")

	// insert using BINARY pseudo-type datum so the server stores the raw bytes
	binaryDatum := map[string]interface{}{
		"$reql_type$": "BINARY",
		"data":        base64.StdEncoding.EncodeToString(original),
	}
	_, cur, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Insert(
		map[string]interface{}{"id": "bin1", "data": binaryDatum},
	), nil)
	closeCursor(cur)
	if err != nil {
		t.Fatalf("insert binary: %v", err)
	}

	_, cur2, err := exec.Run(ctx, reql.DB(dbName).Table("docs").Get("bin1"), nil)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer closeCursor(cur2)

	raw, err := cur2.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal doc: %v", err)
	}

	converted := response.ConvertPseudoTypes(doc)
	convDoc, ok := converted.(map[string]interface{})
	if !ok {
		t.Fatalf("converted is not map, got %T", converted)
	}
	b, ok := convDoc["data"].([]byte)
	if !ok {
		t.Fatalf("data field is not []byte, got %T: %v", convDoc["data"], convDoc["data"])
	}
	if !bytes.Equal(b, original) {
		t.Errorf("binary roundtrip mismatch: got %v, want %v", b, original)
	}
}

func TestUUIDGenerated(t *testing.T) {
	t.Parallel()
	exec := newExecutor(t)

	ctx := context.Background()
	_, cur, err := exec.Run(ctx, reql.UUID(), nil)
	if err != nil {
		t.Fatalf("uuid: %v", err)
	}
	defer closeCursor(cur)

	raw, err := cur.Next()
	if err != nil {
		t.Fatalf("cursor next: %v", err)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal uuid: %v", err)
	}
	if !uuidRe.MatchString(s) {
		t.Errorf("uuid=%q is not valid UUID format", s)
	}
}
