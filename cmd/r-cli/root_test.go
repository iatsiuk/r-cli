package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/response"
)

func TestRootHostDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	host, err := cmd.PersistentFlags().GetString("host")
	if err != nil {
		t.Fatal(err)
	}
	if host != "localhost" {
		t.Errorf("got %q, want %q", host, "localhost")
	}
}

func TestRootPortDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	port, err := cmd.PersistentFlags().GetInt("port")
	if err != nil {
		t.Fatal(err)
	}
	if port != 28015 {
		t.Errorf("got %d, want %d", port, 28015)
	}
}

func TestRootDBDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	db, err := cmd.PersistentFlags().GetString("db")
	if err != nil {
		t.Fatal(err)
	}
	if db != "" {
		t.Errorf("got %q, want empty", db)
	}
}

func TestRootUserDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	user, err := cmd.PersistentFlags().GetString("user")
	if err != nil {
		t.Fatal(err)
	}
	if user != "admin" {
		t.Errorf("got %q, want %q", user, "admin")
	}
}

func TestRootPasswordDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	password, err := cmd.PersistentFlags().GetString("password")
	if err != nil {
		t.Fatal(err)
	}
	if password != "" {
		t.Errorf("got %q, want empty", password)
	}
}

func TestRootTimeoutDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	timeout, err := cmd.PersistentFlags().GetDuration("timeout")
	if err != nil {
		t.Fatal(err)
	}
	if timeout != 30*time.Second {
		t.Errorf("got %v, want %v", timeout, 30*time.Second)
	}
}

func TestRootFormatDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	format, err := cmd.PersistentFlags().GetString("format")
	if err != nil {
		t.Fatal(err)
	}
	if format != "" {
		t.Errorf("got %q, want empty (auto-detect)", format)
	}
}

func TestRootHostShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-H", "myhost"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("host")
	if got != "myhost" {
		t.Errorf("got %q, want %q", got, "myhost")
	}
}

func TestRootPortShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-P", "19015"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetInt("port")
	if got != 19015 {
		t.Errorf("got %d, want %d", got, 19015)
	}
}

func TestRootDBShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-d", "mydb"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("db")
	if got != "mydb" {
		t.Errorf("got %q, want %q", got, "mydb")
	}
}

func TestRootUserShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-u", "myuser"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("user")
	if got != "myuser" {
		t.Errorf("got %q, want %q", got, "myuser")
	}
}

func TestRootPasswordShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-p", "secret"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("password")
	if got != "secret" {
		t.Errorf("got %q, want %q", got, "secret")
	}
}

func TestRootTimeoutShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-t", "10s"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetDuration("timeout")
	if got != 10*time.Second {
		t.Errorf("got %v, want %v", got, 10*time.Second)
	}
}

func TestRootFormatShorthand(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"-f", "jsonl"}); err != nil {
		t.Fatal(err)
	}
	got, _ := cmd.PersistentFlags().GetString("format")
	if got != "jsonl" {
		t.Errorf("got %q, want %q", got, "jsonl")
	}
}

func TestRootFormatValues(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"json", "jsonl", "raw", "table"} {
		cmd := newRootCmd()
		if err := cmd.ParseFlags([]string{"--format", v}); err != nil {
			t.Fatalf("format %q: %v", v, err)
		}
		got, _ := cmd.PersistentFlags().GetString("format")
		if got != v {
			t.Errorf("format %q: got %q", v, got)
		}
	}
}

func TestPasswordFileFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := dir + "/password.txt"
	if err := os.WriteFile(path, []byte("mysecret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &rootConfig{passwordFile: path}
	if err := cfg.resolvePassword(); err != nil {
		t.Fatalf("resolvePassword: %v", err)
	}
	if cfg.password != "mysecret" {
		t.Errorf("password: got %q, want %q", cfg.password, "mysecret")
	}
}

func TestPasswordFileNotFound(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{passwordFile: "/nonexistent/path/password.txt"}
	if err := cfg.resolvePassword(); err == nil {
		t.Error("expected error for missing password file, got nil")
	}
}

func TestPasswordFileStripsNewline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"unix newline", "pass\n", "pass"},
		{"windows newline", "pass\r\n", "pass"},
		{"no newline", "pass", "pass"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := dir + "/password.txt"
			if err := os.WriteFile(path, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg := &rootConfig{passwordFile: path}
			if err := cfg.resolvePassword(); err != nil {
				t.Fatal(err)
			}
			if cfg.password != tc.want {
				t.Errorf("got %q, want %q", cfg.password, tc.want)
			}
		})
	}
}

func TestEnvVarHost(t *testing.T) {
	t.Setenv("RETHINKDB_HOST", "envhost")
	cfg := &rootConfig{host: "localhost"}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if cfg.host != "envhost" {
		t.Errorf("got %q, want %q", cfg.host, "envhost")
	}
}

