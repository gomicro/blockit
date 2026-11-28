package blockit

import (
	"context"
	"sync"
)

// MultiBlocker completes when all its blockers complete. It must not be copied
// after first use.
type MultiBlocker struct {
	// mu protects blockers.
	mu       sync.Mutex
	blockers []Blocker
}

// Add appends blockers for future calls to Blockit.
func (b *MultiBlocker) Add(blocker ...Blocker) {
	b.mu.Lock()
	b.blockers = append(b.blockers, blocker...)
	b.mu.Unlock()
}

// Blockit closes its completion channel when the current snapshot of blockers
// completes or ctx is canceled. It sends no value on the channel. Child blocker
// methods must return their channels promptly, and their own work must honor ctx
// to stop.
func (b *MultiBlocker) Blockit(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})
	if ctx.Err() != nil {
		close(out)
		return out
	}

	b.mu.Lock()
	blockers := make([]Blocker, len(b.blockers))
	copy(blockers, b.blockers)
	b.mu.Unlock()

	go func() {
		defer close(out)

		dones := make([]<-chan struct{}, 0, len(blockers))
		for _, blocker := range blockers {
			if ctx.Err() != nil {
				return
			}

			dones = append(dones, blocker.Blockit(ctx))
		}

		for _, done := range dones {
			select {
			case <-ctx.Done():
				return
			case <-done:
			}
		}
	}()

	return out
}
