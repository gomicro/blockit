package blockit

import "context"

// Blocker signals completion by closing its channel without sending a value.
type Blocker interface {
	Blockit(ctx context.Context) <-chan struct{}
}
