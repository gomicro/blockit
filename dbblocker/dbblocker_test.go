package dbblocker_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomicro/blockit"
	"github.com/gomicro/blockit/dbblocker"
)

type pingConnector struct {
	ping func(context.Context) error
}

func (c pingConnector) Connect(context.Context) (driver.Conn, error) {
	return pingConn(c), nil
}

func (c pingConnector) Driver() driver.Driver {
	return pingDriver(c)
}

type pingDriver struct {
	ping func(context.Context) error
}

func (d pingDriver) Open(string) (driver.Conn, error) {
	return pingConn(d), nil
}

type pingConn struct {
	ping func(context.Context) error
}

func (pingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (pingConn) Close() error {
	return nil
}

func (pingConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions are not supported")
}

func (c pingConn) Ping(ctx context.Context) error {
	err := c.ping(ctx)
	return err
}

func newDB(t *testing.T, ping func(context.Context) error) *sql.DB {
	t.Helper()

	db := sql.OpenDB(pingConnector{ping: ping})
	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Errorf("closing database: %v", err)
		}
	})
	return db
}

func waitClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case _, ok := <-done:
		if ok {
			t.Error("completion sent a value instead of closing")
		}
	case <-time.After(time.Second):
		t.Fatal("completion did not close")
	}
}

func waitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("database ping did not start")
	}
}

func TestDatabaseBlocker(t *testing.T) {
	t.Parallel()

	var calls int32
	db := newDB(t, func(context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})
	waitClosed(t, dbblocker.New(context.Background(), db).Blockit(context.Background()))
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("ping calls = %d, want 1", got)
	}
}

func TestDatabaseInMultiBlocker(t *testing.T) {
	t.Parallel()

	db := newDB(t, func(context.Context) error {
		return nil
	})
	var b blockit.MultiBlocker
	b.Add(dbblocker.New(context.Background(), db))
	waitClosed(t, b.Blockit(context.Background()))
}

func TestDatabaseContextCancellation(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		constructor bool
	}{
		{name: "constructor", constructor: true},
		{name: "call"},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base, cancelBase := context.WithCancel(context.Background())
			defer cancelBase()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			stopped := make(chan struct{})
			db := newDB(t, func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				close(stopped)
				return ctx.Err()
			})

			done := dbblocker.New(base, db).Blockit(ctx)
			waitSignal(t, started)
			if tt.constructor {
				cancelBase()
			} else {
				cancel()
			}
			waitClosed(t, done)

			select {
			case <-stopped:
			case <-time.After(time.Second):
				t.Fatal("database ping context was not canceled")
			}
		})
	}
}

func TestDatabaseAlreadyCanceled(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		constructor bool
	}{
		{name: "constructor", constructor: true},
		{name: "call"},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base, cancelBase := context.WithCancel(context.Background())
			defer cancelBase()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.constructor {
				cancelBase()
			} else {
				cancel()
			}
			var calls int32
			db := newDB(t, func(context.Context) error {
				atomic.AddInt32(&calls, 1)
				return nil
			})

			waitClosed(t, dbblocker.New(base, db).Blockit(ctx))
			if got := atomic.LoadInt32(&calls); got != 0 {
				t.Errorf("ping calls = %d, want 0", got)
			}
		})
	}
}