func TestEnvVarPort(t *testing.T) {
	t.Setenv("RETHINKDB_PORT", "19015")
	cfg := &rootConfig{port: 28015}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if cfg.port != 19015 {
		t.Errorf("got %d, want %d", cfg.port, 19015)
	}
}

func TestEnvVarPortInvalid(t *testing.T) {
	t.Setenv("RETHINKDB_PORT", "notanumber")
	cfg := &rootConfig{port: 28015}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err == nil {
		t.Error("expected error for invalid RETHINKDB_PORT, got nil")
	}
	if cfg.port != 28015 {
		t.Errorf("port should remain unchanged after error, got %d", cfg.port)
	}
}

func TestEnvVarUser(t *testing.T) {
	t.Setenv("RETHINKDB_USER", "envuser")
	cfg := &rootConfig{user: "admin"}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if cfg.user != "envuser" {
		t.Errorf("got %q, want %q", cfg.user, "envuser")
	}
}

func TestEnvVarPassword(t *testing.T) {
	t.Setenv("RETHINKDB_PASSWORD", "envpass")
	cfg := &rootConfig{}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if cfg.password != "envpass" {
		t.Errorf("got %q, want %q", cfg.password, "envpass")
	}
}

func TestEnvVarDatabase(t *testing.T) {
	t.Setenv("RETHINKDB_DATABASE", "envdb")
	cfg := &rootConfig{}
	if err := cfg.resolveEnvVars(func(string) bool { return false }); err != nil {
		t.Fatal(err)
	}
	if cfg.database != "envdb" {
		t.Errorf("got %q, want %q", cfg.database, "envdb")
	}
}

func TestFlagPrecedenceOverEnvVar(t *testing.T) {
	t.Setenv("RETHINKDB_HOST", "envhost")
	t.Setenv("RETHINKDB_PORT", "19015")
	t.Setenv("RETHINKDB_USER", "envuser")
	t.Setenv("RETHINKDB_PASSWORD", "envpass")
	t.Setenv("RETHINKDB_DATABASE", "envdb")

	cfg := &rootConfig{
		host:     "flaghost",
		port:     12345,
		user:     "flaguser",
		password: "flagpass",
		database: "flagdb",
	}
	// simulate all flags explicitly set
	if err := cfg.resolveEnvVars(func(string) bool { return true }); err != nil {
		t.Fatal(err)
	}

	if cfg.host != "flaghost" {
		t.Errorf("host: got %q, want %q", cfg.host, "flaghost")
	}
	if cfg.port != 12345 {
		t.Errorf("port: got %d, want %d", cfg.port, 12345)
	}
	if cfg.user != "flaguser" {
		t.Errorf("user: got %q, want %q", cfg.user, "flaguser")
	}
	if cfg.password != "flagpass" {
		t.Errorf("password: got %q, want %q", cfg.password, "flagpass")
	}
	if cfg.database != "flagdb" {
		t.Errorf("database: got %q, want %q", cfg.database, "flagdb")
	}
}

func TestProfileFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetBool("profile")
	if err != nil {
		t.Fatal(err)
	}
	if v {
		t.Error("profile flag: expected false by default")
	}
}

func TestTimeFormatFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetString("time-format")
	if err != nil {
		t.Fatal(err)
	}
	if v != "native" {
		t.Errorf("time-format: got %q, want %q", v, "native")
	}
}

func TestBinaryFormatFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetString("binary-format")
	if err != nil {
		t.Fatal(err)
	}
	if v != "native" {
		t.Errorf("binary-format: got %q, want %q", v, "native")
	}
}

func TestQuietFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetBool("quiet")
	if err != nil {
		t.Fatal(err)
	}
	if v {
		t.Error("quiet flag: expected false by default")
	}
}

func TestVerboseFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetBool("verbose")
	if err != nil {
		t.Fatal(err)
	}
	if v {
		t.Error("verbose flag: expected false by default")
	}
}

func TestExitCodeSuccess(t *testing.T) {
	t.Parallel()
	if code := exitCode(nil); code != exitOK {
		t.Errorf("exitCode(nil): got %d, want %d", code, exitOK)
	}
}

func TestExitCodeConnection(t *testing.T) {
	t.Parallel()
	err := errors.New("dial tcp: connection refused")
	if code := exitCode(err); code != exitConnection {
		t.Errorf("exitCode(conn error): got %d, want %d", code, exitConnection)
	}
}

func TestExitCodeQuery(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
	}{
		{"compile", &response.ReqlCompileError{Msg: "syntax error"}},
		{"runtime", &response.ReqlRuntimeError{Msg: "runtime error"}},
		{"client", &response.ReqlClientError{Msg: "client error"}},
		{"nonexistence", &response.ReqlNonExistenceError{Msg: "not found"}},
		{"permission", &response.ReqlPermissionError{Msg: "permission denied"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if code := exitCode(tc.err); code != exitQuery {
				t.Errorf("exitCode(%s): got %d, want %d", tc.name, code, exitQuery)
			}
		})
	}
}

