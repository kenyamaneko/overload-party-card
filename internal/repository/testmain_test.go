//go:build integration

package repository_test

import (
	"os"
	"testing"

	"github.com/kenyamaneko/overload-party-card/internal/repository/postgrestest"
)

// sharedPg はパッケージ全体で共有する Postgres コンテナ。
// TestMain で起動し、各テストは Truncate で状態リセットして使う。
var sharedPg *postgrestest.Postgres

func TestMain(m *testing.M) {
	os.Exit(postgrestest.RunMain(m, &sharedPg,
		postgrestest.WithSchemaFile("db/schema.sql"),
		postgrestest.WithSchema("card"),
	))
}
