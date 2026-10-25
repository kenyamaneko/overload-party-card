//go:build integration

package repository_test

import (
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kenyamaneko/overload-party-card/internal/repository/repositorytest"
)

var sharedPg *repositorytest.Postgres

func TestMain(m *testing.M) {
	os.Exit(repositorytest.RunMain(
		m,
		&sharedPg,
		repositorytest.WithSchemaFile("db/schema.sql"),
		repositorytest.WithSchema("card"),
	))
}

func preparePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	sharedPg.Truncate(t)
	return sharedPg.Pool
}
