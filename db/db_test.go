package db_test

import (
	"testing"

	"github.com/gomicro/blockit/db"

	. "github.com/franela/goblin"
	. "github.com/onsi/gomega"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestDefaultBlockers(t *testing.T) {
	g := Goblin(t)
	RegisterFailHandler(func(m string, _ ...int) { g.Fail(m) })

	g.Describe("DB Blocking", func() {
		g.It("should block until connected to the db", func() {
			mockDB, _, _ := sqlmock.New()

			b := db.New(mockDB)
			Eventually(<-b.Blockit()).Should(BeTrue())
		})
	})
}
