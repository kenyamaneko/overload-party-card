// Package apicardclient は card サービスの REST API を呼び出すクライアントを提供する。
// 利用側にパスやステータスコードを意識させないため、ステータスコードをエラー値に変換して返す。
package apicardclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

var (
	// ErrNotFound は対象が存在しないこと (ステータス 404) を示す。
	ErrNotFound = errors.New("apicardclient: not found")

	// ErrUnauthorized は認証に失敗したこと (ステータス 401。X-Internal-Auth の不正など) を示す。
	ErrUnauthorized = errors.New("apicardclient: unauthorized")

	// ErrForbidden はアクセス権限が無いこと (ステータス 403。未所持のカードなど) を示す。
	ErrForbidden = errors.New("apicardclient: forbidden")

	// ErrBadRequest は要求内容が不正なこと (ステータス 400。ValidateDeckForBattle を除く) を示す。
	ErrBadRequest = errors.New("apicardclient: bad request")

	// ErrDeckInvalid はデッキがバトルで使えないこと (ValidateDeckForBattle のステータス 400) を示す。
	// 契約上、デッキの検証失敗と制限枚数超過がこれに当たる。
	ErrDeckInvalid = errors.New("apicardclient: deck invalid")

	// ErrInternalServer はサービスの内部エラー (ステータス 5xx) を示す。
	ErrInternalServer = errors.New("apicardclient: internal server error")

	// ErrUnexpectedResponse は契約に無い応答を受け取ったことを示す。
	// 分類のないステータスや、成功のステータスなのに本文を読み取れない応答がこれに当たる。
	ErrUnexpectedResponse = errors.New("apicardclient: unexpected response")
)

// Client は card サービスの REST API クライアントです。
type Client struct {
	api *apicard.ClientWithResponses
}

// Option は Client の設定です。
type Option func(*config)

type config struct {
	httpClient apicard.HttpRequestDoer
	editors    []apicard.RequestEditorFn
}

// WithHTTPClient は HTTP リクエストの送信に使うクライアントを差し替える。
func WithHTTPClient(doer apicard.HttpRequestDoer) Option {
	return func(c *config) { c.httpClient = doer }
}

// WithRequestEditorFn は全リクエストに適用する編集関数を追加する。
// X-Internal-Auth ヘッダの付与などに使う。
func WithRequestEditorFn(fn apicard.RequestEditorFn) Option {
	return func(c *config) { c.editors = append(c.editors, fn) }
}

// New は baseURL に接続する Client を生成する。
func New(baseURL string, opts ...Option) (*Client, error) {
	cfg := &config{}
	for _, option := range opts {
		option(cfg)
	}

	apiOptions := make([]apicard.ClientOption, 0)
	if cfg.httpClient != nil {
		apiOptions = append(apiOptions, apicard.WithHTTPClient(cfg.httpClient))
	}
	for _, editor := range cfg.editors {
		apiOptions = append(apiOptions, apicard.WithRequestEditorFn(editor))
	}

	api, err := apicard.NewClientWithResponses(baseURL, apiOptions...)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: new: %w", err)
	}
	return &Client{api: api}, nil
}

// GetHealth はサービス死活監視用エンドポイントを叩く。
func (c *Client) GetHealth(ctx context.Context) (*apicard.HealthResponse, error) {
	resp, err := c.api.GetHealthWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: GetHealth: %w", err)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, newStatusError("GetHealth", resp.StatusCode())
}

// ListPlayerCards はプレイヤーの所持カード一覧を返す。
func (c *Client) ListPlayerCards(ctx context.Context) ([]apicard.PlayerCardWithDef, error) {
	resp, err := c.api.ListPlayerCardsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: ListPlayerCards: %w", err)
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return nil, newStatusError("ListPlayerCards", resp.StatusCode())
}

