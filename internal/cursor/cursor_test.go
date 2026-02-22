package cursor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"r-cli/internal/proto"
	"r-cli/internal/response"
)

func rawMsg(s string) json.RawMessage { return json.RawMessage(s) }

func TestAtomCursor_SingleValue(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: []json.RawMessage{rawMsg(`"hello"`)},
	}
	c := NewAtom(resp)

	item, err := c.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(item) != `"hello"` {
		t.Fatalf("got %s, want %q", item, "hello")
	}

	// second call must return EOF
	_, err = c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestAtomCursor_All(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: []json.RawMessage{rawMsg(`42`)},
	}
	c := NewAtom(resp)

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 element, got %d", len(all))
	}
	if string(all[0]) != `42` {
		t.Fatalf("got %s, want 42", all[0])
	}
}

func TestAtomCursor_EOF_Immediately_When_Empty(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: nil,
	}
	c := NewAtom(resp)

	item, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF for empty atom, got err=%v item=%v", err, item)
	}
}

func TestSeqCursor_IterateAll(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type: proto.ResponseSuccessSequence,
		Results: []json.RawMessage{
			rawMsg(`1`),
			rawMsg(`2`),
			rawMsg(`3`),
		},
	}
	c := NewSequence(resp)

	for i := 1; i <= 3; i++ {
		item, err := c.Next()
		if err != nil {
			t.Fatalf("step %d: unexpected error: %v", i, err)
		}
		want := string(rawMsg(string(rune('0' + i))))
		if string(item) != want {
			t.Fatalf("step %d: got %s, want %s", i, item, want)
		}
	}

	_, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF after exhaustion, got %v", err)
	}
}

func TestSeqCursor_All(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type: proto.ResponseSuccessSequence,
		Results: []json.RawMessage{
			rawMsg(`"a"`),
			rawMsg(`"b"`),
		},
	}
	c := NewSequence(resp)

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(all))
	}
	if string(all[0]) != `"a"` || string(all[1]) != `"b"` {
		t.Fatalf("unexpected values: %v", all)
	}
}

func TestSeqCursor_Empty(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessSequence,
		Results: nil,
	}
	c := NewSequence(resp)

	_, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected io.EOF for empty sequence, got %v", err)
	}

	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error from All(): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty slice, got %v", all)
	}
}

// --- Task 5: streaming cursor tests ---

func TestStreamCursor_PartialThenSequence(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response, 1)

	var continueSent bool
	send := func(qt proto.QueryType) error {
		if qt == proto.QueryContinue {
			continueSent = true
			ch <- &response.Response{
				Type:    proto.ResponseSuccessSequence,
				Results: []json.RawMessage{rawMsg(`3`), rawMsg(`4`)},
			}
		}
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`), rawMsg(`2`)},
	}
	c := NewStream(context.Background(), initial, ch, send)

	for i := 1; i <= 4; i++ {
		item, err := c.Next()
		if err != nil {
			t.Fatalf("item %d: unexpected error: %v", i, err)
		}
		want := string(rune('0' + i))
		if string(item) != want {
			t.Fatalf("item %d: got %s, want %s", i, item, want)
		}
	}

	_, err := c.Next()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after last item, got %v", err)
	}
	if !continueSent {
		t.Fatal("expected CONTINUE to be sent")
	}
}

func TestStreamCursor_Close_SendsStop(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response)

	var mu sync.Mutex
	var sent []proto.QueryType
	send := func(qt proto.QueryType) error {
		mu.Lock()
		sent = append(sent, qt)
		mu.Unlock()
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`)},
	}
	c := NewStream(context.Background(), initial, ch, send)

	item, err := c.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if string(item) != `1` {
		t.Fatalf("got %s, want 1", item)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 1 || sent[0] != proto.QueryStop {
		t.Fatalf("expected [STOP], got %v", sent)
	}
}

func TestStreamCursor_ContextCancel_SendsStop(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response) // never receives

	waiting := make(chan struct{})
	var mu sync.Mutex
	var sent []proto.QueryType
	send := func(qt proto.QueryType) error {
		mu.Lock()
		sent = append(sent, qt)
		mu.Unlock()
		if qt == proto.QueryContinue {
			close(waiting) // signal: about to block in waitForResponse
		}
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: nil,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := NewStream(ctx, initial, ch, send)

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Next()
		errCh <- err
	}()

	// wait until CONTINUE was sent, then cancel
	<-waiting
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Next() to return")
	}

	mu.Lock()
	defer mu.Unlock()
	stopCount := 0
	for _, qt := range sent {
		if qt == proto.QueryStop {
			stopCount++
		}
	}
	if stopCount != 1 {
		t.Fatalf("expected exactly 1 STOP, got %d in %v", stopCount, sent)
	}
}

// --- Task 6: changefeed cursor tests ---

