package conn

import (
	"strings"
	"sync"
	"testing"
)

func TestNextTokenMonotonic(t *testing.T) {
	t.Parallel()
	c := &Conn{}
	prev := c.nextToken()
	for range 100 {
		next := c.nextToken()
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
				tokens[i] = c.nextToken()
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
