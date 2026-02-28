# RethinkDB JavaScript API Reference

## Accessing ReQL

- [ ] ### r

**Syntax:** `r`

**Description:** The top-level ReQL namespace.

**Example:**
```js
var r = require('rethinkdb');
```

- [ ] ### connect

**Syntax:**
```
r.connect([options, ]callback)
r.connect([host, ]callback)
r.connect([options]) → promise
r.connect([host]) → promise
```

**Description:** Create a new connection to the database server.

**Example:**
```js
r.connect({
    host: 'localhost',
    port: 28015,
    db: 'marvel'
}, function(err, conn) {
    // ...
});
```

- [ ] ### close

**Syntax:**
```
conn.close([{noreplyWait: true}, ]callback)
conn.close([{noreplyWait: true}]) → promise
```

**Description:** Close an open connection.

**Example:**
```js
conn.close(function(err) { if (err) throw err; })
```

- [ ] ### reconnect

**Syntax:**
```
conn.reconnect([{noreplyWait: true}, ]callback)
conn.reconnect([{noreplyWait: true}]) → promise
```

**Description:** Close and reopen a connection.

**Example:**
```js
conn.reconnect({noreplyWait: false}, function(error, connection) { ... })
```

- [ ] ### use

**Syntax:** `conn.use(dbName)`

**Description:** Change the default database on this connection.

**Example:**
```js
conn.use('marvel')
r.table('heroes').run(conn, ...) // refers to r.db('marvel').table('heroes')
```

- [ ] ### run

**Syntax:**
```
query.run(conn[, options], callback)
query.run(conn[, options]) → promise
```

**Description:** Run a query on a connection, returning either an error, a single JSON result, or a cursor, depending on the query.

**Example:**
```js
r.table('marvel').run(conn, function(err, cursor) {
    cursor.each(console.log);
})
```

- [x] ### changes

**Syntax:**
```
stream.changes([options]) → stream
singleSelection.changes([options]) → stream
```

**Description:** Turn a query into a changefeed, an infinite stream of objects representing changes to the query's results as they occur.

**Example:**
```js
r.table('games').changes().run(conn, function(err, cursor) {
  cursor.each(console.log);
});
```

- [ ] ### noreplyWait

**Syntax:**
```
conn.noreplyWait(callback)
conn.noreplyWait() → promise
```

**Description:** Ensures that previous queries with the `noreply` flag have been processed by the server.

**Example:**
```js
conn.noreplyWait().then(function() {
    // all queries have been processed
}).error(function(err) {
    // process error
})
```

- [ ] ### server

**Syntax:**
```
conn.server(callback)
conn.server() → promise
```

**Description:** Return information about the server being used by a connection.

**Example:**
```js
conn.server(callback);
// Result: { "id": "404bef53-...", "name": "amadeus", "proxy": false }
```

- [ ] ### EventEmitter (connection)

**Syntax:**
```
connection.on(event, listener)
connection.addListener(event, listener)
connection.once(event, listener)
connection.removeListener(event, listener)
```

**Description:** Connections implement the same interface as Node's EventEmitter, allowing you to listen for changes in connection state.

**Example:**
```js
r.connect({}, function(err, conn) {
    if (err) throw err;
    conn.addListener('error', function(e) {
        processNetworkError(e);
    });
    conn.addListener('close', function() {
        cleanup();
    });
    runQueries(conn);
});
```

---

## Cursors

- [ ] ### next

**Syntax:**
```
cursor.next(callback)
cursor.next() → promise
```

**Description:** Get the next element in the cursor.

**Example:**
```js
cursor.next(function(err, row) {
    if (err) throw err;
    processRow(row);
});
```

- [ ] ### each

**Syntax:**
```
cursor.each(callback[, onFinishedCallback])
feed.each(callback)
```

**Description:** Lazily iterate over the result set one element at a time.

**Example:**
```js
cursor.each(function(err, row) {
    if (err) throw err;
    processRow(row);
});
```

- [ ] ### eachAsync

**Syntax:** `sequence.eachAsync(function[, errorFunction]) → promise`

**Description:** Lazily iterate over a cursor, array, or feed one element at a time.

**Example:**
```js
cursor.eachAsync(function (row) {
    var ok = processRowData(row);
    if (!ok) {
        throw new Error('Bad row: ' + row);
    }
}).then(function () {
    console.log('done processing');
}).catch(function (error) {
    console.log('Error:', error.message);
});
```

- [ ] ### toArray

**Syntax:**
```
cursor.toArray(callback)
cursor.toArray() → promise
```

**Description:** Retrieve all results and pass them as an array to the given callback.

**Example:**
```js
cursor.toArray(function(err, results) {
    if (err) throw err;
    processResults(results);
});
```

- [ ] ### close (cursor)

**Syntax:**
```
cursor.close([callback])
cursor.close() → promise
```

**Description:** Close a cursor, cancelling the corresponding query and freeing the memory associated with the open request.

**Example:**
```js
cursor.close(function (err) {
    if (err) {
        console.log("An error occurred on cursor close");
    }
});
```

- [ ] ### EventEmitter (cursor)

**Syntax:**
```
cursor.on(event, listener)
cursor.addListener(event, listener)
cursor.once(event, listener)
cursor.removeListener(event, listener)
```

**Description:** Cursors and feeds implement the same interface as Node's EventEmitter, enabling event-driven data handling with `data` and `error` events.

**Example:**
```js
r.table("messages").orderBy({index: "date"}).run(conn, function(err, cursor) {
    cursor.on("error", function(error) {
        // Handle error
    })
    cursor.on("data", function(message) {
        socket.broadcast.emit("message", message)
    })
});
```

---

## Manipulating Databases

- [x] ### dbCreate

**Syntax:** `r.dbCreate(dbName) → object`

**Description:** Create a database.

**Example:**
```js
r.dbCreate('superheroes').run(conn, callback);
```

- [x] ### dbDrop

**Syntax:** `r.dbDrop(dbName) → object`

**Description:** Drop a database, including all its tables and data.

**Example:**
```js
r.dbDrop('superheroes').run(conn, callback);
```

- [x] ### dbList

**Syntax:** `r.dbList() → array`

**Description:** List all database names in the system.

**Example:**
```js
r.dbList().run(conn, callback);
```

---

## Manipulating Tables

- [x] ### tableCreate

**Syntax:** `db.tableCreate(tableName[, options]) → object`

**Description:** Create a table in a database.

**Example:**
```js
r.db('heroes').tableCreate('dc_universe').run(conn, callback);
```

- [x] ### tableDrop

