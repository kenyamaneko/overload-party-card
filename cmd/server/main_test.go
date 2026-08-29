package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
	"github.com/kenyamaneko/overload-party-card/internal/repository/postgrestest"
)

// mainSubprocessEnvKey は本テストバイナリを子プロセスとして再実行させ、main() の
// os.Exit を伴う異常終了経路を検証するためのマーカー。
const mainSubprocessEnvKey = "CARD_MAIN_TEST_SUBPROCESS"

var sharedPg *postgrestest.Postgres

func TestMain(m *testing.M) {
	if os.Getenv(mainSubprocessEnvKey) == "1" {
		main()
		return
	}
	os.Exit(postgrestest.RunMain(m, &sharedPg,
		postgrestest.WithSchemaFile("db/schema.sql"),
		postgrestest.WithSchemaFile("db/seed/products_seed.sql"),
		postgrestest.WithSchemaFile("db/seed/initiatives_seed.sql"),
		postgrestest.WithSchemaFile("db/seed/cards_seed.sql"),
		postgrestest.WithSchema("card"),
	))
}

// captureOutput は streamPtr (&os.Stdout / &os.Stderr) を一時的にパイプへ差し替えて
// fn 実行中の書き込みを捕捉する。
func captureOutput(t *testing.T, streamPtr **os.File, fn func()) string {
	t.Helper()
	orig := *streamPtr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	*streamPtr = w

	fn()

	require.NoError(t, w.Close())
	*streamPtr = orig

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	return string(data)
}

// findJSONLineContaining は複数行の出力から target を含み、かつ JSON として解釈できる
// 行を 1 行取り出す。テスト実行中に混在しうる無関係な出力行に頑健にするため。
func findJSONLineContaining(t *testing.T, output, target string) map[string]any {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, target) {
			continue
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(line), &got); err == nil {
			return got
		}
	}
	t.Fatalf("no JSON line containing %q found in output: %q", target, output)
	return nil
}

// findLineContaining は複数行の出力から target を含む行を 1 行取り出す。
func findLineContaining(t *testing.T, output, target string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, target) {
			return line
		}
	}
	t.Fatalf("no line containing %q found in output: %q", target, output)
	return ""
}

func TestRunHTTP(t *testing.T) {
	t.Run("HTTPサーバの起動と停止", func(t *testing.T) {
		t.Run("起動処理のコンテキストがキャンセルされたとき、エラーを返さずに戻り、戻った後は当該アドレスへの新規接続が確立しなくなる", func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			addr := ln.Addr().String()
			require.NoError(t, ln.Close())

			srv := &http.Server{Addr: addr, Handler: http.NewServeMux()}
			ctx, cancel := context.WithCancel(context.Background())

			errCh := make(chan error, 1)
			go func() { errCh <- runHTTP(ctx, srv) }()

			require.Eventually(t, func() bool {
				conn, dialErr := net.DialTimeout("tcp", addr, time.Second)
				if dialErr != nil {
					return false
				}
				_ = conn.Close()
				return true
			}, 5*time.Second, 10*time.Millisecond, "server did not start listening")

			cancel()

			select {
			case err := <-errCh:
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("runHTTP did not return after context cancellation")
			}

			_, dialErr := net.DialTimeout("tcp", addr, time.Second)
			assert.Error(t, dialErr)
		})

		t.Run("指定アドレスが既に使用中のとき、起動処理はエラーを返す", func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer func() { _ = ln.Close() }()
			addr := ln.Addr().String()

			srv := &http.Server{Addr: addr, Handler: http.NewServeMux()}

			err = runHTTP(context.Background(), srv)

			assert.Error(t, err)
		})
	})
}

func TestCloudLoggingHandler(t *testing.T) {
	t.Run("Cloud Logging形式のログ整形", func(t *testing.T) {
		tests := []struct {
			name         string
			level        slog.Level
			wantSeverity string
		}{
			{"Infoレベルでログを出力すると、標準出力にseverityキーの値が\"INFO\"である1行のJSONとして出力される", slog.LevelInfo, "INFO"},
			{"Warnレベルでログを出力すると、標準出力にseverityキーの値が\"WARNING\"である1行のJSONとして出力される", slog.LevelWarn, "WARNING"},
			{"Errorレベルでログを出力すると、標準出力にseverityキーの値が\"ERROR\"である1行のJSONとして出力される", slog.LevelError, "ERROR"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				const msg = "dummy cloud logging severity message"

				// newCloudLoggingHandler は構築時点の os.Stdout を書き込み先として
				// 保持するため、差し替え後に構築する必要がある。
				output := captureOutput(t, &os.Stdout, func() {
					logger := slog.New(newCloudLoggingHandler())
					logger.Log(context.Background(), tt.level, msg)
				})

				got := findJSONLineContaining(t, output, msg)
				assert.Equal(t, tt.wantSeverity, got["severity"])
			})
		}

		t.Run("ログを出力すると、ログ本文はmessageキーに出力される", func(t *testing.T) {
			const msg = "dummy cloud logging message body"

			output := captureOutput(t, &os.Stdout, func() {
				logger := slog.New(newCloudLoggingHandler())
				logger.Info(msg)
			})

			got := findJSONLineContaining(t, output, msg)
			assert.Equal(t, msg, got["message"])
		})
	})
}

