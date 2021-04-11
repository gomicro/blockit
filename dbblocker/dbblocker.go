package dbblocker

import (
	"database/sql"
	"time"

	"github.com/gomicro/blockit/cbblocker"
)

// New takes a SQL database object and returns a newly instantiated Blocker
func New(db *sql.DB) *cbblocker.Blocker {
	return cbblocker.New(db.Ping, 1*time.Second)
}
