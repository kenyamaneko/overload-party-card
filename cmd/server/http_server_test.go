package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/cmd/server/servertest"
)

const (
	testReadyPath = "/ready"
	testSlowPath  = "/slow"
)

type runningHTTPServer struct {
	baseURL string
	stop    context.CancelFunc
	exited  <-chan error
}

type slowResult struct {
	response *http.Response
	err      error
}

func startHTTPServer(t *testing.T, handler http.Handler) *runningHTTPServer {
	t.Helper()
	address := servertest.PickFreeAddress(t)
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: servertest.ReadHeaderTimeout}
	exited := make(chan error, 1)
	go func() { exited <- runHTTP(ctx, server) }()
	baseURL := "http://" + address
	client := servertest.NewHTTPClient()
	require.Eventually(t, func() bool {
		response, err := client.Get(baseURL + testReadyPath)
		return err == nil && response.StatusCode == http.StatusOK
	}, servertest.WaitTimeout, servertest.PollInterval)
	return &runningHTTPServer{baseURL: baseURL, stop: stop, exited: exited}
}

func newTestHandler(onSlowRequest func()) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(testReadyPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc(testSlowPath, func(w http.ResponseWriter, _ *http.Request) {
		onSlowRequest()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("done"))
	})
	return mux
}

func TestHTTPServer(t *testing.T) {
	t.Run("[HTTPサーバー]HTTPサーバーの起動と停止", func(t *testing.T) {
		t.Run("正常系", func(t *testing.T) {
			t.Run("停止を要求されたとき", func(t *testing.T) {
				t.Run("サーバーはエラーなしで終了する", func(t *testing.T) {
					server := startHTTPServer(t, newTestHandler(func() {}))

					server.stop()

					assert.NoError(t, servertest.AwaitExit(t, server.exited))
				})

				t.Run("終了後は、待受アドレスへの新しい接続が拒否される", func(t *testing.T) {
					server := startHTTPServer(t, newTestHandler(func() {}))
					server.stop()
					require.NoError(t, servertest.AwaitExit(t, server.exited))

					_, err := servertest.NewHTTPClient().Get(server.baseURL + testReadyPath)

					assert.ErrorIs(t, err, syscall.ECONNREFUSED)
				})
			})

			t.Run("リクエストの処理中に停止を要求されたとき、そのリクエストにステータス200、本文doneの応答が返る", func(t *testing.T) {
				started := make(chan struct{})
				release := make(chan struct{})
				server := startHTTPServer(t, newTestHandler(func() {
					close(started)
					<-release
				}))
				results := make(chan slowResult, 1)
				go func() {
					response, err := servertest.NewHTTPClient().Get(server.baseURL + testSlowPath)
					results <- slowResult{response: response, err: err}
				}()
				<-started
				server.stop()
				require.Eventually(t, func() bool {
					_, err := servertest.NewHTTPClient().Get(server.baseURL + testReadyPath)
					return err != nil
				}, servertest.WaitTimeout, servertest.PollInterval)
				close(release)

				result := <-results

				require.NoError(t, result.err)
				body, err := io.ReadAll(result.response.Body)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, result.response.StatusCode)
				assert.Equal(t, "done", string(body))
			})
		})

		t.Run("異常系", func(t *testing.T) {
			t.Run("待受アドレスがほかのサーバーに使われているとき、待受アドレスが使われていることを示すエラーになる", func(t *testing.T) {
				occupant, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, occupant.Close()) })
				server := &http.Server{Addr: occupant.Addr().String(), ReadHeaderTimeout: servertest.ReadHeaderTimeout}
				exited := make(chan error, 1)
				go func() { exited <- runHTTP(context.Background(), server) }()

				err = servertest.AwaitExit(t, exited)

				assert.ErrorIs(t, err, syscall.EADDRINUSE)
			})
		})
	})
}
