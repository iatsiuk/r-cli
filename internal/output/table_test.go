package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable_AlignedASCII(t *testing.T) {
	t.Parallel()
	iter := newIter(
		`{"name":"alice","age":30,"city":"NYC"}`,
		`{"name":"bob","age":25,"city":"LA"}`,
	)
	var buf bytes.Buffer
	if err := Table(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// header + separator + 2 data rows
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d:\n%s", len(lines), got)
	}
	header := lines[0]
	for _, col := range []string{"name", "age", "city"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing column %q: %q", col, header)
		}
	}
	if !strings.Contains(lines[1], "---") {
		t.Errorf("separator line missing dashes: %q", lines[1])
	}
	row1 := lines[2]
	if !strings.Contains(row1, "alice") {
		t.Errorf("row 1 missing alice: %q", row1)
	}
	if !strings.Contains(row1, "NYC") {
		t.Errorf("row 1 missing NYC: %q", row1)
	}
	if strings.Count(row1, "|") != strings.Count(header, "|") {
		t.Errorf("row 1 | count differs from header: %q", row1)
	}
}

func TestTable_MissingFields(t *testing.T) {
	t.Parallel()
	iter := newIter(
		`{"name":"alice","age":30}`,
		`{"name":"bob"}`,
		`{"age":99}`,
	)
	var buf bytes.Buffer
	if err := Table(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// header + separator + 3 data rows
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(lines), got)
	}
	// header has both columns
	header := lines[0]
	if !strings.Contains(header, "name") || !strings.Contains(header, "age") {
		t.Errorf("header missing columns: %q", header)
	}
	// second data row (bob) has name but empty age - columns still present
	row2 := lines[3]
	if !strings.Contains(row2, "bob") {
		t.Errorf("row 2 missing bob: %q", row2)
	}
	// third data row (age:99) has no name - verify it has | separators still
	row3 := lines[4]
	if strings.Count(row3, "|") != strings.Count(header, "|") {
		t.Errorf("row 3 has wrong | count: %q", row3)
	}
	if !strings.Contains(row3, "99") {
		t.Errorf("row 3 missing age 99: %q", row3)
	}
}

func TestTable_TruncateLongValues(t *testing.T) {
	t.Parallel()
	longVal := strings.Repeat("x", maxColWidth+10)
	iter := newIter(`{"col":"` + longVal + `"}`)
	var buf bytes.Buffer
	if err := Table(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	// header + separator + 1 data row
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), got)
	}
	dataRow := lines[2]
	// value must be truncated to maxColWidth
	fields := strings.SplitN(dataRow, " | ", 2)
	val := strings.TrimSpace(fields[0])
	if len(val) > maxColWidth {
		t.Errorf("value not truncated: len=%d, expected <=%d", len(val), maxColWidth)
	}
	// truncated value ends with ~
	if !strings.HasSuffix(val, "~") {
		t.Errorf("truncated value should end with ~: %q", val)
	}
}

func TestTable_NonObjectFallback(t *testing.T) {
	t.Parallel()
	iter := newIter(`"hello"`, `"world"`, `42`)
	var buf bytes.Buffer
	if err := Table(&buf, iter); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// no table headers - raw fallback
	if strings.Contains(got, "---") {
		t.Errorf("non-object should not produce table separator: %q", got)
	}
	if !strings.Contains(got, "hello") || !strings.Contains(got, "world") || !strings.Contains(got, "42") {
		t.Errorf("missing raw values in output: %q", got)
	}
}

func TestTable_MaxRowsTruncation(t *testing.T) {
	t.Parallel()
	const testMax = 3
	items := []string{
		`{"n":1}`, `{"n":2}`, `{"n":3}`, `{"n":4}`,
	}
	iter := newIter(items...)
	var out, errOut bytes.Buffer
	if err := tableWriter(&out, &errOut, iter, testMax); err != nil {
		t.Fatal(err)
	}
	// check warning on stderr
	warn := errOut.String()
	if !strings.Contains(warn, "warning") {
		t.Errorf("expected truncation warning, got: %q", warn)
	}
	// check output has exactly testMax data rows (header + separator + testMax rows)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	dataLines := len(lines) - 2 // subtract header and separator
	if dataLines != testMax {
		t.Errorf("expected %d data rows, got %d:\n%s", testMax, dataLines, out.String())
	}
}