**Syntax:** `db.tableDrop(tableName) → object`

**Description:** Drop a table from a database, deleting all its data.

**Example:**
```js
r.db('test').tableDrop('dc_universe').run(conn, callback);
```

- [x] ### tableList

**Syntax:** `db.tableList() → array`

**Description:** List all table names in a database.

**Example:**
```js
r.db('test').tableList().run(conn, callback);
```

- [x] ### indexCreate

**Syntax:** `table.indexCreate(indexName[, indexFunction][, {multi: false, geo: false}]) → object`

**Description:** Create a new secondary index on a table.

**Example:**
```js
r.table('comments').indexCreate('postId').run(conn, callback);
```

- [x] ### indexDrop

**Syntax:** `table.indexDrop(indexName) → object`

**Description:** Delete a previously created secondary index of this table.

**Example:**
```js
r.table('dc').indexDrop('code_name').run(conn, callback);
```

- [x] ### indexList

**Syntax:** `table.indexList() → array`

**Description:** List all the secondary indexes of this table.

**Example:**
```js
r.table('marvel').indexList().run(conn, callback);
```

- [x] ### indexRename

**Syntax:** `table.indexRename(oldIndexName, newIndexName[, {overwrite: false}]) → object`

**Description:** Rename an existing secondary index on a table.

**Example:**
```js
r.table('comments').indexRename('postId', 'messageId').run(conn, callback);
```

- [x] ### indexStatus

**Syntax:** `table.indexStatus([, index...]) → array`

**Description:** Get the status of the specified indexes on this table, or the status of all indexes if none specified.

**Example:**
```js
r.table('test').indexStatus().run(conn, callback);
```

- [x] ### indexWait

**Syntax:** `table.indexWait([, index...]) → array`

**Description:** Wait for the specified indexes on this table to be ready.

**Example:**
```js
r.table('test').indexWait('timestamp').run(conn, callback);
```

- [ ] ### setWriteHook

**Syntax:** `table.setWriteHook(function | binary | null) → object`

**Description:** Sets the write hook on a table or overwrites it if one already exists.

**Example:**
```js
r.table('comments').setWriteHook(function(context, oldVal, newVal) {
  return newVal.merge({
    modified_at: context('timestamp')
  });
}).run(conn, callback);
```

- [ ] ### getWriteHook

**Syntax:** `table.getWriteHook() → null/object`

**Description:** Gets the write hook of this table.

**Example:**
```js
r.table('comments').getWriteHook().run(conn, callback);
```

---

## Writing Data

- [x] ### insert

**Syntax:** `table.insert(object | [object1, object2, ...][, {durability: "hard", returnChanges: false, conflict: "error", ignoreWriteHook: false}]) → object`

**Description:** Insert documents into a table, accepting a single document or an array of documents.

**Example:**
```js
r.table("posts").insert({
    id: 1,
    title: "Lorem ipsum",
    content: "Dolor sit amet"
}).run(conn, callback)
```

- [x] ### update

**Syntax:**
```
table.update(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
selection.update(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
singleSelection.update(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
```

**Description:** Update JSON documents in a table, accepting a JSON document, a ReQL expression, or a combination of the two.

**Example:**
```js
r.table("posts").get(1).update({
    views: r.row("views").add(1)
}).run(conn, callback)
```

- [x] ### replace

**Syntax:**
```
table.replace(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
selection.replace(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
singleSelection.replace(object | function[, {durability: "hard", returnChanges: false, nonAtomic: false, ignoreWriteHook: false}]) → object
```

**Description:** Replace documents in a table by substituting the original document with a new one having the same primary key.

**Example:**
```js
r.table("posts").get(1).replace({
    id: 1,
    title: "Lorem ipsum",
    content: "Aleas jacta est",
    status: "draft"
}).run(conn, callback)
```

- [x] ### delete

**Syntax:**
```
table.delete([{durability: "hard", returnChanges: false, ignoreWriteHook: false}]) → object
selection.delete([{durability: "hard", returnChanges: false, ignoreWriteHook: false}]) → object
singleSelection.delete([{durability: "hard", returnChanges: false, ignoreWriteHook: false}]) → object
```

**Description:** Delete one or more documents from a table.

**Example:**
```js
r.table("comments").get("7eab9e63-73f1-4f33-8ce4-95cbea626f59").delete().run(conn, callback)
```

- [x] ### sync

**Syntax:** `table.sync() → object`

**Description:** Ensures that writes on a given table are written to permanent storage.

**Example:**
```js
r.table('marvel').sync().run(conn, callback)
```

---

## Selecting Data

- [x] ### db

**Syntax:** `r.db(dbName) → db`

**Description:** Reference a database to designate which database a query operates against.

**Example:**
```js
r.db('heroes').table('marvel').run(conn, callback)
```

- [x] ### table

**Syntax:** `db.table(name[, {readMode: 'single', identifierFormat: 'name'}]) → table`

**Description:** Return all documents in a table.

**Example:**
```js
r.table('marvel').run(conn, callback)
```

- [x] ### get

**Syntax:** `table.get(key) → singleRowSelection`

**Description:** Get a document by primary key, returning null if no document matches.

**Example:**
```js
r.table('posts').get('a9849eef-7176-4411-935b-79a6e3c56a74').run(conn, callback)
```

- [x] ### getAll

**Syntax:** `table.getAll([key, key2...], [{index: 'id'}]) → selection`

**Description:** Get all documents where the given value matches the value of the requested index.

**Example:**
```js
r.table('marvel').getAll('man_of_steel', {index: 'code_name'}).run(conn, callback)
```

- [x] ### between

**Syntax:** `table.between(lowerKey, upperKey[, options]) → table_slice`

**Description:** Get all documents between two keys, with support for secondary indexes and configurable boundary conditions.

**Example:**
```js
r.table('marvel').between(10, 20).run(conn, callback)
```

- [x] ### filter

**Syntax:** `selection.filter(predicate_function[, {default: false}]) → selection`

**Description:** Return all the elements in a sequence for which the given predicate is true.

**Example:**
```js
r.table('users').filter({age: 30}).run(conn, callback)
```

---

## Joins

- [x] ### innerJoin

**Syntax:** `sequence.innerJoin(otherSequence, predicate_function) → stream`

**Description:** Returns an inner join of two sequences.

**Example:**
```js
r.table('marvel').innerJoin(r.table('dc'), function(marvelRow, dcRow) {
    return marvelRow('strength').lt(dcRow('strength'))
}).zip().run(conn, callback)
```

- [x] ### outerJoin

**Syntax:** `sequence.outerJoin(otherSequence, predicate_function) → stream`

