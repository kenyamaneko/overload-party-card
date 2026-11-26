//go:build integration

package main

import (
	"os"
	"testing"

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
