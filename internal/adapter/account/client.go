// Package account は account サービスの内部 API を叩く HTTP クライアントを提供します。
package account

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.FactionClient = (*Client)(nil)

// Client は account サービスの内部 API クライアントです。
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient は account サービスの Client を生成します。
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

type factionListing struct {
	Factions []string `json:"factions"`
}

// ListPlayerFactions は指定プレイヤーの所持ファクション一覧を取得します。
func (c *Client) ListPlayerFactions(ctx context.Context, playerID string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/internal/v1/players/%s/factions", c.baseURL, url.PathEscape(playerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build account request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call account: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("account returned status %d for player factions", resp.StatusCode)
	}

	var body factionListing
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode player factions: %w", err)
	}
	return body.Factions, nil
}
