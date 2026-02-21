package proto

// TermType identifies a ReQL term in a serialized query.
type TermType int

const (
	// base / special
	TermDatum       TermType = 1
	TermMakeArray   TermType = 2
	TermMakeObj     TermType = 3
	TermVar         TermType = 10
	TermJavaScript  TermType = 11
	TermError       TermType = 12
	TermImplicitVar TermType = 13
	TermArgs        TermType = 154
	TermBinary      TermType = 155
	TermUUID        TermType = 169

	// database / table DDL
	TermDB          TermType = 14
	TermTable       TermType = 15
	TermDBCreate    TermType = 57
	TermDBDrop      TermType = 58
	TermDBList      TermType = 59
	TermTableCreate TermType = 60
	TermTableDrop   TermType = 61
	TermTableList   TermType = 62
	TermConfig      TermType = 174
	TermStatus      TermType = 175
	TermWait        TermType = 177
	TermReconfigure TermType = 176
	TermRebalance   TermType = 179
	TermSync        TermType = 138

	// index DDL
	TermIndexCreate TermType = 75
	TermIndexDrop   TermType = 76
	TermIndexList   TermType = 77
	TermIndexStatus TermType = 139
	TermIndexWait   TermType = 140
	TermIndexRename TermType = 156

	// row read
	TermGet    TermType = 16
	TermGetAll TermType = 78

	// comparison operators
	TermEq  TermType = 17
	TermNe  TermType = 18
	TermLt  TermType = 19
	TermLe  TermType = 20
	TermGt  TermType = 21
	TermGe  TermType = 22
	TermNot TermType = 23

	// math operators
	TermAdd   TermType = 24
	TermSub   TermType = 25
	TermMul   TermType = 26
	TermDiv   TermType = 27
	TermMod   TermType = 28
	TermFloor TermType = 183
	TermCeil  TermType = 184
	TermRound TermType = 185

	// sequence / array operators
	TermAppend        TermType = 29
	TermPrepend       TermType = 80
	TermDifference    TermType = 95
	TermSetInsert     TermType = 88
	TermSetIntersect  TermType = 89
	TermSetUnion      TermType = 90
	TermSetDifference TermType = 91
	TermSlice         TermType = 30
	TermSkip          TermType = 70
	TermLimit         TermType = 71
	TermOffsetsOf     TermType = 87
	TermContains      TermType = 93
	TermRange         TermType = 173
	TermInsertAt      TermType = 82
	TermDeleteAt      TermType = 83
	TermChangeAt      TermType = 84
	TermSpliceAt      TermType = 85
	TermNth           TermType = 45
	TermBracket       TermType = 170
	TermInnerJoin     TermType = 48
	TermOuterJoin     TermType = 49
	TermEqJoin        TermType = 50
	TermZip           TermType = 72
	TermUnion         TermType = 44
	TermSample        TermType = 81
	TermIsEmpty       TermType = 86
	TermDistinct      TermType = 42
	TermCount         TermType = 43
	TermGroup         TermType = 144
	TermUngroup       TermType = 150
	TermSum           TermType = 145
	TermAvg           TermType = 146
	TermMin           TermType = 147
	TermMax           TermType = 148
	TermMinVal        TermType = 180
	TermMaxVal        TermType = 181
	TermRandom        TermType = 151

	// document operators
	TermGetField   TermType = 31
	TermKeys       TermType = 94
	TermValues     TermType = 186
	TermObject     TermType = 143
	TermHasFields  TermType = 32
	TermWithFields TermType = 96
	TermPluck      TermType = 33
	TermWithout    TermType = 34
	TermMerge      TermType = 35
	TermLiteral    TermType = 137

	// query operators
	TermBetween   TermType = 36
	TermFilter    TermType = 39
	TermReduce    TermType = 37
	TermMap       TermType = 38
	TermConcatMap TermType = 40
	TermOrderBy   TermType = 41
	TermFold      TermType = 187
	TermChanges   TermType = 152

	// write operators
	TermUpdate  TermType = 53
	TermDelete  TermType = 54
	TermReplace TermType = 55
	TermInsert  TermType = 56
	TermForEach TermType = 68

	// control flow
	TermFuncCall TermType = 64
	TermBranch   TermType = 65
	TermOr       TermType = 66
	TermAnd      TermType = 67
	TermFunc     TermType = 69
	TermAsc      TermType = 73
	TermDesc     TermType = 74
	TermDefault  TermType = 92

	// type / coercion
	TermCoerceTo TermType = 51
	TermTypeOf   TermType = 52
	TermInfo     TermType = 79

	// string operators
	TermMatch    TermType = 97
	TermUpcase   TermType = 141
	TermDowncase TermType = 142
	TermSplit    TermType = 149

	// JSON / HTTP
	TermJSON TermType = 98
	TermHTTP TermType = 153

	// time constructors
	TermISO8601     TermType = 99
	TermToISO8601   TermType = 100
	TermEpochTime   TermType = 101
	TermToEpochTime TermType = 102
	TermNow         TermType = 103
	TermInTimezone  TermType = 104
	TermDuring      TermType = 105
	TermDate        TermType = 106
	TermTimeOfDay   TermType = 126
	TermTimezone    TermType = 127
	TermYear        TermType = 128
	TermMonth       TermType = 129
	TermDay         TermType = 130
	TermDayOfWeek   TermType = 131
	TermDayOfYear   TermType = 132
	TermHours       TermType = 133
	TermMinutes     TermType = 134
	TermSeconds     TermType = 135
	TermTime        TermType = 136

	// day-of-week constants
	TermMonday    TermType = 107
	TermTuesday   TermType = 108
	TermWednesday TermType = 109
	TermThursday  TermType = 110
	TermFriday    TermType = 111
	TermSaturday  TermType = 112
	TermSunday    TermType = 113

	// month constants
	TermJanuary   TermType = 114
	TermFebruary  TermType = 115
	TermMarch     TermType = 116
	TermApril     TermType = 117
	TermMay       TermType = 118
	TermJune      TermType = 119
	TermJuly      TermType = 120
	TermAugust    TermType = 121
	TermSeptember TermType = 122
	TermOctober   TermType = 123
	TermNovember  TermType = 124
	TermDecember  TermType = 125

	// geospatial
	TermGeoJSON         TermType = 157
	TermToGeoJSON       TermType = 158
	TermPoint           TermType = 159
	TermLine            TermType = 160
	TermPolygon         TermType = 161
	TermDistance        TermType = 162
	TermIntersects      TermType = 163
	TermIncludes        TermType = 164
	TermCircle          TermType = 165
	TermGetIntersecting TermType = 166
	TermFill            TermType = 167
	TermGetNearest      TermType = 168
	TermPolygonSub      TermType = 171
)
