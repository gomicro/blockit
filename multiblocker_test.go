package blockit_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomicro/blockit"
	"github.com/gomicro/blockit/cbblocker"
)

type testBlocker struct {
	done        <-chan struct{}
	started     chan struct{}
	startedOnce sync.Once
	calls       int32
}

func (b *testBlocker) Blockit(context.Context) <-chan struct{} {
	atomic.AddInt32(&b.calls, 1)
	if b.started != nil {
		b.startedOnce.Do(func() {
			close(b.started)
		})
	}

	return b.done
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
		t.Fatal("child did not start")
	}
}

func TestMultiBlockerWaitsForAll(t *testing.T) {
	t.Parallel()

	first := make(chan struct{})
	second := make(chan struct{})
	startedFirst := make(chan struct{})
	startedSecond := make(chan struct{})
	b := blockit.MultiBlocker{}
	b.Add(&testBlocker{done: first, started: startedFirst}, &testBlocker{done: second, started: startedSecond})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := b.Blockit(ctx)
	waitSignal(t, startedFirst)
	waitSignal(t, startedSecond)
	close(first)

	select {
	case <-done:
		t.Fatal("completion closed before all children finished")
	default:
	}
	close(second)
	waitClosed(t, done)
}

func TestMultiBlockerCancellation(t *testing.T) {
	t.Parallel()

	child := make(chan struct{})
	defer close(child)
	started := make(chan struct{})
	b := blockit.MultiBlocker{}
	b.Add(&testBlocker{done: child, started: started})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := b.Blockit(ctx)
	waitSignal(t, started)
	cancel()
	waitClosed(t, done)
}

func TestMultiBlockerAlreadyCanceled(t *testing.T) {
	t.Parallel()

	child := &testBlocker{done: make(chan struct{})}
	b := blockit.MultiBlocker{}
	b.Add(child)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	waitClosed(t, b.Blockit(ctx))
	if got := atomic.LoadInt32(&child.calls); got != 0 {
		t.Errorf("child calls = %d, want 0", got)
	}
}

func TestMultiBlockerPropagatesContext(t *testing.T) {
	t.Parallel()

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "value")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	values := make(chan any, 1)
	stopped := make(chan struct{})
	child := cbblocker.New(context.Background(), func(ctx context.Context) error {
		values <- ctx.Value(key{})
		<-ctx.Done()
		close(stopped)
		return ctx.Err()
	}, time.Second)
	var b blockit.MultiBlocker
	b.Add(child)
	done := b.Blockit(ctx)

	select {
	case value := <-values:
		if value != "value" {
			t.Errorf("child context value = %v, want value", value)
		}
	case <-time.After(time.Second):
		t.Fatal("child did not receive context")
	}

	cancel()
	waitClosed(t, done)
	waitSignal(t, stopped)
}

func TestMultiBlockerEmpty(t *testing.T) {
	t.Parallel()

	b := blockit.MultiBlocker{}
	waitClosed(t, b.Blockit(context.Background()))
}

func TestMultiBlockerUsesIndependentSnapshots(t *testing.T) {
	t.Parallel()

	first := make(chan struct{})
	added := make(chan struct{})
	addedStarted := make(chan struct{})
	addedBlocker := &testBlocker{done: added, started: addedStarted}
	var b blockit.MultiBlocker
	b.Add(&testBlocker{done: first})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstCall := b.Blockit(ctx)
	b.Add(addedBlocker)
	secondCall := b.Blockit(ctx)
	waitSignal(t, addedStarted)
	close(first)
	waitClosed(t, firstCall)

	select {
	case <-secondCall:
		t.Fatal("second call completed before the added blocker")
	default:
	}

	close(added)
	waitClosed(t, secondCall)
	if got := atomic.LoadInt32(&addedBlocker.calls); got != 1 {
		t.Errorf("added blocker calls = %d, want 1", got)
	}
}

func TestMultiBlockerConcurrentAddAndBlockit(t *testing.T) {
	t.Parallel()

	childDone := make(chan struct{})
	close(childDone)
	child := &testBlocker{done: childDone}
	var b blockit.MultiBlocker
	b.Add(child)
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)

	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 200; i++ {
			b.Add(child)
		}
	}()

	go func() {
		defer workers.Done()
		<-start
		for i := 0; i < 200; i++ {
			<-b.Blockit(context.Background())
		}
	}()

	close(start)
	workers.Wait()
}