// ListCardsWithOwnership は全カード定義にプレイヤー所持状態を付与して返す。
func (c *Client) ListCardsWithOwnership(ctx context.Context) ([]apicard.CardWithOwnership, error) {
	resp, err := c.api.ListCardsWithOwnershipWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: ListCardsWithOwnership: %w", err)
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return nil, newStatusError("ListCardsWithOwnership", resp.StatusCode())
}

// ListDecks はプレイヤーのデッキ一覧を返す。
func (c *Client) ListDecks(ctx context.Context) ([]apicard.Deck, error) {
	resp, err := c.api.ListDecksWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: ListDecks: %w", err)
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return nil, newStatusError("ListDecks", resp.StatusCode())
}

// GetDeck はデッキ詳細を返す。カード構成は応答の deck_cards フィールドに含まれる。
func (c *Client) GetDeck(ctx context.Context, deckID int64) (*apicard.Deck, error) {
	resp, err := c.api.GetDeckWithResponse(ctx, deckID)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: GetDeck: %w", err)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, newStatusError("GetDeck", resp.StatusCode())
}

// CreateDeck は新しいデッキを作成する。
func (c *Client) CreateDeck(ctx context.Context, req apicard.DeckCreateRequest) (*apicard.Deck, error) {
	resp, err := c.api.CreateDeckWithResponse(ctx, apicard.CreateDeckJSONRequestBody(req))
	if err != nil {
		return nil, fmt.Errorf("apicardclient: CreateDeck: %w", err)
	}
	if resp.JSON201 != nil {
		return resp.JSON201, nil
	}
	return nil, newStatusError("CreateDeck", resp.StatusCode())
}

// UpdateDeck は既存デッキを更新する。
func (c *Client) UpdateDeck(ctx context.Context, deckID int64, req apicard.DeckUpdateRequest) (*apicard.Deck, error) {
	resp, err := c.api.UpdateDeckWithResponse(ctx, deckID, apicard.UpdateDeckJSONRequestBody(req))
	if err != nil {
		return nil, fmt.Errorf("apicardclient: UpdateDeck: %w", err)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	return nil, newStatusError("UpdateDeck", resp.StatusCode())
}

// DeleteDeck はデッキを削除する。
func (c *Client) DeleteDeck(ctx context.Context, deckID int64) error {
	resp, err := c.api.DeleteDeckWithResponse(ctx, deckID)
	if err != nil {
		return fmt.Errorf("apicardclient: DeleteDeck: %w", err)
	}
	if resp.StatusCode() == http.StatusNoContent {
		return nil
	}
	return newStatusError("DeleteDeck", resp.StatusCode())
}

// ValidateDeckForBattle はデッキがバトルで使えるか検証する。
// ステータス 400 は契約上デッキの不正を意味するため、ErrDeckInvalid を返す。
func (c *Client) ValidateDeckForBattle(ctx context.Context, deckID int64) error {
	resp, err := c.api.ValidateDeckForBattleWithResponse(ctx, deckID)
	if err != nil {
		return fmt.Errorf("apicardclient: ValidateDeckForBattle: %w", err)
	}
	switch code := resp.StatusCode(); code {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("apicardclient: ValidateDeckForBattle: %w: %s", ErrDeckInvalid, string(resp.Body))
	default:
		return newStatusError("ValidateDeckForBattle", code)
	}
}

// newStatusError はステータスコードを、errors.Is で区別できるエラーに変換する。
func newStatusError(op string, code int) error {
	var sentinel error
	switch {
	case code == http.StatusUnauthorized:
		sentinel = ErrUnauthorized
	case code == http.StatusForbidden:
		sentinel = ErrForbidden
	case code == http.StatusNotFound:
		sentinel = ErrNotFound
	case code == http.StatusBadRequest:
		sentinel = ErrBadRequest
	case code >= http.StatusInternalServerError:
		sentinel = ErrInternalServer
	default:
		return fmt.Errorf("apicardclient: %s: %w: status %d", op, ErrUnexpectedResponse, code)
	}
	return fmt.Errorf("apicardclient: %s: %w", op, sentinel)
}
