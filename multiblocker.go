package blockit

import (
	"context"
)

// MultiBlocker represent many blockers to monitor and determine when they are
// done.
type MultiBlocker struct {
	blockers []Blocker
}

// Add takes one to many blockers to append into the multiblocker
func (b *MultiBlocker) Add(blocker ...Blocker) {
	b.blockers = append(b.blockers, blocker...)
}

// Blockit meets the blocker interface and summarizes multiple blockers into a
// single channel for blocking on.
func (b *MultiBlocker) Blockit() <-chan struct{} {
	return b.BlockitWithContext(context.Background())
}

// BlockitWithContext takes a context to include with the blockers. The
// multiblocker will return if the context is cancelled before the blockers
// finish.
func (b *MultiBlocker) BlockitWithContext(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})

	dones := make([]chan struct{}, len(b.blockers))
	for i := range dones {
		dones[i] = make(chan struct{})
	}

	for i := range b.blockers {
		go func(j int) {
			dones[j] <- <-b.blockers[j].BlockitWithContext(ctx)
		}(i)
	}

	go func() {
		for i := range dones {
			<-dones[i]
		}

		out <- struct{}{}
		close(out)
	}()

	return out
}
