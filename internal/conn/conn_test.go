package conn

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"r-cli/internal/wire"
)

func TestNextTokenMonotonic(t *testing.T) {
	t.Parallel()
	c := &Conn{}
	prev := c.NextToken()
	for range 100 {
		next := c.NextToken()
		if next <= prev {
			t.Fatalf("token %d is not greater than previous %d", next, prev)
		}
		prev = next
	}
}

func TestNextTokenConcurrentNoDuplicates(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	const tokensEach = 100

	c := &Conn{}
	seen := make(map[uint64]struct{}, goroutines*tokensEach)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			tokens := make([]uint64, tokensEach)
			for i := range tokensEach {
				tokens[i] = c.NextToken()
			}
			mu.Lock()
			for _, tok := range tokens {
				if _, dup := seen[tok]; dup {
					t.Errorf("duplicate token: %d", tok)
				}
				seen[tok] = struct{}{}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestConfigStringNoPassword(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Host:     "localhost",
		Port:     28015,
		User:     "admin",
		Password: "supersecret",
	}
	s := cfg.String()
	if strings.Contains(s, "supersecret") {
		t.Fatalf("Config.String() leaks password: %q", s)
	}
}

// setupConn creates a client *Conn over net.Pipe, performing the V1_0 handshake.
// Returns the client Conn (ready for wire queries) and the raw server-side net.Conn.
func setupConn(t *testing.T) (clientConn *Conn, serverNC net.Conn) {
	t.Helper()
	client, srvNC := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = srvNC.Close()
	})
	const user, pass = "testuser", "testpass"
	go func() {
		srv := &mockSCRAMServer{password: pass}
		srv.serve(t, srvNC)
	}()
	if err := Handshake(client, user, pass); err != nil {
		t.Fatalf("setupConn: Handshake: %v", err)
	}
	c := newConn(client)
	t.Cleanup(func() { _ = c.Close() })
	return c, srvNC
}

func TestConnBasicSendReceive(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok := c.NextToken()
	query := []byte(`[1,[39,[]],{}]`)
	resp := []byte(`{"t":1,"r":[42]}`)

	go func() {
		if _, _, err := wire.ReadResponse(server); err != nil {
			t.Errorf("server read: %v", err)
			return
		}
		if err := wire.WriteQuery(server, tok, resp); err != nil {
			t.Errorf("server write: %v", err)
		}
	}()

	got, err := c.Send(context.Background(), tok, query)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !bytes.Equal(got, resp) {
		t.Errorf("got %q, want %q", got, resp)
	}
}

