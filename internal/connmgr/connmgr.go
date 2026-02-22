package connmgr

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"r-cli/internal/conn"
)

// DialFunc creates a new connection.
type DialFunc func(ctx context.Context) (*conn.Conn, error)

// ConnManager manages a single lazily-created connection.
type ConnManager struct {
	dial DialFunc
	mu   sync.Mutex
	c    *conn.Conn
}

// New creates a ConnManager using the provided dial function.
func New(dial DialFunc) *ConnManager {
	return &ConnManager{dial: dial}
}

// NewFromConfig creates a ConnManager that dials addr using the given config.
func NewFromConfig(cfg conn.Config, tlsCfg *tls.Config) *ConnManager {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return New(func(ctx context.Context) (*conn.Conn, error) {
		return conn.Dial(ctx, addr, cfg, tlsCfg)
	})
}

// Get returns the current connection, creating one lazily on first call.
func (m *ConnManager) Get(ctx context.Context) (*conn.Conn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.c != nil {
		return m.c, nil
	}
	c, err := m.dial(ctx)
	if err != nil {
		return nil, err
	}
	m.c = c
	return m.c, nil
}

// Close closes the managed connection if one exists.
func (m *ConnManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.c == nil {
		return nil
	}
	err := m.c.Close()
	m.c = nil
	return err
}