func TestSetupLogger(t *testing.T) {
	t.Run("環境別のログ初期化", func(t *testing.T) {
		prodStgTests := []struct {
			name string
			env  config.Env
		}{
			{"envがprodのとき、エラーを返さず、以降のログがCloud Logging形式(severity/messageキーを持つJSON)で標準出力に出力される", config.EnvProd},
			{"envがstgのとき、エラーを返さず、以降のログがCloud Logging形式(severity/messageキーを持つJSON)で標準出力に出力される", config.EnvStg},
		}
		for _, tt := range prodStgTests {
			t.Run(tt.name, func(t *testing.T) {
				orig := slog.Default()
				defer slog.SetDefault(orig)

				const msg = "dummy setup logger prod stg message"

				// setupLogger が内部で newCloudLoggingHandler を構築する時点の
				// os.Stdout を書き込み先として保持するため、差し替え後に呼ぶ必要がある。
				var setupErr error
				output := captureOutput(t, &os.Stdout, func() {
					setupErr = setupLogger(tt.env)
					slog.Info(msg)
				})
				require.NoError(t, setupErr)

				got := findJSONLineContaining(t, output, msg)
				assert.Equal(t, "INFO", got["severity"])
				assert.Equal(t, msg, got["message"])
			})
		}

		localDevTests := []struct {
			name string
			env  config.Env
		}{
			{"envがlocalのとき、エラーを返さず、以降のログが標準エラー出力へプレーンテキスト形式で出力される", config.EnvLocal},
			{"envがdevのとき、エラーを返さず、以降のログが標準エラー出力へプレーンテキスト形式で出力される", config.EnvDev},
		}
		for _, tt := range localDevTests {
			t.Run(tt.name, func(t *testing.T) {
				orig := slog.Default()
				defer slog.SetDefault(orig)

				const msg = "dummy setup logger local dev message"

				var setupErr error
				output := captureOutput(t, &os.Stderr, func() {
					setupErr = setupLogger(tt.env)
					slog.Info(msg)
				})
				require.NoError(t, setupErr)

				line := findLineContaining(t, output, msg)
				assert.Contains(t, line, "level=INFO")
			})
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("起動時の内部認証公開鍵検証", func(t *testing.T) {
		t.Run("INTERNAL_AUTH_PUBLIC_KEYが空でない文字列だがPEMとして解釈できないとき、起動処理はエラーを返し、そのエラーの内容にINTERNAL_AUTH_PUBLIC_KEY_is_invalidという文言を含む", func(t *testing.T) {
			orig := slog.Default()
			defer slog.SetDefault(orig)

			dsn, err := sharedPg.Container.ConnectionString(context.Background(), "sslmode=disable")
			require.NoError(t, err)

			t.Setenv("PORT", "8080")
			t.Setenv("ENV", "local")
			t.Setenv("DATABASE_CONN", dsn)
			t.Setenv("DATABASE_IAM_AUTH_ENABLED", "false")
			t.Setenv("INTERNAL_AUTH_PUBLIC_KEY", "not-a-valid-pem")
			t.Setenv("ACCOUNT_SERVICE_URL", "http://account.internal.test")

			err = run()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "INTERNAL_AUTH_PUBLIC_KEY is invalid")
		})
	})
}

func TestMainProcessExit(t *testing.T) {
	t.Run("起動失敗時のプロセス終了処理", func(t *testing.T) {
		t.Run("runがエラーを返したとき、card_fatalを含むログを標準エラー出力し、終了コード1でプロセスを終了する", func(t *testing.T) {
			execPath, err := os.Executable()
			require.NoError(t, err)

			cmd := exec.Command(execPath)
			cmd.Env = []string{mainSubprocessEnvKey + "=1"}
			var stderr strings.Builder
			cmd.Stderr = &stderr

			runErr := cmd.Run()

			var exitErr *exec.ExitError
			require.ErrorAs(t, runErr, &exitErr)
			assert.Equal(t, 1, exitErr.ExitCode())
			assert.Contains(t, stderr.String(), "card fatal")
		})
	})
}