**Description:** Returns a left outer join of two sequences.

**Example:**
```js
r.table('marvel').outerJoin(r.table('dc'), function(marvelRow, dcRow) {
    return marvelRow('strength').lt(dcRow('strength'))
}).run(conn, callback)
```

- [x] ### eqJoin

**Syntax:** `sequence.eqJoin(leftField, rightTable[, {index: 'id', ordered: false}]) → sequence`

**Description:** Join tables using a field or function on the left-hand sequence matching primary keys or secondary indexes on the right-hand table.

**Example:**
```js
r.table('players').eqJoin('gameId', r.table('games')).without({right: "id"}).zip().orderBy('gameId').run(conn, callback)
```

- [x] ### zip

**Syntax:** `stream.zip() → stream`

**Description:** Used to 'zip' up the result of a join by merging the 'right' fields into 'left' fields of each member of the sequence.

**Example:**
```js
r.table('marvel').eqJoin('main_dc_collaborator', r.table('dc'))
    .zip().run(conn, callback)
```

---

## Transformations

- [x] ### map

**Syntax:** `sequence1.map([sequence2, ...], function) → stream`

**Description:** Transform each element of one or more sequences by applying a mapping function to them.

**Example:**
```js
r.expr([1, 2, 3, 4, 5]).map(function (val) {
    return val.mul(val);
}).run(conn, callback);
```

- [x] ### withFields

**Syntax:** `sequence.withFields([selector1, selector2...]) → stream`

**Description:** Plucks one or more attributes from a sequence of objects, filtering out any objects that do not have the specified fields.

**Example:**
```js
r.table('users').withFields('id', 'user', 'posts').run(conn, callback)
```

- [x] ### concatMap

**Syntax:** `stream.concatMap(function) → stream`

**Description:** Concatenate one or more elements into a single sequence using a mapping function.

**Example:**
```js
r.expr([1, 2, 3]).concatMap(function(x) {
    return [x, x.mul(2)]
}).run(conn, callback)
```

- [x] ### orderBy

**Syntax:**
```
table.orderBy([key | function...], {index: index_name}) → table_slice
selection.orderBy(key | function[, ...]) → selection<array>
sequence.orderBy(key | function[, ...]) → array
```

**Description:** Sort the sequence by document values of the given key(s), defaulting to ascending order.

**Example:**
```js
r.table('posts').orderBy({index: 'date'}).run(conn, callback);
```

- [x] ### skip

**Syntax:** `sequence.skip(n) → stream`

**Description:** Skip a number of elements from the head of the sequence.

**Example:**
```js
r.table('marvel').orderBy('successMetric').skip(10).run(conn, callback)
```

- [x] ### limit

**Syntax:** `sequence.limit(n) → stream`

**Description:** End the sequence after the given number of elements.

**Example:**
```js
r.table('marvel').orderBy('belovedness').limit(10).run(conn, callback)
```

- [x] ### slice

**Syntax:** `selection.slice(startOffset[, endOffset, {leftBound:'closed', rightBound:'open'}]) → selection`

**Description:** Return the elements of a sequence within the specified range.

**Example:**
```js
r.expr([0,1,2,3,4,5]).slice(2,-2).run(conn, callback);
// Result: [2,3]
```

- [x] ### nth

**Syntax:** `sequence.nth(index) → object`

**Description:** Get the nth element of a sequence, counting from zero; negative values count from the last element.

**Example:**
```js
r.expr([1,2,3]).nth(1).run(conn, callback)
```

- [x] ### offsetsOf

**Syntax:** `sequence.offsetsOf(datum | predicate_function) → array`

**Description:** Get the indexes of an element in a sequence; if the argument is a predicate, get the indexes of all elements matching it.

**Example:**
```js
r.expr(['a','b','c']).offsetsOf('c').run(conn, callback)
```

- [x] ### isEmpty

**Syntax:** `sequence.isEmpty() → bool`

**Description:** Test if a sequence is empty.

**Example:**
```js
r.table('marvel').isEmpty().run(conn, callback)
```

- [x] ### union

**Syntax:** `stream.union(sequence[, sequence, ...][, {interleave: true}]) → stream`

**Description:** Merge two or more sequences.

**Example:**
```js
r.expr([1, 2]).union([3, 4], [5, 6], [7, 8, 9]).run(conn, callback)
// Result: [1, 2, 3, 4, 5, 6, 7, 8, 9]
```

- [x] ### sample

**Syntax:** `sequence.sample(number) → selection`

**Description:** Select a given number of elements from a sequence with uniform random distribution, without replacement.

**Example:**
```js
r.table('marvel').sample(3).run(conn, callback)
```

---

## Aggregation

- [x] ### group

**Syntax:** `sequence.group(field | function..., [{index: <indexname>, multi: false}]) → grouped_stream`

**Description:** Takes a stream and partitions it into multiple groups based on the fields or functions provided.

**Example:**
```js
r.table('games').group('player').max('points').run(conn, callback)
```

- [x] ### ungroup

**Syntax:** `grouped_stream.ungroup() → array`

**Description:** Takes a grouped stream or grouped data and turns it into an array of objects representing the groups.

**Example:**
```js
r.table('games')
    .group('player').max('points')('points')
    .ungroup().orderBy(r.desc('reduction')).run(conn, callback)
```

- [x] ### reduce

**Syntax:** `sequence.reduce(function) → value`

**Description:** Produce a single value from a sequence through repeated application of a reduction function.

**Example:**
```js
r.table("posts").map(function(doc) {
    return 1;
}).reduce(function(left, right) {
    return left.add(right);
}).default(0).run(conn, callback);
```

- [x] ### fold

**Syntax:**
```
sequence.fold(base, function) → value
sequence.fold(base, function, {emit: function[, finalEmit: function]}) → sequence
```

**Description:** Apply a function to a sequence in order, maintaining state via an accumulator.

**Example:**
```js
r.table('words').orderBy('id').fold('', function (acc, word) {
    return acc.add(r.branch(acc.eq(''), '', ', ')).add(word);
}).run(conn, callback);
```

- [x] ### count

**Syntax:**
```
sequence.count([value | predicate_function]) → number
binary.count() → number
string.count() → number
object.count() → number
array.count() → number
r.count(sequence | binary | string | object[, predicate_function]) → number
```

**Description:** Counts the number of elements in a sequence or key/value pairs in an object, or returns the size of a string or binary object.

**Example:**
```js
r.table('users').count().run(conn, callback);
```

- [x] ### sum

**Syntax:** `sequence.sum([field | function]) → number`

