package cbblocker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gomicro/blockit/cbblocker"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
)

func TestCallbackBlockers(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("Callback Blocking", func() {
		g.It("should block until callback returns no error", func() {
			ep := eventualPass{}

			b := cbblocker.New(ep.do, 10*time.Millisecond)
			Eventually(b.Blockit()).Should(Receive())
			Expect(ep.Fails).To(Equal(4))
		})

		g.It("should block with context", func() {
			n := never{}
			b := cbblocker.New(n.do, 10*time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				<-time.After(4 * time.Millisecond)
				cancel()
			}()

			Eventually(b.BlockitWithContext(ctx)).Should(Receive())
		})
	})
}

type eventualPass struct {
	Fails int
}

func (ep *eventualPass) do() error {
	if ep.Fails > 3 {
		return nil
	}

	ep.Fails++

	return fmt.Errorf("didn't pass")
}

type never struct {
}

func (n *never) do() error {
	return fmt.Errorf("will never pass")
}
