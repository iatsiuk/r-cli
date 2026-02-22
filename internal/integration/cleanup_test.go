//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"r-cli/internal/conn"
	"r-cli/internal/reql"
)

func TestUserCleanup(t *testing.T) {
	host, port := startRethinkDBWithPassword(t, "testpass")
	admin := adminExecAt(t, host, port, "testpass")
	ctx := context.Background()

	t.Run("DeletedUserCannotConnect", func(t *testing.T) {
		// insert user directly to manage its lifecycle explicitly in this test
		_, cur, err := admin.Run(ctx,
			reql.DB("rethinkdb").Table("users").Insert(
				map[string]interface{}{"id": "cleanup_del", "password": "pass"},
			),
			nil,
		)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("create user cleanup_del: %v", err)
		}

		dialCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// verify initial connect succeeds
		c, err := dialAs(dialCtx, host, port, "cleanup_del", "pass")
		if err != nil {
			// user was created; clean up before failing
			_, c2, _ := admin.Run(context.Background(),
				reql.DB("rethinkdb").Table("users").Get("cleanup_del").Delete(), nil)
			closeCursor(c2)
			t.Fatalf("initial connect as cleanup_del: %v", err)
		}
		_ = c.Close()

		// delete the user via admin
		_, cur, err = admin.Run(ctx,
			reql.DB("rethinkdb").Table("users").Get("cleanup_del").Delete(), nil)
		closeCursor(cur)
		if err != nil {
			t.Fatalf("delete user cleanup_del: %v", err)
		}

		// reconnect attempt must fail with auth error
		_, err = dialAs(dialCtx, host, port, "cleanup_del", "pass")
		if err == nil {
			t.Error("expected auth error after user deletion, got nil")
		} else if !errors.Is(err, conn.ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth after deletion, got %v (type %T)", err, err)
		}
	})

	t.Run("CleanupRemovesTestUsers", func(t *testing.T) {
		// Run a non-parallel subtest that creates a user via createUser.
		// After the subtest completes, its t.Cleanup callbacks run (including
		// the one that deletes the user), so the parent can verify no leftover state.
		const username = "cleanup_leftover"

		t.Run("createAndCleanup", func(t *testing.T) {
			createUser(t, admin, username, "pass")
			// subtest ends here; t.Cleanup (delete user) runs before t.Run returns
		})

		// at this point createUser's cleanup has fired and the user is deleted
		qCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := dialAs(qCtx, host, port, username, "pass")
		if err == nil {
			t.Error("expected deleted user to fail auth, but connect succeeded (leftover state)")
		} else if !errors.Is(err, conn.ErrReqlAuth) {
			t.Errorf("expected ErrReqlAuth for cleaned-up user, got %v (type %T)", err, err)
		}
	})
}
