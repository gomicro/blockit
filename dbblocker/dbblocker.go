package dbblocker

import (
	"context"
	"database/sql"
	"time"

	"github.com/gomicro/blockit/cbblocker"
)

// New polls db until a ping succeeds or either the constructor or
// blocking-call context is canceled.
func New(ctx context.Context, db *sql.DB) *cbblocker.Blocker {
	blocker := cbblocker.New(ctx, db.PingContext, time.Second)
	return blocker
}