func TestChangefeedCursor_InfiniteStream(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response, 2)

	ch <- &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`2`), rawMsg(`3`)},
	}
	ch <- &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`4`)},
	}

	var continueMu sync.Mutex
	continueCount := 0
	send := func(qt proto.QueryType) error {
		if qt == proto.QueryContinue {
			continueMu.Lock()
			continueCount++
			continueMu.Unlock()
		}
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`)},
	}
	c := NewChangefeed(context.Background(), initial, ch, send)

	for i := 1; i <= 4; i++ {
		item, err := c.Next()
		if err != nil {
			t.Fatalf("item %d: unexpected error: %v", i, err)
		}
		want := string(rune('0' + i))
		if string(item) != want {
			t.Fatalf("item %d: got %s, want %s", i, item, want)
		}
	}

	continueMu.Lock()
	count := continueCount
	continueMu.Unlock()
	if count != 2 {
		t.Fatalf("expected exactly 2 CONTINUE sends, got %d", count)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestChangefeedCursor_Close_SendsStop(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response) // never receives

	var mu sync.Mutex
	var sent []proto.QueryType
	send := func(qt proto.QueryType) error {
		mu.Lock()
		sent = append(sent, qt)
		mu.Unlock()
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`)},
	}
	c := NewChangefeed(context.Background(), initial, ch, send)

	item, err := c.Next()
	if err != nil {
		t.Fatalf("Next() error: %v", err)
	}
	if string(item) != `1` {
		t.Fatalf("got %s, want 1", item)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(sent) != 1 || sent[0] != proto.QueryStop {
		t.Fatalf("expected [STOP], got %v", sent)
	}
}

func TestChangefeedCursor_ConnectionDrop(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response)

	send := func(qt proto.QueryType) error { return nil }

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: nil,
	}
	c := NewChangefeed(context.Background(), initial, ch, send)

	errCh := make(chan error, 1)
	go func() {
		_, err := c.Next()
		errCh <- err
	}()

	// simulate connection drop
	close(ch)

	select {
	case err := <-errCh:
		if err == nil || errors.Is(err, io.EOF) {
			t.Fatalf("expected connection error, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Next() on connection drop")
	}
}

func TestAtomCursor_All_AfterNext(t *testing.T) {
	t.Parallel()
	resp := &response.Response{
		Type:    proto.ResponseSuccessAtom,
		Results: []json.RawMessage{rawMsg(`99`)},
	}
	c := NewAtom(resp)

	// consume with Next
	_, err := c.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All() must return nothing after Next consumed the item
	all, err := c.All()
	if err != nil {
		t.Fatalf("unexpected error from All(): %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty All() after Next consumed item, got %v", all)
	}
}

func TestChangefeedCursor_All_ReturnsError(t *testing.T) {
	t.Parallel()
	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: nil,
	}
	c := NewChangefeed(context.Background(), initial, make(chan *response.Response), func(proto.QueryType) error { return nil })
	_, err := c.All()
	if err == nil {
		t.Fatal("expected error from changefeed All(), got nil")
	}
}

func TestStreamCursor_ServerError(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response, 1)
	send := func(qt proto.QueryType) error {
		if qt == proto.QueryContinue {
			ch <- &response.Response{
				Type:    proto.ResponseRuntimeError,
				Results: []json.RawMessage{rawMsg(`"query failed"`)},
			}
		}
		return nil
	}
	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`)},
	}
	c := NewStream(context.Background(), initial, ch, send)

	_, err := c.Next() // consume initial item
	if err != nil {
		t.Fatalf("unexpected error on first item: %v", err)
	}
	_, err = c.Next() // triggers fetchBatch -> server returns error
	if err == nil {
		t.Fatal("expected error from server, got nil")
	}
	var re *response.ReqlRuntimeError
	if !errors.As(err, &re) {
		t.Fatalf("expected *ReqlRuntimeError, got %T: %v", err, err)
	}
}

func TestStreamCursor_UnexpectedResponseType(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response, 1)
	send := func(qt proto.QueryType) error {
		if qt == proto.QueryContinue {
			ch <- &response.Response{Type: proto.ResponseWaitComplete}
		}
		return nil
	}
	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`1`)},
	}
	c := NewStream(context.Background(), initial, ch, send)

	_, err := c.Next() // consume initial item
	if err != nil {
		t.Fatalf("unexpected error on first item: %v", err)
	}
	_, err = c.Next() // triggers fetchBatch -> unexpected response type
	if err == nil {
		t.Fatal("expected error for unexpected response type, got nil")
	}
}

func TestStreamCursor_ConcurrentNext(t *testing.T) {
	t.Parallel()
	ch := make(chan *response.Response, 1)

	send := func(qt proto.QueryType) error {
		if qt == proto.QueryContinue {
			ch <- &response.Response{
				Type: proto.ResponseSuccessSequence,
				Results: []json.RawMessage{
					rawMsg(`"b"`), rawMsg(`"c"`), rawMsg(`"d"`), rawMsg(`"e"`),
				},
			}
		}
		return nil
	}

	initial := &response.Response{
		Type:    proto.ResponseSuccessPartial,
		Results: []json.RawMessage{rawMsg(`"a"`)},
	}
	c := NewStream(context.Background(), initial, ch, send)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []string

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := c.Next()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			mu.Lock()
			results = append(results, string(item))
			mu.Unlock()
		}()
	}

	wg.Wait()
	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d: %v", len(results), results)
	}
	// verify each goroutine got a distinct value
	seen := make(map[string]bool, 5)
	for _, r := range results {
		if seen[r] {
			t.Fatalf("duplicate result %q in concurrent Next()", r)
		}
		seen[r] = true
	}
}
