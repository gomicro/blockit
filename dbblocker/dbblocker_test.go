package dbblocker_test

import (
	"context"
	"testing"
	"time"

	"github.com/gomicro/blockit/dbblocker"

	"github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestDefaultBlockers(t *testing.T) {
	g := goblin.Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("DB Blocking", func() {
		g.It("should block until connected to the db", func() {
			mockDB, _, _ := sqlmock.New()

			b := dbblocker.New(mockDB)
			Eventually(b.Blockit()).Should(Receive())
		})

		g.It("should block with context", func() {
			mockDB, _, _ := sqlmock.New()
			b := dbblocker.New(mockDB)
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				<-time.After(2 * time.Millisecond)
				cancel()
			}()

			Eventually(b.BlockitWithContext(ctx)).Should(Receive())
		})
	})
}
