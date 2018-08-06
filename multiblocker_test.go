package blockit_test

import (
	"testing"
	"time"

	"github.com/gomicro/blockit"
	"github.com/gomicro/blockit/dbblocker"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestMultiBlocker(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("MultiBlocker", func() {
		g.Describe("Multiple Blockers", func() {
			g.It("should block with several blockers", func() {
				b := blockit.MultiBlocker{}

				f := &foo{}
				b.Add(f)

				ba := &bar{}
				b.Add(ba)

				Eventually(<-b.Blockit()).Should(BeTrue())
			})
		})

		g.Describe("DB Blocker", func() {
			g.It("should work in a multiblocker", func() {
				mockDB, _, _ := sqlmock.New()

				b := blockit.MultiBlocker{}
				b.Add(dbblocker.New(mockDB))
				Eventually(<-b.Blockit()).Should(BeTrue())
			})
		})
	})
}

type foo struct{}
type bar struct{}

func (f *foo) Blockit() <-chan bool {
	out := make(chan bool)

	go func() {
		<-time.After(1 * time.Second)
		close(out)
	}()

	return out
}

func (b *bar) Blockit() <-chan bool {
	out := make(chan bool)

	go func() {
		<-time.After(3 * time.Second)
		close(out)
	}()

	return out
}
