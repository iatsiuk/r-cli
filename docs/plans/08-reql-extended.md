# Plan: ReQL Extended Operations

## Overview

Extend ReQL term builder with joins, string operations, time operations, array operations, control flow, additional sequence/object operations, admin operations, geospatial operations, and additional arithmetic.

Package: `internal/reql`

Depends on: `04-reql-core`, `06-reql-functions-index`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Join operations

- [x] Test: `INNER_JOIN(48)` -> `[48,[<seq>,<seq>,<func>]]`
- [x] Test: `OUTER_JOIN(49)` -> `[49,[<seq>,<seq>,<func>]]`
- [x] Test: `EQ_JOIN(50)` with index optarg -> `[50,[<seq>,"field",<table>],{"index":"name"}]`
- [x] Test: `ZIP(72)` -> `[72,[<term>]]`
- [x] Test: eqJoin with index optarg, innerJoin with predicate function, zip after join
- [x] Implement: InnerJoin, OuterJoin, EqJoin, Zip methods

### Task 2: String operations

- [x] Test: `MATCH(97)` -> `[97,[<term>,"pattern"]]`
- [x] Test: `SPLIT(149)` with delimiter -> `[149,[<term>,"delim"]]`
- [x] Test: `SPLIT(149)` without delimiter -> `[149,[<term>]]`
- [x] Test: `UPCASE(141)` -> `[141,[<term>]]`
- [x] Test: `DOWNCASE(142)` -> `[142,[<term>]]`
- [x] Test: `TO_JSON_STRING(172)` -> `[172,[<term>]]`
- [x] Test: `JSON(98)` -> `[98,["json_string"]]`
- [x] Implement: Match, Split, Upcase, Downcase, ToJsonString, Json methods

### Task 3: Time operations

Construction:
- [x] Test: `ISO8601(99)` -> `[99,["2024-01-01T00:00:00Z"]]`
- [x] Test: `EPOCH_TIME(101)` -> `[101,[1234567890]]`
- [x] Test: `TIME(136)` -> `[136,[2024,1,1,"Z"]]`
- [x] Test: `NOW(103)` -> `[103,[]]`

Extraction:
- [x] Test: `TO_ISO8601(100)` -> `[100,[<time_term>]]`
- [x] Test: `TO_EPOCH_TIME(102)` -> `[102,[<time_term>]]`
- [x] Test: `DATE(106)` -> `[106,[<time_term>]]`
- [x] Test: `TIME_OF_DAY(126)` -> `[126,[<time_term>]]`
- [x] Test: `TIMEZONE(127)` -> `[127,[<time_term>]]`
- [x] Test: `YEAR(128)` -> `[128,[<time_term>]]`
- [x] Test: `MONTH(129)` -> `[129,[<time_term>]]`
- [x] Test: `DAY(130)` -> `[130,[<time_term>]]`
- [x] Test: `DAY_OF_WEEK(131)` -> `[131,[<time_term>]]`
- [x] Test: `DAY_OF_YEAR(132)` -> `[132,[<time_term>]]`
- [x] Test: `HOURS(133)` -> `[133,[<time_term>]]`
- [x] Test: `MINUTES(134)` -> `[134,[<time_term>]]`
- [x] Test: `SECONDS(135)` -> `[135,[<time_term>]]`

Operations:
- [x] Test: `IN_TIMEZONE(104)` -> `[104,[<time_term>,"+02:00"]]`
- [x] Test: `DURING(105)` -> `[105,[<time_term>,<start>,<end>]]`

Constants:
- [x] Test: `MONDAY(107)` through `SUNDAY(113)` -> `[107,[]]` .. `[113,[]]`
- [x] Test: `JANUARY(114)` through `DECEMBER(125)` -> `[114,[]]` .. `[125,[]]`
- [x] Implement: all time construction, extraction, operation, and constant methods

### Task 4: Array operations

- [x] Test: `APPEND(29)` -> `[29,[<term>,<value>]]`
- [x] Test: `PREPEND(80)` -> `[80,[<term>,<value>]]`
- [x] Test: `SLICE(30)` -> `[30,[<term>,<start>,<end>]]`
- [x] Test: `DIFFERENCE(95)` -> `[95,[<term>,<array>]]`
- [x] Test: `INSERT_AT(82)` -> `[82,[<term>,<index>,<value>]]`
- [x] Test: `DELETE_AT(83)` -> `[83,[<term>,<index>]]`
- [x] Test: `CHANGE_AT(84)` -> `[84,[<term>,<index>,<value>]]`
- [x] Test: `SPLICE_AT(85)` -> `[85,[<term>,<index>,<array>]]`
- [x] Test: `SET_INSERT(88)` -> `[88,[<term>,<value>]]`
- [x] Test: `SET_INTERSECTION(89)` -> `[89,[<term>,<array>]]`
- [x] Test: `SET_UNION(90)` -> `[90,[<term>,<array>]]`
- [x] Test: `SET_DIFFERENCE(91)` -> `[91,[<term>,<array>]]`
- [x] Implement: Append, Prepend, Slice, Difference, InsertAt, DeleteAt, ChangeAt, SpliceAt, SetInsert, SetIntersection, SetUnion, SetDifference methods

### Task 5: Control flow

