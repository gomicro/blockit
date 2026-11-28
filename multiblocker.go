package blockit

import "context"

// MultiBlocker completes when all its blockers complete.
type MultiBlocker struct {
	blockers []Blocker
}

// Add appends blockers to monitor.
func (b *MultiBlocker) Add(blocker ...Blocker) {
	b.blockers = append(b.blockers, blocker...)
}

// Blockit closes its completion channel when all blockers complete
// or ctx is canceled. It sends no value on the channel. Child blocker methods
// must return their channels promptly, and their own work must honor ctx to stop.
func (b *MultiBlocker) Blockit(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})
	if ctx.Err() != nil {
		close(out)
		return out
	}

	go func() {
		defer close(out)

		dones := make([]<-chan struct{}, 0, len(b.blockers))
		for _, blocker := range b.blockers {
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