func TestConnConcurrentQueries(t *testing.T) { //nolint:cyclop
	t.Parallel()
	c, server := setupConn(t)

	const n = 10
	type pair struct {
		token   uint64
		payload []byte
	}
	pairs := make([]pair, n)
	for i := range n {
		pairs[i] = pair{
			token:   c.NextToken(),
			payload: []byte(fmt.Sprintf(`{"i":%d}`, i)),
		}
	}

	go func() {
		for range n {
			if _, _, err := wire.ReadResponse(server); err != nil {
				t.Errorf("server read: %v", err)
				return
			}
		}
		for _, p := range pairs {
			if err := wire.WriteQuery(server, p.token, p.payload); err != nil {
				t.Errorf("server write tok=%d: %v", p.token, err)
				return
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(n)
	for _, p := range pairs {
		go func() {
			defer wg.Done()
			got, err := c.Send(context.Background(), p.token, p.payload)
			if err != nil {
				t.Errorf("Send tok=%d: %v", p.token, err)
				return
			}
			if !bytes.Equal(got, p.payload) {
				t.Errorf("tok=%d: got %q, want %q", p.token, got, p.payload)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent queries timed out - server goroutine may have failed")
	}
}

func TestConnOutOfOrderResponses(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok1 := c.NextToken()
	tok2 := c.NextToken()
	resp1 := []byte(`"r1"`)
	resp2 := []byte(`"r2"`)

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		for range 2 {
			if _, _, err := wire.ReadResponse(server); err != nil {
				t.Errorf("server read: %v", err)
				return
			}
		}
		// respond to tok2 first, then tok1
		if err := wire.WriteQuery(server, tok2, resp2); err != nil {
			t.Errorf("server write tok2: %v", err)
			return
		}
		if err := wire.WriteQuery(server, tok1, resp1); err != nil {
			t.Errorf("server write tok1: %v", err)
		}
	}()

	got1C := make(chan []byte, 1)
	got2C := make(chan []byte, 1)
	go func() {
		got, err := c.Send(context.Background(), tok1, []byte(`"q1"`))
		if err != nil {
			t.Errorf("Send tok1: %v", err)
		}
		got1C <- got
	}()
	go func() {
		got, err := c.Send(context.Background(), tok2, []byte(`"q2"`))
		if err != nil {
			t.Errorf("Send tok2: %v", err)
		}
		got2C <- got
	}()

	got1 := <-got1C
	got2 := <-got2C
	<-serverDone

	if !bytes.Equal(got1, resp1) {
		t.Errorf("tok1: got %q, want %q", got1, resp1)
	}
	if !bytes.Equal(got2, resp2) {
		t.Errorf("tok2: got %q, want %q", got2, resp2)
	}
}

func TestConnSlowConsumerNoBlock(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok2 := c.NextToken()
	resp2 := []byte(`"fast"`)

	// pre-fill a waiter for a fake token to simulate a slow/full consumer
	const fakeTok = uint64(99999)
	fullCh := make(chan result, 1)
	fullCh <- result{payload: []byte("occupied")}
	c.mu.Lock()
	c.waiters[fakeTok] = fullCh
	c.mu.Unlock()

	got2 := make(chan []byte, 1)
	go func() {
		got, err := c.Send(context.Background(), tok2, []byte(`"q2"`))
		if err != nil {
			t.Errorf("Send tok2: %v", err)
		}
		got2 <- got
	}()

	go func() {
		_, _, _ = wire.ReadResponse(server)                 // tok2 query
		_ = wire.WriteQuery(server, fakeTok, []byte(`"x"`)) // to full channel: discarded
		_ = wire.WriteQuery(server, tok2, resp2)
	}()

	select {
	case got := <-got2:
		if !bytes.Equal(got, resp2) {
			t.Errorf("tok2: got %q, want %q", got, resp2)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("tok2 blocked - readLoop did not skip full consumer channel")
	}
}

func TestConnLateResponseDiscarded(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok := c.NextToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// server: read query, then read STOP, then send a late response
	querySeen := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		if _, _, err := wire.ReadResponse(server); err != nil {
			return
		}
		close(querySeen)
		_, _, _ = wire.ReadResponse(server)                // STOP
		_ = wire.WriteQuery(server, tok, []byte(`"late"`)) // late response
	}()

	sendDone := make(chan error, 1)
	go func() {
		_, err := c.Send(ctx, tok, []byte(`"q"`))
		sendDone <- err
	}()

	<-querySeen
	cancel()

	select {
	case err := <-sendDone:
		if err == nil {
			t.Fatal("expected context error, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Send did not return after cancel")
	}
	<-serverDone // no panic = late response was discarded
}

func TestConnCloseUnblocksWaiters(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok := c.NextToken()
	serverGotQuery := make(chan struct{})
	go func() {
		_, _, _ = wire.ReadResponse(server)
		close(serverGotQuery)
		// do not send a response - let Close() unblock Send
	}()

	sendErr := make(chan error, 1)
	go func() {
		_, err := c.Send(context.Background(), tok, []byte(`"q"`))
		sendErr <- err
	}()

	<-serverGotQuery // query was received, so Send() is now in select
	_ = c.Close()

	select {
	case err := <-sendErr:
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Send did not unblock after Close")
	}
}

func TestConnSendAfterClose(t *testing.T) {
	t.Parallel()
	c, _ := setupConn(t)

	if err := c.Close(); err != nil {
		t.Logf("Close: %v", err)
	}

	_, err := c.Send(context.Background(), c.NextToken(), []byte(`"q"`))
	if err == nil {
		t.Fatal("expected error after Close, got nil")
	}
	if !errors.Is(err, ErrClosed) {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}

func TestConnContextCancellationSendsStop(t *testing.T) { //nolint:cyclop
	t.Parallel()
	c, server := setupConn(t)

	tok := c.NextToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	querySeen := make(chan struct{})
	stopSeen := make(chan bool, 1)
	go func() {
		if _, _, err := wire.ReadResponse(server); err != nil {
			stopSeen <- false
			return
		}
		close(querySeen)
		stopTok, stopPld, err := wire.ReadResponse(server)
		if err != nil {
			stopSeen <- false
			return
		}
		stopSeen <- stopTok == tok && bytes.Equal(stopPld, stopPayload)
	}()

	sendDone := make(chan error, 1)
	go func() {
		_, err := c.Send(ctx, tok, []byte(`"q"`))
		sendDone <- err
	}()

	<-querySeen
	cancel()

	select {
	case err := <-sendDone:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Send did not return after cancel")
	}

	select {
	case ok := <-stopSeen:
		if !ok {
			t.Error("STOP not received correctly by server")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive STOP")
	}

	c.mu.Lock()
	_, exists := c.waiters[tok]
	c.mu.Unlock()
	if exists {
		t.Error("waiter not cleaned up after context cancellation")
	}
}

func TestDialContextCancellationNoGoroutineLeak(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	// accept connections but never send handshake response
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = conn.Close() }()
				buf := make([]byte, 4096)
				for {
					if _, err := conn.Read(buf); err != nil {
						return
					}
				}
			}()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{User: "admin", Password: "pass"}

	dialDone := make(chan error, 1)
	go func() {
		_, err := Dial(ctx, ln.Addr().String(), cfg, nil)
		dialDone <- err
	}()

	time.Sleep(10 * time.Millisecond) // let Dial block on handshake
	cancel()

	select {
	case err := <-dialDone:
		if err == nil {
			t.Fatal("expected error after context cancellation")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Dial did not return after cancel - goroutine leaked")
	}
}

func TestConnSendWriteError(t *testing.T) {
	t.Parallel()
	client, _ := net.Pipe()
	_ = client.Close() // writes will fail immediately

	c := &Conn{
		nc:      client,
		waiters: make(map[uint64]chan result),
		done:    make(chan struct{}),
	}
	// readLoop not started; we only exercise the write-failure path

	tok := c.token.Add(1)
	_, err := c.Send(context.Background(), tok, []byte(`"q"`))
	if err == nil {
		t.Fatal("expected write error, got nil")
	}

	// waiter must be cleaned up after write failure
	c.mu.Lock()
	_, exists := c.waiters[tok]
	c.mu.Unlock()
	if exists {
		t.Error("waiter not cleaned up after write error")
	}
}

// testTLSServer generates a self-signed cert and starts a TLS listener on a
// random port. Returns the address and the cert PEM for building CA pools.
func testTLSServer(t *testing.T) (addr string, certPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("testTLSServer: generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("testTLSServer: create cert: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("testTLSServer: marshal key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("testTLSServer: key pair: %v", err)
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{tlsCert}})
	if err != nil {
		t.Fatalf("testTLSServer: listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				_ = c.(*tls.Conn).Handshake() //nolint:forcetypeassert
			}()
		}
	}()

	return ln.Addr().String(), certPEM
}

func TestDialTLSValidCACert(t *testing.T) {
	t.Parallel()
	addr, certPEM := testTLSServer(t)

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		t.Fatal("AppendCertsFromPEM: no valid certificate found")
	}

	nc, err := DialTLS(context.Background(), addr, &tls.Config{RootCAs: pool})
	if err != nil {
		t.Fatalf("DialTLS: %v", err)
	}
	_ = nc.Close()
}

func TestDialTLSWrongCACert(t *testing.T) {
	t.Parallel()
	addr, _ := testTLSServer(t)

	// generate an unrelated CA cert that won't verify the server's certificate
	wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(99),
		Subject:               pkix.Name{CommonName: "wrong-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &wrongKey.PublicKey, wrongKey)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	wrongCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	wrongPool := x509.NewCertPool()
	if !wrongPool.AppendCertsFromPEM(wrongCertPEM) {
		t.Fatal("AppendCertsFromPEM: no valid certificate found")
	}

	_, err = DialTLS(context.Background(), addr, &tls.Config{RootCAs: wrongPool})
	if err == nil {
		t.Fatal("expected TLS verification error, got nil")
	}
}

func TestDialTLSInsecureSkipVerify(t *testing.T) {
	t.Parallel()
	addr, _ := testTLSServer(t)

	nc, err := DialTLS(context.Background(), addr, &tls.Config{InsecureSkipVerify: true}) //nolint:gosec
	if err != nil {
		t.Fatalf("DialTLS: %v", err)
	}
	_ = nc.Close()
}

func TestConnStopWithLatePartialNoDeadlock(t *testing.T) {
	t.Parallel()
	c, server := setupConn(t)

	tok := c.NextToken()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// server: read query, then STOP, then send a late SUCCESS_PARTIAL
	querySeen := make(chan struct{})
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		if _, _, err := wire.ReadResponse(server); err != nil { // query
			return
		}
		close(querySeen)
		if _, _, err := wire.ReadResponse(server); err != nil { // STOP
			return
		}
		// late response after STOP - readLoop must discard without blocking
		_ = wire.WriteQuery(server, tok, []byte(`{"t":3,"r":[1]}`))
	}()

	sendDone := make(chan error, 1)
	go func() {
		_, err := c.Send(ctx, tok, []byte(`"q"`))
		sendDone <- err
	}()

	<-querySeen
	cancel()

	select {
	case <-sendDone:
	case <-time.After(3 * time.Second):
		t.Fatal("Send deadlocked after context cancel")
	}

	select {
	case <-serverDone:
	case <-time.After(3 * time.Second):
		t.Fatal("server deadlocked - late partial caused blocking")
	}
}