- [x] Test: `BRANCH(65)` -> `[65,[<cond>,<true_val>,<false_val>]]`
- [x] Test: `BRANCH(65)` with multiple condition pairs -> `[65,[<c1>,<v1>,<c2>,<v2>,<else>]]`
- [x] Test: `FOR_EACH(68)` -> `[68,[<seq>,<func>]]`
- [x] Test: `DEFAULT(92)` -> `[92,[<term>,<default_val>]]`
- [x] Test: `ERROR(12)` -> `[12,["message"]]`
- [x] Test: `COERCE_TO(51)` -> `[51,[<term>,"string"]]`
- [x] Test: `TYPE_OF(52)` -> `[52,[<term>]]`
- [x] Implement: Branch, ForEach, Default, Error, CoerceTo, TypeOf methods

### Task 6: Additional sequence operations

- [x] Test: `CONCAT_MAP(40)` -> `[40,[<seq>,<func>]]`
- [x] Test: `NTH(45)` -> `[45,[<seq>,<index>]]`
- [x] Test: `UNION(44)` -> `[44,[<seq>,<seq>]]`
- [x] Test: `UNION(44)` with multiple sequences -> `[44,[<seq>,<seq>,<seq>]]`
- [x] Test: `IS_EMPTY(86)` -> `[86,[<seq>]]`
- [x] Test: `CONTAINS(93)` -> `[93,[<seq>,<value>]]`
- [x] Test: `BRACKET(170)` -> `[170,[<term>,"field"]]`
- [x] Test: `BRACKET(170)` chained -> `[170,[[170,[<term>,"a"]],"b"]]`
- [x] Implement: ConcatMap, Nth, Union, IsEmpty, Contains, Bracket methods

### Task 7: Additional object operations

- [ ] Test: `WITH_FIELDS(96)` -> `[96,[<seq>,"field1","field2"]]`
- [ ] Test: `KEYS(94)` -> `[94,[<term>]]`
- [ ] Test: `VALUES(186)` -> `[186,[<term>]]`
- [ ] Test: `LITERAL(137)` -> `[137,[<value>]]`
- [ ] Test: `LITERAL(137)` in merge -> merge uses literal to replace nested field
- [ ] Implement: WithFields, Keys, Values, Literal methods

### Task 8: Admin operations

- [ ] Test: `SYNC(138)` -> `[138,[<table_term>]]`
- [ ] Test: `RECONFIGURE(176)` -> `[176,[<table_term>],{"shards":2,"replicas":1}]`
- [ ] Test: `REBALANCE(179)` -> `[179,[<table_term>]]`
- [ ] Test: `WAIT(177)` -> `[177,[<table_term>]]`
- [ ] Test: `ARGS(154)` -> `[154,[<array>]]`
- [ ] Test: `MINVAL(180)` -> `[180,[]]`
- [ ] Test: `MAXVAL(181)` -> `[181,[]]`
- [ ] Test: Between with minval/maxval -> `[182,[<seq>,[180,[]],[181,[]]]]`
- [ ] Implement: Sync, Reconfigure, Rebalance, Wait, Args, MinVal, MaxVal methods

### Task 9: Geospatial operations

- [ ] Test: `GEOJSON(157)` -> `[157,[<geojson_obj>]]`
- [ ] Test: `TO_GEOJSON(158)` -> `[158,[<geo_term>]]`
- [ ] Test: `POINT(159)` -> `[159,[-122.4,37.7]]`
- [ ] Test: `LINE(160)` -> `[160,[[159,[-122.4,37.7]],[159,[-122.3,37.8]]]]`
- [ ] Test: `POLYGON(161)` -> `[161,[[159,[...]],[159,[...]],[159,[...]]]]`
- [ ] Test: `CIRCLE(165)` -> `[165,[[159,[-122.4,37.7]],1000],{"unit":"m"}]`
- [ ] Test: `DISTANCE(162)` -> `[162,[<geo1>,<geo2>],{"unit":"km"}]`
- [ ] Test: `INTERSECTS(163)` -> `[163,[<geo1>,<geo2>]]`
- [ ] Test: `INCLUDES(164)` -> `[164,[<geo>,<point>]]`
- [ ] Test: `GET_INTERSECTING(166)` with index optarg -> `[166,[<table>,<geo>],{"index":"location"}]`
- [ ] Test: `GET_NEAREST(168)` with index optarg -> `[168,[<table>,<point>],{"index":"location"}]`
- [ ] Test: `FILL(167)` -> `[167,[<line_term>]]`
- [ ] Test: `POLYGON_SUB(171)` -> `[171,[<polygon1>,<polygon2>]]`
- [ ] Implement: GeoJson, ToGeoJson, Point, Line, Polygon, Circle, Distance, Intersects, Includes, GetIntersecting, GetNearest, Fill, PolygonSub methods

### Task 10: Additional arithmetic

- [ ] Test: `MOD(28)` -> `[28,[<term>,<value>]]`
- [ ] Test: `FLOOR(183)` -> `[183,[<term>]]`
- [ ] Test: `CEIL(184)` -> `[184,[<term>]]`
- [ ] Test: `ROUND(185)` -> `[185,[<term>]]`
- [ ] Implement: Mod, Floor, Ceil, Round methods
