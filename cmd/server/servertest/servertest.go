// Package servertest はサーバーの起動と停止のテストで共有する部品を提供する。
package servertest

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// WaitTimeout は起動や終了を待つ上限です。
	WaitTimeout = 30 * time.Second
	// PollInterval は条件の成立を確かめる間隔です。
	PollInterval = 20 * time.Millisecond
	// HTTPClientTimeout はテスト用の HTTP クライアントの 1 リクエストあたりの上限です。
	HTTPClientTimeout = 5 * time.Second
	// ReadHeaderTimeout はテスト用の HTTP サーバーがリクエストヘッダーを読む上限です。
	ReadHeaderTimeout = 1 * time.Second
)

// NewHTTPClient は接続を使い回さない HTTP クライアントを生成します。
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   HTTPClientTimeout,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
}

// PickFreeAddress はローカルで空いている待受アドレスを選んで返します。
func PickFreeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}

// AwaitExit は exited に終了結果が届くのを待って返します。WaitTimeout を過ぎるとテストを失敗させます。
func AwaitExit(t *testing.T, exited <-chan error) error {
	t.Helper()
	select {
	case err := <-exited:
		return err
	case <-time.After(WaitTimeout):
		t.Fatal("timed out waiting for exit")
		return nil
	}
}