**Description:** Sums all the elements of a sequence, skipping elements that lack the specified field.

**Example:**
```js
r.expr([3, 5, 7]).sum().run(conn, callback)
```

- [x] ### avg

**Syntax:** `sequence.avg([field | function]) → number`

**Description:** Averages all the elements of a sequence, skipping elements that lack the specified field.

**Example:**
```js
r.table('games').avg('points').run(conn, callback)
```

- [x] ### min

**Syntax:**
```
sequence.min(field | function) → element
sequence.min({index: <indexname>}) → element
r.min(sequence, field | function) → element
r.min(sequence, {index: <indexname>}) → element
```

**Description:** Finds the minimum element of a sequence.

**Example:**
```js
r.table('users').min('points').run(conn, callback);
```

- [x] ### max

**Syntax:**
```
sequence.max(field | function) → element
sequence.max({index: <indexname>}) → element
r.max(sequence, field | function) → element
r.max(sequence, {index: <indexname>}) → element
```

**Description:** Finds the maximum element of a sequence.

**Example:**
```js
r.table('users').max('points').run(conn, callback);
```

- [x] ### distinct

**Syntax:** `sequence.distinct() → array`

**Description:** Removes duplicates from elements in a sequence.

**Example:**
```js
r.table('marvel').concatMap(function(hero) {
    return hero('villainList')
}).distinct().run(conn, callback)
```

- [x] ### contains

**Syntax:** `sequence.contains([value | predicate_function, ...]) → bool`

**Description:** Returns true if a sequence contains all the specified values.

**Example:**
```js
r.table('marvel').get('ironman')('opponents').contains('superman').run(conn, callback);
```

---

## Document Manipulation

- [x] ### row

**Syntax:** `r.row → value`

**Description:** Returns the currently visited document.

**Example:**
```js
r.table('users').filter(r.row('age').gt(5)).run(conn, callback)
```

- [x] ### pluck

**Syntax:** `sequence.pluck([selector1, selector2...]) → stream`

**Description:** Plucks out one or more attributes from either an object or a sequence of objects (projection).

**Example:**
```js
r.table('marvel').get('IronMan').pluck('reactorState', 'reactorPower').run(conn, callback)
```

- [x] ### without

**Syntax:** `sequence.without([selector1, selector2...]) → stream`

**Description:** The opposite of pluck; returns objects with the specified paths removed.

**Example:**
```js
r.table('marvel').get('IronMan').without('personalVictoriesList').run(conn, callback)
```

- [x] ### merge

**Syntax:** `singleSelection.merge([object | function, object | function, ...]) → object`

**Description:** Merge two or more objects together to construct a new object with properties from all.

**Example:**
```js
r.table('marvel').get('thor').merge(
    r.table('equipment').get('hammer'),
    r.table('equipment').get('pimento_sandwich')
).run(conn, callback)
```

- [x] ### append

**Syntax:** `array.append(value) → array`

**Description:** Append a value to an array.

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').append('newBoots').run(conn, callback)
```

- [x] ### prepend

**Syntax:** `array.prepend(value) → array`

**Description:** Prepend a value to an array.

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').prepend('newBoots').run(conn, callback)
```

- [x] ### difference

**Syntax:** `array.difference(array) → array`

**Description:** Remove the elements of one array from another array.

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').difference(['Boots']).run(conn, callback)
```

- [x] ### setInsert

**Syntax:** `array.setInsert(value) → array`

**Description:** Add a value to an array and return it as a set (an array with distinct values).

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').setInsert('newBoots').run(conn, callback)
```

- [x] ### setUnion

**Syntax:** `array.setUnion(array) → array`

**Description:** Add several values to an array and return it as a set (an array with distinct values).

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').setUnion(['newBoots', 'arc_reactor']).run(conn, callback)
```

- [x] ### setIntersection

**Syntax:** `array.setIntersection(array) → array`

**Description:** Intersect two arrays returning values that occur in both of them as a set.

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').setIntersection(['newBoots', 'arc_reactor']).run(conn, callback)
```

- [x] ### setDifference

**Syntax:** `array.setDifference(array) → array`

**Description:** Remove the elements of one array from another and return them as a set.

**Example:**
```js
r.table('marvel').get('IronMan')('equipment').setDifference(['newBoots', 'arc_reactor']).run(conn, callback)
```

- [x] ### () (bracket)

**Syntax:** `sequence(attr) → sequence` / `singleSelection(attr) → value` / `object(attr) → value` / `array(index) → value`

**Description:** Get a single field from an object, or that field from every object in a sequence.

**Example:**
```js
r.table('marvel').get('IronMan')('firstAppearance').run(conn, callback)
```

- [x] ### getField

**Syntax:** `sequence.getField(attr) → sequence` / `singleSelection.getField(attr) → value` / `object.getField(attr) → value`

**Description:** Get a single field from an object; if called on a sequence, gets that field from every object in the sequence.

**Example:**
```js
r.table('marvel').get('IronMan').getField('firstAppearance').run(conn, callback)
```

- [x] ### hasFields

**Syntax:** `sequence.hasFields([selector1, selector2...]) → stream` / `object.hasFields([selector1, selector2...]) → boolean`

**Description:** Test if an object has one or more fields with non-null values.

**Example:**
```js
r.table('players').hasFields('games_won').run(conn, callback)
```

- [x] ### insertAt

**Syntax:** `array.insertAt(offset, value) → array`

**Description:** Insert a value in to an array at a given index.

**Example:**
```js
r.expr(["Iron Man", "Spider-Man"]).insertAt(1, "Hulk").run(conn, callback)
```

- [x] ### spliceAt

**Syntax:** `array.spliceAt(offset, array) → array`

**Description:** Insert several values in to an array at a given index.

**Example:**
```js
r.expr(["Iron Man", "Spider-Man"]).spliceAt(1, ["Hulk", "Thor"]).run(conn, callback)
```

- [x] ### deleteAt

**Syntax:** `array.deleteAt(offset [,endOffset]) → array`

**Description:** Remove one or more elements from an array at a given index.

**Example:**
```js
r(['a','b','c','d','e','f']).deleteAt(1).run(conn, callback)
```

- [x] ### changeAt

**Syntax:** `array.changeAt(offset, value) → array`

**Description:** Change a value in an array at a given index.

**Example:**
```js
r.expr(["Iron Man", "Bruce", "Spider-Man"]).changeAt(1, "Hulk").run(conn, callback)
```

- [x] ### keys

**Syntax:** `singleSelection.keys() → array` / `object.keys() → array`

