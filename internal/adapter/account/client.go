// Package account は account サービスの内部 API を呼び出すクライアントを提供します。
package account

import (
	"context"
	"fmt"
	"net/http"
	"time"

	apiaccount "github.com/kenyamaneko/overload-party-account/packages/api-account"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

const requestTimeout = 5 * time.Second

var _ port.FactionClient = (*Client)(nil)

// Client は account サービスの内部 API クライアントです。
type Client struct {
	api *apiaccount.ClientWithResponses
}

// NewClient は baseURL の account サービスに接続する Client を生成します。
func NewClient(baseURL string) (*Client, error) {
	api, err := apiaccount.NewClientWithResponses(baseURL,
		apiaccount.WithHTTPClient(&http.Client{Timeout: requestTimeout}),
	)
	if err != nil {
		return nil, fmt.Errorf("new account client: %w", err)
	}
	return &Client{api: api}, nil
}

// ListPlayerFactions は指定プレイヤーが所持する陣営の一覧を取得します。
func (c *Client) ListPlayerFactions(ctx context.Context, playerID string) ([]string, error) {
	resp, err := c.api.ListPlayerFactionsWithResponse(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("call account: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("account returned status %d for player factions", resp.StatusCode())
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("account returned status %d without a JSON body (content type %q) for player factions", resp.StatusCode(), resp.ContentType())
	}
	return resp.JSON200.Factions, nil
}
