package dbblocker_test

import (
	"testing"

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
			Eventually(<-b.Blockit()).Should(BeTrue())
		})
	})
}