**Description:** Return an array containing all of an object's keys.

**Example:**
```js
r.table('users').get(1).keys().run(conn, callback)
```

- [x] ### values

**Syntax:** `singleSelection.values() → array` / `object.values() → array`

**Description:** Return an array containing all of an object's values.

**Example:**
```js
r.table('users').get(1).values().run(conn, callback)
```

- [x] ### literal

**Syntax:** `r.literal(object) → special`

**Description:** Replace an object in a field instead of merging it with an existing object in a merge or update operation.

**Example:**
```js
r.table('users').get(1).update({
  data: r.literal({ age: 19, job: 'Engineer' })
}).run(conn, callback)
```

- [x] ### object

**Syntax:** `r.object([key, value,]...) → object`

**Description:** Creates an object from a list of key-value pairs, where the keys must be strings.

**Example:**
```js
r.object('id', 5, 'data', ['foo', 'bar']).run(conn, callback)
```

---

## String Manipulation

- [x] ### match

**Syntax:** `string.match(regexp) → null/object`

**Description:** Matches against a regular expression, returning an object with the fields `str`, `start`, `end`, `groups` if there is a match, or `null` otherwise.

**Example:**
```js
r.expr("name@domain.com").match(".*@(.*)").run(conn, callback)
```

- [x] ### split

**Syntax:** `string.split([separator, [max_splits]]) → array`

**Description:** Splits a string into substrings, splitting on whitespace when called with no arguments or on the given separator otherwise.

**Example:**
```js
r.expr("12,37,,22,").split(",").run(conn, callback)
```

- [x] ### upcase

**Syntax:** `string.upcase() → string`

**Description:** Uppercases a string.

**Example:**
```js
r.expr("Sentence about LaTeX.").upcase().run(conn, callback)
```

- [x] ### downcase

**Syntax:** `string.downcase() → string`

**Description:** Lowercases a string.

**Example:**
```js
r.expr("Sentence about LaTeX.").downcase().run(conn, callback)
```

---

## Math and Logic

- [x] ### add

**Syntax:** `value.add(value[, value, ...]) → value`

**Description:** Sum two or more numbers, or concatenate two or more strings or arrays.

**Example:**
```js
r.expr(2).add(2).run(conn, callback)
```

- [x] ### sub

**Syntax:** `number.sub(number[, number, ...]) → number`

**Description:** Subtract two numbers.

**Example:**
```js
r.expr(2).sub(2).run(conn, callback)
```

- [x] ### mul

**Syntax:** `number.mul(number[, number, ...]) → number`

**Description:** Multiply two numbers, or make a periodic array.

**Example:**
```js
r.expr(2).mul(2).run(conn, callback)
```

- [x] ### div

**Syntax:** `number.div(number[, number, ...]) → number`

**Description:** Divide two numbers.

**Example:**
```js
r.expr(2).div(2).run(conn, callback)
```

- [x] ### mod

**Syntax:** `number.mod(number) → number`

**Description:** Find the remainder when dividing two numbers.

**Example:**
```js
r.expr(2).mod(2).run(conn, callback)
```

- [x] ### and

**Syntax:** `bool.and([bool, bool, ...]) → bool` / `r.and([bool, bool, ...]) → bool`

**Description:** Compute the logical "and" of one or more values.

**Example:**
```js
var x = true, y = true, z = true;
r.and(x, y, z).run(conn, callback);
```

- [x] ### or

**Syntax:** `bool.or([bool, bool, ...]) → bool` / `r.or([bool, bool, ...]) → bool`

**Description:** Compute the logical "or" of one or more values.

**Example:**
```js
var a = true, b = false;
r.expr(a).or(b).run(conn, callback);
```

- [x] ### eq

**Syntax:** `value.eq(value[, value, ...]) → bool`

**Description:** Test if two or more values are equal.

**Example:**
```js
r.table('users').get(1)('role').eq('administrator').run(conn, callback);
```

- [x] ### ne

**Syntax:** `value.ne(value[, value, ...]) → bool`

**Description:** Test if two or more values are not equal.

**Example:**
```js
r.table('users').get(1)('role').ne('administrator').run(conn, callback);
```

- [x] ### gt

**Syntax:** `value.gt(value[, value, ...]) → bool`

**Description:** Compare values, testing if the left-hand value is greater than the right-hand.

**Example:**
```js
r.table('players').get(1)('score').gt(10).run(conn, callback);
```

- [x] ### ge

**Syntax:** `value.ge(value[, value, ...]) → bool`

**Description:** Compare values, testing if the left-hand value is greater than or equal to the right-hand.

**Example:**
```js
r.table('players').get(1)('score').ge(10).run(conn, callback);
```

- [x] ### lt

**Syntax:** `value.lt(value[, value, ...]) → bool`

**Description:** Compare values, testing if the left-hand value is less than the right-hand.

**Example:**
```js
r.table('players').get(1)('score').lt(10).run(conn, callback);
```

- [x] ### le

**Syntax:** `value.le(value[, value, ...]) → bool`

**Description:** Compare values, testing if the left-hand value is less than or equal to the right-hand.

**Example:**
```js
r.table('players').get(1)('score').le(10).run(conn, callback);
```

- [x] ### not

**Syntax:** `bool.not() → bool` / `not(bool) → bool`

**Description:** Compute the logical inverse (not) of an expression.

**Example:**
```js
r.table('users').filter(function(user) {
    return user.hasFields('flag').not()
}).run(conn, callback)
```

- [x] ### random

**Syntax:** `r.random() → number` / `r.random(number[, number], {float: true}) → number` / `r.random(integer[, integer]) → integer`

**Description:** Generate a random number between given (or implied) bounds.

**Example:**
```js
r.random(100).run(conn, callback)
```

- [x] ### round

**Syntax:** `r.round(number) → number` / `number.round() → number`

**Description:** Rounds the given value to the nearest whole integer.

**Example:**
```js
r.round(12.345).run(conn, callback);
```

- [x] ### ceil

**Syntax:** `r.ceil(number) → number` / `number.ceil() → number`

**Description:** Rounds the given value up, returning the smallest integer greater than or equal to the given value.

**Example:**
```js
r.ceil(12.345).run(conn, callback);
```

- [x] ### floor

**Syntax:** `r.floor(number) → number` / `number.floor() → number`

**Description:** Rounds the given value down, returning the largest integer less than or equal to the given value.

**Example:**
```js
r.floor(12.345).run(conn, callback);
```

- [x] ### bitAnd

**Syntax:** `r.bitAnd(number) → number` / `r.bitAnd(number[, number, ...]) → number`

