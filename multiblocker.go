package blockit

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
func (b *MultiBlocker) Blockit() <-chan bool {
	out := make(chan bool)

	dones := make([]chan bool, len(b.blockers))
	for i := range dones {
		dones[i] = make(chan bool)
	}

	for i := range b.blockers {
		go func(j int) {
			<-b.blockers[j].Blockit()
			dones[j] <- true
		}(i)
	}

	go func() {
		done := false
		for i := range dones {
			done = <-dones[i]
		}

		if done {
			out <- true
		}
		close(out)
	}()

	return out
}
