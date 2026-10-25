// Package configtest は起動設定の環境変数を設定するテスト用の部品を提供する。
package configtest

import "testing"

// SetEnvironment は起動に必要な環境変数一式を、受け付けられる値で設定します。
// overrides に指定した環境変数は、その値で上書きします。
func SetEnvironment(t *testing.T, overrides map[string]string) {
	t.Helper()
	environment := map[string]string{
		"PORT":                      "8080",
		"ENV":                       "local",
		"DATABASE_CONN":             "TST-conn",
		"DATABASE_IAM_AUTH_ENABLED": "false",
		"CLOUDSQL_CONNECTION_NAME":  "",
		"INTERNAL_AUTH_PUBLIC_KEY":  "TST-public-key",
		"ACCOUNT_SERVICE_URL":       "http://TST-account.invalid",
	}
	for key, value := range overrides {
		environment[key] = value
	}
	for key, value := range environment {
		t.Setenv(key, value)
	}
}