**Description:** Performs a bitwise AND operation on each pair of corresponding bits.

**Example:**
```js
r.expr(5).bitAnd(3).run(conn);
```

- [x] ### bitOr

**Syntax:** `r.bitOr(number) → number` / `r.bitOr(number[, number, ...]) → number`

**Description:** Performs a bitwise OR operation on each pair of corresponding bits.

**Example:**
```js
r.expr(5).bitOr(3).run(conn);
```

- [x] ### bitXor

**Syntax:** `r.bitXor(number) → number` / `r.bitXor(number[, number, ...]) → number`

**Description:** Performs a bitwise XOR (exclusive OR) operation on each pair of corresponding bits.

**Example:**
```js
r.expr(6).bitXor(4).run(conn);
```

- [x] ### bitNot

**Syntax:** `r.bitNot() → number`

**Description:** Performs a bitwise NOT (ones' complement) unary operation, performing logical negation on each bit.

**Example:**
```js
r.expr(7).bitNot().run(conn);
```

- [x] ### bitSal

**Syntax:** `r.bitSal(number) → number` / `r.bitSal(number[, number, ...]) → number`

**Description:** Performs an arithmetic left shift where bits that slide off the end disappear.

**Example:**
```js
r.expr(5).bitSal(4).run(conn);
```

- [x] ### bitSar

**Syntax:** `r.bitSar(number) → number` / `r.bitSar(number[, number, ...]) → number`

**Description:** Performs an arithmetic right shift preserving the sign of the number.

**Example:**
```js
r.expr(32).bitSar(3).run(conn);
```

---

## Dates and Times

- [x] ### now

**Syntax:** `r.now() → time`

**Description:** Return a time object representing the current time in UTC.

**Example:**
```js
r.table("users").insert({
    name: "John",
    subscription_date: r.now()
}).run(conn, callback)
```

- [x] ### time

**Syntax:** `r.time(year, month, day[, hour, minute, second], timezone) → time`

**Description:** Create a time object for a specific time.

**Example:**
```js
r.table("user").get("John").update({birthdate: r.time(1986, 11, 3, 'Z')}).run(conn, callback)
```

- [x] ### epochTime

**Syntax:** `r.epochTime(number) → time`

**Description:** Create a time object based on seconds since epoch, rounded to millisecond precision.

**Example:**
```js
r.table("user").get("John").update({birthdate: r.epochTime(531360000)}).run(conn, callback)
```

- [x] ### ISO8601

**Syntax:** `r.ISO8601(string[, {defaultTimezone:''}]) → time`

**Description:** Create a time object based on an ISO 8601 date-time string.

**Example:**
```js
r.table("user").get("John").update({birth: r.ISO8601('1986-11-03T08:30:00-07:00')}).run(conn, callback)
```

- [x] ### inTimezone

**Syntax:** `time.inTimezone(timezone) → time`

**Description:** Return a new time object with a different timezone.

**Example:**
```js
r.now().inTimezone('-08:00').hours().run(conn, callback)
```

- [x] ### timezone

**Syntax:** `time.timezone() → string`

**Description:** Return the timezone of the time object.

**Example:**
```js
r.table("users").filter(function(user) {
    return user("subscriptionDate").timezone().eq("-07:00")
})
```

- [x] ### during

**Syntax:** `time.during(startTime, endTime[, {leftBound: "closed", rightBound: "open"}]) → bool`

**Description:** Return whether a time is between two other times.

**Example:**
```js
r.table("posts").filter(
    r.row('date').during(r.time(2013, 12, 1, "Z"), r.time(2013, 12, 10, "Z"))
).run(conn, callback)
```

- [x] ### date

**Syntax:** `time.date() → time`

**Description:** Return a new time object only based on the day, month and year (the same day at 00:00).

**Example:**
```js
r.table("users").filter(function(user) {
    return user("birthdate").date().eq(r.now().date())
}).run(conn, callback)
```

- [x] ### timeOfDay

**Syntax:** `time.timeOfDay() → number`

**Description:** Return the number of seconds elapsed since the beginning of the day stored in the time object.

**Example:**
```js
r.table("posts").filter(
    r.row("date").timeOfDay().le(12*60*60)
).run(conn, callback)
```

- [x] ### year

**Syntax:** `time.year() → number`

**Description:** Return the year of a time object.

**Example:**
```js
r.table("users").filter(function(user) {
    return user("birthdate").year().eq(1986)
}).run(conn, callback)
```

- [x] ### month

**Syntax:** `time.month() → number`

**Description:** Return the month of a time object as a number between 1 and 12. For convenience, the terms `r.january`, `r.february`, `r.march`, `r.april`, `r.may`, `r.june`, `r.july`, `r.august`, `r.september`, `r.october`, `r.november` and `r.december` are defined and map to the appropriate integer.

**Example:**
```js
r.table("users").filter(
    r.row("birthdate").month().eq(11)
)
```

- [x] ### day

**Syntax:** `time.day() → number`

**Description:** Return the day of a time object as a number between 1 and 31.

**Example:**
```js
r.table("users").filter(
    r.row("birthdate").day().eq(24)
).run(conn, callback)
```

- [x] ### dayOfWeek

**Syntax:** `time.dayOfWeek() → number`

**Description:** Return the day of week of a time object as a number between 1 and 7 (ISO 8601). For convenience, the terms `r.monday`, `r.tuesday`, `r.wednesday`, `r.thursday`, `r.friday`, `r.saturday` and `r.sunday` are defined and map to the appropriate integer.

**Example:**
```js
r.table("users").filter(
    r.row("birthdate").dayOfWeek().eq(r.tuesday)
)
```

- [x] ### dayOfYear

**Syntax:** `time.dayOfYear() → number`

**Description:** Return the day of the year of a time object as a number between 1 and 366.

**Example:**
```js
r.table("users").filter(
    r.row("birthdate").dayOfYear().eq(1)
)
```

- [x] ### hours

**Syntax:** `time.hours() → number`

**Description:** Return the hour in a time object as a number between 0 and 23.

**Example:**
```js
r.table("posts").filter(function(post) {
    return post("date").hours().lt(4)
})
```

- [x] ### minutes

**Syntax:** `time.minutes() → number`

**Description:** Return the minute in a time object as a number between 0 and 59.

**Example:**
```js
r.table("posts").filter(function(post) {
    return post("date").minutes().lt(10)
})
```

- [x] ### seconds

**Syntax:** `time.seconds() → number`

**Description:** Return the seconds in a time object as a number between 0 and 59.999 (double precision).

**Example:**
```js
r.table("posts").filter(function(post) {
    return post("date").seconds().lt(30)
})
```

- [x] ### toISO8601

**Syntax:** `time.toISO8601() → string`

**Description:** Convert a time object to a string in ISO 8601 format.

**Example:**
```js
r.now().toISO8601().run(conn, callback)
```

- [x] ### toEpochTime

**Syntax:** `time.toEpochTime() → number`

**Description:** Convert a time object to its epoch time.

**Example:**
```js
r.now().toEpochTime().run(conn, callback)
```

---

## Control Structures

- [x] ### args

**Syntax:** `r.args(array) → special`

**Description:** A special term used to splice an array of arguments into another term.

**Example:**
```js
r.table('people').getAll(r.args(['Alice', 'Bob'])).run(conn, callback)
```

- [x] ### binary

**Syntax:** `r.binary(data) → binary`

**Description:** Encapsulate binary data within a query.

**Example:**
```js
var fs = require('fs');
fs.readFile('./defaultAvatar.png', function (err, avatarImage) {
    if (err) return;
    r.table('users').get(100).update({
        avatar: avatarImage
    })
});
```

- [x] ### do

**Syntax:** `any.do(function) → any` / `r.do([args]*, function) → any`

**Description:** Call an anonymous function using return values from other ReQL commands or queries as arguments.

**Example:**
```js
r.table('players').get('f19b5f16-ef14-468f-bd48-e194761df255').do(
    function (player) {
        return player('gross_score').sub(player('course_handicap'));
    }
).run(conn, callback);
```

- [x] ### branch

**Syntax:** `r.branch(test, true_action[, test2, test2_action, ...], false_action) → any`

**Description:** Perform a branching conditional equivalent to `if-then-else`.

**Example:**
```js
var x = 10;
r.branch(r.expr(x).gt(5), 'big', 'small').run(conn, callback);
// Result: "big"
```

- [x] ### forEach

**Syntax:** `sequence.forEach(write_function) → object`

**Description:** Loop over a sequence, evaluating the given write query for each element.

**Example:**
```js
r.table('marvel').forEach(function(hero) {
    return r.table('villains').get(hero('villainDefeated')).delete()
}).run(conn, callback)
```

- [x] ### range

**Syntax:** `r.range() → stream` / `r.range([startValue, ]endValue) → stream`

**Description:** Generate a stream of sequential integers in a specified range.

**Example:**
```js
r.range(4).run(conn, callback)
// Result: [0, 1, 2, 3]
```

- [x] ### error

**Syntax:** `r.error(message) → error`

**Description:** Throw a runtime error.

**Example:**
```js
r.table('marvel').get('IronMan').do(function(ironman) {
    return r.branch(ironman('victories').lt(ironman('battles')),
        r.error('impossible code path'),
        ironman)
}).run(conn, callback)
```

- [x] ### default

**Syntax:** `value.default(default_value | function) → any` / `sequence.default(default_value | function) → any`

**Description:** Provide a default value in case of non-existence errors.

**Example:**
```js
r.table("posts").map(function (post) {
    return {
        title: post("title"),
        author: post("author").default("Anonymous")
    }
}).run(conn, callback);
```

- [x] ### expr

**Syntax:** `r.expr(value) → value`

**Description:** Construct a ReQL JSON object from a native object.

**Example:**
```js
r.expr({a:'b'}).merge({b:[1,2,3]}).run(conn, callback)
```

- [ ] ### js

**Syntax:** `r.js(jsString[, {timeout: <number>}]) → value`

**Description:** Create a javascript expression.

**Example:**
```js
r.table('marvel').filter(
    r.js('(function (row) { return row.magazines.length > 5; })')
).run(conn, callback)
```

- [x] ### coerceTo

**Syntax:** `sequence.coerceTo('array') → array` / `value.coerceTo('string') → string` / `string.coerceTo('number') → number` / `array.coerceTo('object') → object`

**Description:** Convert a value of one type into another.

**Example:**
```js
r.expr([['name', 'Ironman'], ['victories', 2000]]).coerceTo('object').run(conn, callback)
```

- [x] ### typeOf

**Syntax:** `any.typeOf() → string`

**Description:** Gets the type of a ReQL query's return value.

**Example:**
```js
r.expr("foo").typeOf().run(conn, callback);
// Result: "STRING"
```

- [x] ### info

**Syntax:** `any.info() → object` / `r.info(any) → object`

**Description:** Get information about a ReQL value.

**Example:**
```js
r.table('marvel').info().run(conn, callback)
```

- [x] ### json

**Syntax:** `r.json(json_string) → value`

**Description:** Parse a JSON string on the server.

**Example:**
```js
r.json("[1,2,3]").run(conn, callback)
```

- [x] ### toJsonString / toJSON

**Syntax:** `value.toJsonString() → string` / `value.toJSON() → string`

**Description:** Convert a ReQL value or object to a JSON string.

**Example:**
```js
r.table('hero').get(1).toJSON()
```

- [ ] ### http

**Syntax:** `r.http(url[, options]) → value` / `r.http(url[, options]) → stream`

**Description:** Retrieve data from the specified URL over HTTP.

**Example:**
```js
r.table('posts').insert(r.http('http://httpbin.org/get')).run(conn, callback)
```

- [x] ### uuid

**Syntax:** `r.uuid([string]) → string`

**Description:** Return a UUID (universally unique identifier).

**Example:**
```js
r.uuid().run(conn, callback)
// Result: "27961a0e-f4e8-4eb3-bf95-c5203e1d87b9"
```

---

## Geospatial Commands

- [x] ### circle

**Syntax:** `r.circle([longitude, latitude], radius[, {numVertices: 32, geoSystem: 'WGS84', unit: 'm', fill: true}]) → geometry`

**Description:** Construct a circular line or polygon by approximating a circle of specified radius around a center point.

**Example:**
```js
r.table('geo').insert({
    id: 300,
    name: 'Hayes Valley',
    neighborhood: r.circle([-122.423246,37.779388], 1000)
}).run(conn, callback);
```

- [x] ### distance

**Syntax:** `geometry.distance(geometry[, {geoSystem: 'WGS84', unit: 'm'}]) → number`

**Description:** Compute the distance between a point and another geometry object.

**Example:**
```js
var point1 = r.point(-122.423246,37.779388);
var point2 = r.point(-117.220406,32.719464);
r.distance(point1, point2, {unit: 'km'}).run(conn, callback);
// Result: 734.125
```

- [x] ### fill

**Syntax:** `line.fill() → polygon`

**Description:** Convert a Line object into a Polygon object, closing the polygon by connecting the last point to the first if needed.

**Example:**
```js
r.table('geo').get(201).update({
    rectangle: r.row('rectangle').fill()
}, {nonAtomic: true}).run(conn, callback);
```

- [x] ### geojson

**Syntax:** `r.geojson(geojson) → geometry`

**Description:** Convert a GeoJSON object to a ReQL geometry object.

**Example:**
```js
var geoJson = {
    'type': 'Point',
    'coordinates': [ -122.423246, 37.779388 ]
};
r.table('geo').insert({
    id: 'sfo',
    name: 'San Francisco',
    location: r.geojson(geoJson)
}).run(conn, callback);
```

- [x] ### toGeojson

**Syntax:** `geometry.toGeojson() → object`

**Description:** Convert a ReQL geometry object to a GeoJSON object.

**Example:**
```js
r.table('geo').get('sfo')('location').toGeojson().run(conn, callback);
// Result: { 'type': 'Point', 'coordinates': [ -122.423246, 37.779388 ] }
```

- [x] ### getIntersecting

**Syntax:** `table.getIntersecting(geometry, {index: 'indexname'}) → selection<stream>`

**Description:** Get all documents where the given geometry object intersects the geometry object of the requested geospatial index.

**Example:**
```js
var circle1 = r.circle([-117.220406,32.719464], 10, {unit: 'mi'});
r.table('parks').getIntersecting(circle1, {index: 'area'}).run(conn, callback);
```

- [x] ### getNearest

**Syntax:** `table.getNearest(point, {index: 'indexname'[, maxResults: 100, maxDist: 100000, unit: 'm', geoSystem: 'WGS84']}) → array`

**Description:** Return a list of documents closest to a specified point based on a geospatial index, sorted by increasing distance.

**Example:**
```js
var secretBase = r.point(-122.422876,37.777128);
r.table('hideouts').getNearest(secretBase,
    {index: 'location', maxResults: 25}
).run(conn, callback);
```

- [x] ### includes

**Syntax:** `sequence.includes(geometry) → sequence` / `geometry.includes(geometry) → bool`

**Description:** Tests whether a geometry object is completely contained within another.

**Example:**
```js
var point1 = r.point(-117.220406,32.719464);
var point2 = r.point(-117.206201,32.725186);
r.circle(point1, 2000).includes(point2).run(conn, callback);
// Result: true
```

- [x] ### intersects

**Syntax:** `sequence.intersects(geometry) → sequence` / `geometry.intersects(geometry) → bool`

**Description:** Tests whether two geometry objects intersect with one another.

**Example:**
```js
var point1 = r.point(-117.220406,32.719464);
var point2 = r.point(-117.206201,32.725186);
r.circle(point1, 2000).intersects(point2).run(conn, callback);
// Result: true
```

- [x] ### line

**Syntax:** `r.line([lon1, lat1], [lon2, lat2], ...) → line`

**Description:** Construct a geometry object of type Line by providing two or more coordinate pairs or Point objects.

**Example:**
```js
r.table('geo').insert({
    id: 101,
    route: r.line([-122.423246,37.779388], [-121.886420,37.329898])
}).run(conn, callback);
```

- [x] ### point

**Syntax:** `r.point(longitude, latitude) → point`

**Description:** Construct a geometry object of type Point specified by longitude and latitude coordinates.

**Example:**
```js
r.table('geo').insert({
    id: 1,
    name: 'San Francisco',
    location: r.point(-122.423246,37.779388)
}).run(conn, callback);
```

- [x] ### polygon

**Syntax:** `r.polygon([lon1, lat1], [lon2, lat2], [lon3, lat3], ...) → polygon`

**Description:** Construct a geometry object of type Polygon using coordinate arrays or Point objects representing the vertices.

**Example:**
```js
r.table('geo').insert({
    id: 101,
    rectangle: r.polygon(
        [-122.423246,37.779388],
        [-122.423246,37.329898],
        [-121.886420,37.329898],
        [-121.886420,37.779388]
    )
}).run(conn, callback);
```

- [x] ### polygonSub

**Syntax:** `polygon1.polygonSub(polygon2) → polygon`

**Description:** Use polygon2 to "punch out" a hole in polygon1, where the inner polygon must be completely contained within the outer one.

**Example:**
```js
var outerPolygon = r.polygon(
    [-122.4,37.7],
    [-122.4,37.3],
    [-121.8,37.3],
    [-121.8,37.7]
);
var innerPolygon = r.polygon(
    [-122.3,37.4],
    [-122.3,37.6],
    [-122.0,37.6],
    [-122.0,37.4]
);
outerPolygon.polygonSub(innerPolygon).run(conn, callback);
```

---

## Administration

- [x] ### grant

**Syntax:** `r.grant("username", {permission: bool, ...}) → object` / `db.grant("username", {permission: bool, ...}) → object` / `table.grant("username", {permission: bool, ...}) → object`

**Description:** Grant or deny access permissions for a user account, globally, on a database, or on a specific table.

**Example:**
```js
r.grant("bob", {read: true, write: false, config: false}).run(conn, callback)
```

- [x] ### config

**Syntax:** `table.config() → selection<object>` / `database.config() → selection<object>`

**Description:** Query (read and/or update) the configurations for individual tables or databases.

**Example:**
```js
r.table('users').config().run(conn, callback)
```

- [x] ### rebalance

**Syntax:** `table.rebalance() → object` / `database.rebalance() → object`

**Description:** Rebalance the shards of a table, distributing data evenly across replicas.

**Example:**
```js
r.table('superheroes').rebalance().run(conn, callback)
```

- [x] ### reconfigure

**Syntax:** `table.reconfigure({shards: <s>, replicas: <r>[, primaryReplicaTag: <tag>, dryRun: false]}) → object`

**Description:** Reconfigure a table's sharding and replication.

**Example:**
```js
r.table('superheroes').reconfigure({shards: 2, replicas: 1}).run(conn, callback)
```

- [x] ### status

**Syntax:** `table.status() → selection<object>`

**Description:** Return the status of a table, including information about storage engine and replica/shard status.

**Example:**
```js
r.table('superheroes').status().run(conn, callback)
```

- [x] ### wait

**Syntax:** `table.wait([{waitFor: 'ready_for_writes', timeout: <sec>}]) → object` / `database.wait([{waitFor: 'ready_for_writes', timeout: <sec>}]) → object`

**Description:** Wait for a table or all the tables in a database to be ready.

**Example:**
```js
r.table('superheroes').wait({waitFor: 'all_replicas_ready'}).run(conn, callback)
```
