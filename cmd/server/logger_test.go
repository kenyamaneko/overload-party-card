package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/config"
)

func captureLogOutput(t *testing.T, env config.Env, emit func()) (stdout, stderr string) {
	t.Helper()
	originalStdout, originalStderr, originalLogger := os.Stdout, os.Stderr, slog.Default()
	t.Cleanup(func() {
		os.Stdout, os.Stderr = originalStdout, originalStderr
		slog.SetDefault(originalLogger)
	})
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter

	require.NoError(t, setupLogger(env))
	emit()

	os.Stdout, os.Stderr = originalStdout, originalStderr
	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())
	stdoutBytes, err := io.ReadAll(stdoutReader)
	require.NoError(t, err)
	stderrBytes, err := io.ReadAll(stderrReader)
	require.NoError(t, err)
	return string(stdoutBytes), string(stderrBytes)
}

func parseJSONLogLine(t *testing.T, output string) map[string]any {
	t.Helper()
	var entry map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &entry))
	return entry
}

func TestLogger(t *testing.T) {
	t.Run("[ログ出力]ログの環境別の出力形式", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("ENVがprodのとき", func(t *testing.T) {
				t.Run("本文helloのInfoログを出すと", func(t *testing.T) {
					t.Run("標準出力にhelloを含む出力が出る", func(t *testing.T) {
						stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Info("hello") })

						assert.Contains(t, stdout, "hello")
					})

					t.Run("出力は1行のJSONになる", func(t *testing.T) {
						stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Info("hello") })

						line := strings.TrimSuffix(stdout, "\n")
						assert.NotContains(t, line, "\n")
						assert.True(t, json.Valid([]byte(line)))
					})

					t.Run("severity項目はINFOになる", func(t *testing.T) {
						stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Info("hello") })

						assert.Equal(t, "INFO", parseJSONLogLine(t, stdout)["severity"])
					})

					t.Run("message項目はhelloになる", func(t *testing.T) {
						stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Info("hello") })

						assert.Equal(t, "hello", parseJSONLogLine(t, stdout)["message"])
					})
				})

				t.Run("Errorログのseverity項目はERRORになる", func(t *testing.T) {
					stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Error("hello") })

					assert.Equal(t, "ERROR", parseJSONLogLine(t, stdout)["severity"])
				})

				t.Run("Warnログのseverity項目はWARNINGになる", func(t *testing.T) {
					stdout, _ := captureLogOutput(t, config.EnvProd, func() { slog.Warn("hello") })

					assert.Equal(t, "WARNING", parseJSONLogLine(t, stdout)["severity"])
				})

				t.Run("本文helloのDebugログを出すと、標準出力にも標準エラー出力にもhelloを含む出力が出ない", func(t *testing.T) {
					stdout, stderr := captureLogOutput(t, config.EnvProd, func() { slog.Debug("hello") })

					assert.NotContains(t, stdout, "hello")
					assert.NotContains(t, stderr, "hello")
				})
			})

			t.Run("ENVがlocalのとき、本文helloのDebugログを出すと", func(t *testing.T) {
				t.Run("標準エラー出力にhelloを含む出力が出る", func(t *testing.T) {
					_, stderr := captureLogOutput(t, config.EnvLocal, func() { slog.Debug("hello") })

					assert.Contains(t, stderr, "hello")
				})

				t.Run("標準出力にはhelloを含む出力が出ない", func(t *testing.T) {
					stdout, _ := captureLogOutput(t, config.EnvLocal, func() { slog.Debug("hello") })

					assert.NotContains(t, stdout, "hello")
				})
			})

			t.Run("ENVごとのログのservice項目", func(t *testing.T) {
				tests := []struct {
					name        string
					env         config.Env
					readService func(t *testing.T, stdout, stderr string) string
				}{
					{
						name: "ENVがprodのとき、ログのservice項目はcardになる",
						env:  config.EnvProd,
						readService: func(t *testing.T, stdout, _ string) string {
							return fmt.Sprint(parseJSONLogLine(t, stdout)["service"])
						},
					},
					{
						name: "ENVがlocalのとき、ログのservice項目はcardになる",
						env:  config.EnvLocal,
						readService: func(t *testing.T, _, stderr string) string {
							match := regexp.MustCompile(`service=(\S+)`).FindStringSubmatch(stderr)
							require.Len(t, match, 2)
							return match[1]
						},
					},
				}
				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						stdout, stderr := captureLogOutput(t, tt.env, func() { slog.Info("hello") })

						assert.Equal(t, "card", tt.readService(t, stdout, stderr))
					})
				}
			})
		})
	})
}
