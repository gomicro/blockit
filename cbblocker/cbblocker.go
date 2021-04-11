package cbblocker

import (
	"time"
)

type Blocker struct {
	callback func() error
	duration time.Duration
}

func New(callback func() error, poll time.Duration) *Blocker {
	return &Blocker{
		callback: callback,
		duration: poll,
	}
}

func (g *Blocker) Blockit() <-chan bool {
	pass := make(chan bool)

	go func() {
		for {
			err := g.callback()
			if err != nil {
				<-time.After(g.duration)
				continue
			}

			pass <- true
			close(pass)

			break
		}
	}()

	return pass
}
