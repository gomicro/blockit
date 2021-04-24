package blockit

import (
	"context"
)

// Blocker represents an object that will have a Blockit function as a method
// for determining when the object is done with a given task.
type Blocker interface {
	Blockit() <-chan struct{}
	BlockitWithContext(ctx context.Context) <-chan struct{}
}
