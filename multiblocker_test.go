package blockit_test

import (
	"context"
	"testing"
	"time"

	"github.com/gomicro/blockit"
	"github.com/gomicro/blockit/dbblocker"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestMultiBlocker(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("MultiBlocker", func() {
		g.Describe("Multiple Blockers", func() {
			g.It("should block with several blockers", func() {
				b := blockit.MultiBlocker{}
				b.Add(&foo{})
				b.Add(&bar{})

				Eventually(b.Blockit()).Should(Receive())
			})

			g.It("should block until context is cancelled", func() {
				ctx, cancel := context.WithCancel(context.Background())

				b := blockit.MultiBlocker{}
				b.Add(&never{})

				go func() {
					<-time.After(1 * time.Millisecond)
					cancel()
				}()

				Eventually(b.BlockitWithContext(ctx)).Should(Receive())
			})
		})

		g.Describe("DB Blocker", func() {
			g.It("should work in a multiblocker", func() {
				mockDB, _, _ := sqlmock.New()

				b := blockit.MultiBlocker{}
				b.Add(dbblocker.New(mockDB))

				Eventually(b.Blockit()).Should(Receive())
			})
		})
	})
}

type foo struct{}

func (f *foo) Blockit() <-chan struct{} {
	return f.BlockitWithContext(context.Background())
}

func (f *foo) BlockitWithContext(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})

	go func() {
		defer close(out)

		<-time.After(10 * time.Millisecond)
	}()

	return out
}

type bar struct{}

func (b *bar) Blockit() <-chan struct{} {
	return b.BlockitWithContext(context.Background())
}

func (b *bar) BlockitWithContext(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})

	go func() {
		defer close(out)

		<-time.After(30 * time.Millisecond)
	}()

	return out
}

type never struct{}

func (n *never) Blockit() <-chan struct{} {
	return n.BlockitWithContext(context.Background())
}

func (n *never) BlockitWithContext(ctx context.Context) <-chan struct{} {
	out := make(chan struct{})

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()

	return out
}
