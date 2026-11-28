package cbblocker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomicro/blockit/cbblocker"
)

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
		t.Fatal("expected signal did not arrive")
	}
}

func TestCallbackRetry(t *testing.T) {
	t.Parallel()

	var attempts int32
	b := cbblocker.New(context.Background(), func(context.Context) error {
		if atomic.AddInt32(&attempts, 1) < 3 {
			return errors.New("not ready")
		}

		return nil
	}, time.Millisecond)

	waitClosed(t, b.Blockit(context.Background()))
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestCallbackCancellationStopsRetry(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	var attempts int32
	b := cbblocker.New(context.Background(), func(context.Context) error {
		if atomic.AddInt32(&attempts, 1) == 1 {
			close(started)
		}

		return errors.New("not ready")
	}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := b.Blockit(ctx)
	waitSignal(t, started)
	cancel()
	waitClosed(t, done)

	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1", got)
	}
}

func TestCallbackCancellationInterruptsContextAwareCallback(t *testing.T) {
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
			b := cbblocker.New(base, func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				close(stopped)
				return ctx.Err()
			}, time.Hour)

			done := b.Blockit(ctx)
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
				t.Fatal("callback context was not canceled")
			}
		})
	}
}

func TestCallbackReceivesCancellationCause(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		constructor bool
	}{
		{name: "constructor", constructor: true},
		{name: "call"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base, cancelBase := context.WithCancelCause(context.Background())
			defer cancelBase(nil)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			started := make(chan struct{})
			causes := make(chan error, 1)
			b := cbblocker.New(base, func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				causes <- context.Cause(ctx)
				return ctx.Err()
			}, time.Hour)

			done := b.Blockit(ctx)
			waitSignal(t, started)
			cause := errors.New("shutdown requested")
			if tt.constructor {
				cancelBase(cause)
			} else {
				cancel(cause)
			}
			waitClosed(t, done)

			select {
			case got := <-causes:
				if !errors.Is(got, cause) {
					t.Errorf("cancellation cause = %v, want %v", got, cause)
				}
			case <-time.After(time.Second):
				t.Fatal("callback did not report its cancellation cause")
			}
		})
	}
}

func TestCallbackCauseWhenPollingStopsOnConstructorCancellation(t *testing.T) {
	t.Parallel()

	base, cancelBase := context.WithCancelCause(context.Background())
	defer cancelBase(nil)
	started := make(chan struct{})
	contexts := make(chan context.Context, 1)
	b := cbblocker.New(base, func(ctx context.Context) error {
		contexts <- ctx
		close(started)
		<-base.Done()
		return base.Err()
	}, time.Hour)

	done := b.Blockit(context.Background())
	waitSignal(t, started)
	cause := errors.New("shared shutdown")
	cancelBase(cause)
	waitClosed(t, done)

	callbackCtx := <-contexts
	if got := context.Cause(callbackCtx); !errors.Is(got, cause) {
		t.Errorf("cancellation cause = %v, want %v", got, cause)
	}
}

func TestCallbackReceivesConstructorDeadlineCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("shared deadline")
	base, cancel := context.WithTimeoutCause(context.Background(), 250*time.Millisecond, cause)
	defer cancel()
	started := make(chan struct{})
	causes := make(chan error, 1)
	b := cbblocker.New(base, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		causes <- context.Cause(ctx)
		return ctx.Err()
	}, time.Hour)

	done := b.Blockit(context.Background())
	waitSignal(t, started)
	waitClosed(t, done)

	select {
	case got := <-causes:
		if !errors.Is(got, cause) {
			t.Errorf("deadline cause = %v, want %v", got, cause)
		}
	case <-time.After(time.Second):
		t.Fatal("callback did not report its deadline cause")
	}
}

func TestCanceledCallbackDoesNotStart(t *testing.T) {
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
			var attempts int32
			b := cbblocker.New(base, func(context.Context) error {
				atomic.AddInt32(&attempts, 1)
				return nil
			}, time.Second)

			waitClosed(t, b.Blockit(ctx))
			if got := atomic.LoadInt32(&attempts); got != 0 {
				t.Errorf("attempts = %d, want 0", got)
			}
		})
	}
}

func TestUncooperativeCallbackDoesNotBlockCancellation(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	stopped := make(chan struct{})
	t.Cleanup(func() {
		close(release)
		waitSignal(t, stopped)
	})
	b := cbblocker.New(context.Background(), func(context.Context) error {
		close(started)
		<-release
		close(stopped)
		return errors.New("not ready")
	}, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := b.Blockit(ctx)
	waitSignal(t, started)
	cancel()
	waitClosed(t, done)
}

func TestAbandonedCompletion(t *testing.T) {
	t.Parallel()

	called := make(chan struct{})
	b := cbblocker.New(context.Background(), func(context.Context) error {
		close(called)
		return nil
	}, time.Second)
	done := b.Blockit(context.Background())

	waitSignal(t, called)
	waitClosed(t, done)
}

func TestCallbackContextCombinesValuesAndDeadlines(t *testing.T) {
	t.Parallel()

	type key string
	base := context.WithValue(context.Background(), key("base"), "constructor")
	base = context.WithValue(base, key("shared"), "constructor")
	base, cancelBase := context.WithTimeout(base, time.Minute)
	defer cancelBase()
	ctx := context.WithValue(context.Background(), key("call"), "blocking call")
	ctx = context.WithValue(ctx, key("shared"), "blocking call")
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	wantDeadline, hasDeadline := base.Deadline()
	if !hasDeadline {
		t.Fatal("constructor context has no deadline")
	}

	type observed struct {
		base     any
		call     any
		shared   any
		deadline time.Time
		hasLimit bool
	}
	values := make(chan observed, 1)
	b := cbblocker.New(base, func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		values <- observed{
			base:     ctx.Value(key("base")),
			call:     ctx.Value(key("call")),
			shared:   ctx.Value(key("shared")),
			deadline: deadline,
			hasLimit: ok,
		}
		return nil
	}, time.Second)

	waitClosed(t, b.Blockit(ctx))
	got := <-values
	if got.base != "constructor" || got.call != "blocking call" {
		t.Errorf("context values = (%v, %v), want constructor and blocking call", got.base, got.call)
	}
	if got.shared != "blocking call" {
		t.Errorf("shared context value = %v, want blocking call", got.shared)
	}
	if !got.hasLimit || !got.deadline.Equal(wantDeadline) {
		t.Errorf("deadline = (%v, %v), want (%v, true)", got.deadline, got.hasLimit, wantDeadline)
	}
}
