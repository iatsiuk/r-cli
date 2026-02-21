package conn

import (
	"fmt"
	"sync/atomic"
)

// Config holds connection parameters.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"-"`
}

// String returns a human-readable representation of Config without the password.
func (c Config) String() string {
	return fmt.Sprintf("conn{%s:%d user=%s}", c.Host, c.Port, c.User)
}

// Conn manages a single RethinkDB connection with multiplexed query dispatch.
type Conn struct {
	token atomic.Uint64
}

// nextToken returns the next unique query token, incrementing the counter atomically.
func (c *Conn) nextToken() uint64 {
	return c.token.Add(1)
}
