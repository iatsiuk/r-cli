# Plan: ReQL Extended Operations

## Overview

Extend ReQL term builder with joins, string operations, time operations, array operations, control flow, additional sequence/object operations, admin operations, geospatial operations, and additional arithmetic.

Package: `internal/reql`

Depends on: `04-reql-core`, `06-reql-functions-index`

## Validation Commands
- `go test ./internal/reql/... -race -count=1`
- `make build`

### Task 1: Join operations

- [ ] Test: `INNER_JOIN(48)` -> `[48,[<seq>,<seq>,<func>]]`
- [ ] Test: `OUTER_JOIN(49)` -> `[49,[<seq>,<seq>,<func>]]`
- [ ] Test: `EQ_JOIN(50)` with index optarg -> `[50,[<seq>,"field",<table>],{"index":"name"}]`
- [ ] Test: `ZIP(72)` -> `[72,[<term>]]`
- [ ] Test: eqJoin with index optarg, innerJoin with predicate function, zip after join
- [ ] Implement: InnerJoin, OuterJoin, EqJoin, Zip methods

### Task 2: String operations

- [ ] Test: `MATCH(97)` -> `[97,[<term>,"pattern"]]`
- [ ] Test: `SPLIT(149)` with delimiter -> `[149,[<term>,"delim"]]`
- [ ] Test: `SPLIT(149)` without delimiter -> `[149,[<term>]]`
- [ ] Test: `UPCASE(141)` -> `[141,[<term>]]`
- [ ] Test: `DOWNCASE(142)` -> `[142,[<term>]]`
- [ ] Test: `TO_JSON_STRING(172)` -> `[172,[<term>]]`
- [ ] Test: `JSON(98)` -> `[98,["json_string"]]`
- [ ] Implement: Match, Split, Upcase, Downcase, ToJsonString, Json methods

### Task 3: Time operations

Construction:
- [ ] Test: `ISO8601(99)` -> `[99,["2024-01-01T00:00:00Z"]]`
- [ ] Test: `EPOCH_TIME(101)` -> `[101,[1234567890]]`
- [ ] Test: `TIME(136)` -> `[136,[2024,1,1,"Z"]]`
- [ ] Test: `NOW(103)` -> `[103,[]]`

Extraction:
- [ ] Test: `TO_ISO8601(100)` -> `[100,[<time_term>]]`
- [ ] Test: `TO_EPOCH_TIME(102)` -> `[102,[<time_term>]]`
- [ ] Test: `DATE(106)` -> `[106,[<time_term>]]`
- [ ] Test: `TIME_OF_DAY(126)` -> `[126,[<time_term>]]`
- [ ] Test: `TIMEZONE(127)` -> `[127,[<time_term>]]`
- [ ] Test: `YEAR(128)` -> `[128,[<time_term>]]`
- [ ] Test: `MONTH(129)` -> `[129,[<time_term>]]`
- [ ] Test: `DAY(130)` -> `[130,[<time_term>]]`
- [ ] Test: `DAY_OF_WEEK(131)` -> `[131,[<time_term>]]`
- [ ] Test: `DAY_OF_YEAR(132)` -> `[132,[<time_term>]]`
- [ ] Test: `HOURS(133)` -> `[133,[<time_term>]]`
- [ ] Test: `MINUTES(134)` -> `[134,[<time_term>]]`
- [ ] Test: `SECONDS(135)` -> `[135,[<time_term>]]`

Operations:
- [ ] Test: `IN_TIMEZONE(104)` -> `[104,[<time_term>,"+02:00"]]`
- [ ] Test: `DURING(105)` -> `[105,[<time_term>,<start>,<end>]]`

Constants:
- [ ] Test: `MONDAY(107)` through `SUNDAY(113)` -> `[107,[]]` .. `[113,[]]`
- [ ] Test: `JANUARY(114)` through `DECEMBER(125)` -> `[114,[]]` .. `[125,[]]`
- [ ] Implement: all time construction, extraction, operation, and constant methods

### Task 4: Array operations

- [ ] Test: `APPEND(29)` -> `[29,[<term>,<value>]]`
- [ ] Test: `PREPEND(80)` -> `[80,[<term>,<value>]]`
- [ ] Test: `SLICE(30)` -> `[30,[<term>,<start>,<end>]]`
- [ ] Test: `DIFFERENCE(95)` -> `[95,[<term>,<array>]]`
- [ ] Test: `INSERT_AT(82)` -> `[82,[<term>,<index>,<value>]]`
- [ ] Test: `DELETE_AT(83)` -> `[83,[<term>,<index>]]`
- [ ] Test: `CHANGE_AT(84)` -> `[84,[<term>,<index>,<value>]]`
- [ ] Test: `SPLICE_AT(85)` -> `[85,[<term>,<index>,<array>]]`
- [ ] Test: `SET_INSERT(88)` -> `[88,[<term>,<value>]]`
- [ ] Test: `SET_INTERSECTION(89)` -> `[89,[<term>,<array>]]`
- [ ] Test: `SET_UNION(90)` -> `[90,[<term>,<array>]]`
- [ ] Test: `SET_DIFFERENCE(91)` -> `[91,[<term>,<array>]]`
- [ ] Implement: Append, Prepend, Slice, Difference, InsertAt, DeleteAt, ChangeAt, SpliceAt, SetInsert, SetIntersection, SetUnion, SetDifference methods

### Task 5: Control flow

- [ ] Test: `BRANCH(65)` -> `[65,[<cond>,<true_val>,<false_val>]]`
- [ ] Test: `BRANCH(65)` with multiple condition pairs -> `[65,[<c1>,<v1>,<c2>,<v2>,<else>]]`
- [ ] Test: `FOR_EACH(68)` -> `[68,[<seq>,<func>]]`
- [ ] Test: `DEFAULT(92)` -> `[92,[<term>,<default_val>]]`
- [ ] Test: `ERROR(12)` -> `[12,["message"]]`
- [ ] Test: `COERCE_TO(51)` -> `[51,[<term>,"string"]]`
- [ ] Test: `TYPE_OF(52)` -> `[52,[<term>]]`
- [ ] Implement: Branch, ForEach, Default, Error, CoerceTo, TypeOf methods

### Task 6: Additional sequence operations

- [ ] Test: `CONCAT_MAP(40)` -> `[40,[<seq>,<func>]]`
- [ ] Test: `NTH(45)` -> `[45,[<seq>,<index>]]`
- [ ] Test: `UNION(44)` -> `[44,[<seq>,<seq>]]`
- [ ] Test: `UNION(44)` with multiple sequences -> `[44,[<seq>,<seq>,<seq>]]`
- [ ] Test: `IS_EMPTY(86)` -> `[86,[<seq>]]`
- [ ] Test: `CONTAINS(93)` -> `[93,[<seq>,<value>]]`
- [ ] Test: `BRACKET(170)` -> `[170,[<term>,"field"]]`
- [ ] Test: `BRACKET(170)` chained -> `[170,[[170,[<term>,"a"]],"b"]]`
- [ ] Implement: ConcatMap, Nth, Union, IsEmpty, Contains, Bracket methods

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
