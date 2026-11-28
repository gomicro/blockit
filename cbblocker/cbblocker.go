package cbblocker

import (
	"context"
	"time"
)

// Blocker polls a callback until it succeeds or its context is canceled.
type Blocker struct {
	ctx      context.Context
	callback func(context.Context) error
	duration time.Duration
}

// New polls callback until it returns nil or either the constructor
// context or blocking-call context is canceled. Callbacks must honor their
// context to stop an in-flight call. Blocking-call context values take
// precedence over constructor context values.
func New(ctx context.Context, callback func(context.Context) error, poll time.Duration) *Blocker {
	return &Blocker{
		ctx:      ctx,
		callback: callback,
		duration: poll,
	}
}

// Blockit closes its completion channel when polling finishes or
// either context is canceled. It sends no value on the channel.
func (b *Blocker) Blockit(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})
	if ctx.Err() != nil || b.ctx.Err() != nil {
		close(out)
		return out
	}

	ctx, cancel := context.WithCancelCause(ctx)
	callbackCtx := mergedContext{Context: ctx, base: b.ctx}
	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			if ctx.Err() != nil || b.ctx.Err() != nil {
				return
			}

			err := b.callback(callbackCtx)
			if err == nil {
				return
			}

			timer := time.NewTimer(b.duration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-b.ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}()

	go func() {
		defer close(out)
		defer cancel(nil)

		select {
		case <-ctx.Done():
		case <-b.ctx.Done():
			cancel(context.Cause(b.ctx))
		case <-done:
			if b.ctx.Err() != nil {
				cancel(context.Cause(b.ctx))
			}
		}
	}()

	return out
}

type mergedContext struct {
	context.Context
	base context.Context
}

func (c mergedContext) Deadline() (time.Time, bool) {
	deadline, ok := c.Context.Deadline()
	baseDeadline, baseOK := c.base.Deadline()
	if baseOK && (!ok || baseDeadline.Before(deadline)) {
		return baseDeadline, true
	}

	return deadline, ok
}

func (c mergedContext) Value(key any) any {
	value := c.Context.Value(key)
	if value != nil {
		return value
	}

	value = c.base.Value(key)
	return value
}
