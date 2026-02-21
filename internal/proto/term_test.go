package proto

import "testing"

func testTerms(t *testing.T, tests []struct {
	name string
	got  TermType
	want TermType
}) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestTermTypeCoreConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"DB", TermDB, 14},
		{"TABLE", TermTable, 15},
		{"FILTER", TermFilter, 39},
		{"INSERT", TermInsert, 56},
		{"DATUM", TermDatum, 1},
		{"MAKE_ARRAY", TermMakeArray, 2},
		{"MAKE_OBJ", TermMakeObj, 3},
		{"VAR", TermVar, 10},
		{"JAVASCRIPT", TermJavaScript, 11},
		{"ERROR", TermError, 12},
		{"IMPLICIT_VAR", TermImplicitVar, 13},
	})
}

func TestTermTypeDDLConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"DB_CREATE", TermDBCreate, 57},
		{"DB_DROP", TermDBDrop, 58},
		{"DB_LIST", TermDBList, 59},
		{"TABLE_CREATE", TermTableCreate, 60},
		{"TABLE_DROP", TermTableDrop, 61},
		{"TABLE_LIST", TermTableList, 62},
		{"GET", TermGet, 16},
		{"GET_ALL", TermGetAll, 78},
		{"UPDATE", TermUpdate, 53},
		{"DELETE", TermDelete, 54},
		{"REPLACE", TermReplace, 55},
		{"FOR_EACH", TermForEach, 68},
	})
}

func TestTermTypeOperatorConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"EQ", TermEq, 17},
		{"NE", TermNe, 18},
		{"LT", TermLt, 19},
		{"LE", TermLe, 20},
		{"GT", TermGt, 21},
		{"GE", TermGe, 22},
		{"NOT", TermNot, 23},
		{"ADD", TermAdd, 24},
		{"SUB", TermSub, 25},
		{"MUL", TermMul, 26},
		{"DIV", TermDiv, 27},
		{"MOD", TermMod, 28},
		{"AND", TermAnd, 67},
		{"OR", TermOr, 66},
		{"BRANCH", TermBranch, 65},
	})
}

func TestTermTypeSequenceConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"APPEND", TermAppend, 29},
		{"SLICE", TermSlice, 30},
		{"SKIP", TermSkip, 70},
		{"LIMIT", TermLimit, 71},
		{"NTH", TermNth, 45},
		{"UNION", TermUnion, 44},
		{"COUNT", TermCount, 43},
		{"DISTINCT", TermDistinct, 42},
		{"REDUCE", TermReduce, 37},
		{"MAP", TermMap, 38},
		{"CONCAT_MAP", TermConcatMap, 40},
		{"ORDER_BY", TermOrderBy, 41},
	})
}

func TestTermTypeJoinConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"INNER_JOIN", TermInnerJoin, 48},
		{"OUTER_JOIN", TermOuterJoin, 49},
		{"EQ_JOIN", TermEqJoin, 50},
		{"BETWEEN", TermBetween, 182},
		{"SET_INTERSECTION", TermSetIntersection, 89},
	})
}

func TestTermTypeExtendedConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"TO_JSON_STRING", TermToJSONString, 172},
		{"GRANT", TermGrant, 188},
		{"SET_WRITE_HOOK", TermSetWriteHook, 189},
		{"GET_WRITE_HOOK", TermGetWriteHook, 190},
		{"BIT_AND", TermBitAnd, 191},
		{"BIT_OR", TermBitOr, 192},
		{"BIT_XOR", TermBitXor, 193},
		{"BIT_NOT", TermBitNot, 194},
		{"BIT_SAL", TermBitSal, 195},
		{"BIT_SAR", TermBitSar, 196},
	})
}

func TestTermTypeDocumentConstants(t *testing.T) {
	t.Parallel()
	testTerms(t, []struct {
		name string
		got  TermType
		want TermType
	}{
		{"GET_FIELD", TermGetField, 31},
		{"HAS_FIELDS", TermHasFields, 32},
		{"PLUCK", TermPluck, 33},
		{"WITHOUT", TermWithout, 34},
		{"MERGE", TermMerge, 35},
		{"COERCE_TO", TermCoerceTo, 51},
		{"TYPE_OF", TermTypeOf, 52},
		{"NOW", TermNow, 103},
		{"TIME", TermTime, 136},
		{"EPOCH_TIME", TermEpochTime, 101},
		{"POINT", TermPoint, 159},
		{"LINE", TermLine, 160},
		{"POLYGON", TermPolygon, 161},
		{"DISTANCE", TermDistance, 162},
	})
}