func TestExitCodeAuth(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("wrapped: %w", conn.ErrReqlAuth)
	if code := exitCode(err); code != exitAuth {
		t.Errorf("exitCode(auth): got %d, want %d", code, exitAuth)
	}
}

func TestSIGINTExitConstant(t *testing.T) {
	t.Parallel()
	if exitINT != 130 {
		t.Errorf("exitINT: got %d, want 130", exitINT)
	}
}

func TestSignalCancelsContext(t *testing.T) {
	ctx, stop := signal.NotifyContext(t.Context(), os.Interrupt)
	defer stop()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
		// context cancelled by signal as expected
	case <-time.After(time.Second):
		t.Error("context not cancelled after SIGINT")
	}
}

func TestVersionFlag(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "r-cli") {
		t.Errorf("version output does not contain 'r-cli': %q", out)
	}
}

// generateTestCert creates a self-signed ECDSA certificate and returns
// (certPEM, keyPEM) as byte slices.
func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}

func TestTLSCertFlagDefault(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	v, err := cmd.PersistentFlags().GetString("tls-cert")
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Errorf("tls-cert: got %q, want empty", v)
	}
}

// TestTLSCertFlagSetsCACert verifies that --tls-cert sets the CA certificate path
// and buildTLSConfig loads it into RootCAs.
func TestTLSCertFlagSetsCACert(t *testing.T) {
	t.Parallel()
	certPEM, _ := generateTestCert(t)
	dir := t.TempDir()
	certPath := dir + "/ca.pem"
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &rootConfig{tlsCACert: certPath}
	tlsCfg, err := cfg.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("buildTLSConfig: got nil, want *tls.Config")
	}
	if tlsCfg.RootCAs == nil {
		t.Error("RootCAs: got nil, want non-nil pool")
	}
}

// TestTLSClientCertAndKeyFlags verifies that --tls-client-cert + --tls-key
// configure client certificate authentication in the TLS config.
func TestTLSClientCertAndKeyFlags(t *testing.T) {
	t.Parallel()
	certPEM, keyPEM := generateTestCert(t)
	dir := t.TempDir()
	certPath := dir + "/client.pem"
	keyPath := dir + "/client-key.pem"
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &rootConfig{tlsClientCert: certPath, tlsKey: keyPath}
	tlsCfg, err := cfg.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("buildTLSConfig: got nil, want *tls.Config")
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("Certificates: got %d, want 1", len(tlsCfg.Certificates))
	}
}

// TestInsecureSkipVerifyFlag verifies that --insecure-skip-verify sets
// InsecureSkipVerify=true in the resulting TLS config.
func TestInsecureSkipVerifyFlag(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd()
	if err := cmd.ParseFlags([]string{"--insecure-skip-verify"}); err != nil {
		t.Fatal(err)
	}
	v, err := cmd.PersistentFlags().GetBool("insecure-skip-verify")
	if err != nil {
		t.Fatal(err)
	}
	if !v {
		t.Error("insecure-skip-verify: expected true after flag set")
	}

	cfg := &rootConfig{insecureSkipVerify: true}
	tlsCfg, err := cfg.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("buildTLSConfig: got nil, want *tls.Config")
	}
	if !tlsCfg.InsecureSkipVerify { //nolint:gosec
		t.Error("InsecureSkipVerify: expected true")
	}
}

func TestBuildTLSConfigNilWhenNoFlags(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{}
	tlsCfg, err := cfg.buildTLSConfig()
	if err != nil {
		t.Fatalf("buildTLSConfig: unexpected error: %v", err)
	}
	if tlsCfg != nil {
		t.Errorf("buildTLSConfig: got non-nil config when no TLS flags set")
	}
}

func TestBuildTLSConfigClientCertMissingKey(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{tlsClientCert: "/some/cert.pem"}
	_, err := cfg.buildTLSConfig()
	if err == nil {
		t.Error("expected error when --tls-client-cert set without --tls-key")
	}
}

func TestBuildTLSConfigKeyMissingClientCert(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{tlsKey: "/some/key.pem"}
	_, err := cfg.buildTLSConfig()
	if err == nil {
		t.Error("expected error when --tls-key set without --tls-client-cert")
	}
}

func TestBuildTLSConfigCACertNotFound(t *testing.T) {
	t.Parallel()
	cfg := &rootConfig{tlsCACert: "/nonexistent/ca.pem"}
	_, err := cfg.buildTLSConfig()
	if err == nil {
		t.Error("expected error for missing CA cert file")
	}
}
