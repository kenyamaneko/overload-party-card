// Package apicardclient は consumer から wire 詳細 (REST path / HTTP status code) を
// 隠蔽するため、生成 client を sentinel error 変換でラップする。
package apicardclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// Sentinel errors. wire 契約 (OpenAPI spec) で定義された status code の意味を sentinel として export する。
var (
	// ErrNotFound は status 404。
	ErrNotFound = errors.New("apicardclient: not found")

	// ErrUnauthorized は status 401。X-Internal-Auth 不正等。
	ErrUnauthorized = errors.New("apicardclient: unauthorized")

	// ErrForbidden は status 403。未所持カード等のアクセス権限エラー。
	ErrForbidden = errors.New("apicardclient: forbidden")

	// ErrBadRequest は status 400 の generic 形 (validate-for-battle を除く)。
	ErrBadRequest = errors.New("apicardclient: bad request")

	// ErrDeckInvalid は ValidateDeckForBattle の status 400。
	// wire 契約上「デッキバリデーション失敗 / 制限枚数超過」を意味する。
	ErrDeckInvalid = errors.New("apicardclient: deck invalid")

	// ErrInternalServer は status 5xx。
	ErrInternalServer = errors.New("apicardclient: internal server error")
)

// Client は overload-party-card REST API の sentinel-error converted client。
type Client struct {
	api *apicard.ClientWithResponses
}

// Option は Client 構築時の設定。
type Option func(*config)

type config struct {
	httpClient apicard.HttpRequestDoer
	editors    []apicard.RequestEditorFn
}

// WithHTTPClient は基底 HTTP doer を差し替える。デフォルトは http.DefaultClient。
func WithHTTPClient(doer apicard.HttpRequestDoer) Option {
	return func(c *config) { c.httpClient = doer }
}

// WithRequestEditorFn は全リクエストに適用する RequestEditor を追加する。
// X-Internal-Auth ヘッダ注入等に使用する。
func WithRequestEditorFn(fn apicard.RequestEditorFn) Option {
	return func(c *config) { c.editors = append(c.editors, fn) }
}

// New は baseURL に接続する Client を生成する。
func New(baseURL string, opts ...Option) (*Client, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	apiOpts := make([]apicard.ClientOption, 0, 1+len(cfg.editors))
	if cfg.httpClient != nil {
		apiOpts = append(apiOpts, apicard.WithHTTPClient(cfg.httpClient))
	}
	for _, ed := range cfg.editors {
		apiOpts = append(apiOpts, apicard.WithRequestEditorFn(ed))
	}

	api, err := apicard.NewClientWithResponses(baseURL, apiOpts...)
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

// ListCards は全カード定義 (master データ) を返す。
func (c *Client) ListCards(ctx context.Context) ([]apicard.CardDefinition, error) {
	resp, err := c.api.ListCardsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("apicardclient: ListCards: %w", err)
	}
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}
	return nil, newStatusError("ListCards", resp.StatusCode())
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

// GetDeck はデッキ詳細 (deck メタとカード構成) を返す。
func (c *Client) GetDeck(ctx context.Context, deckID int64) (*apicard.Deck, []apicard.DeckCard, error) {
	resp, err := c.api.GetDeckWithResponse(ctx, deckID)
	if err != nil {
		return nil, nil, fmt.Errorf("apicardclient: GetDeck: %w", err)
	}
	if resp.JSON200 != nil {
		return &resp.JSON200.Deck, resp.JSON200.Cards, nil
	}
	return nil, nil, newStatusError("GetDeck", resp.StatusCode())
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

// ValidateDeckForBattle はデッキがバトル使用可能か検証する。
// 400 は wire 契約上「デッキ不正」を意味するため ErrDeckInvalid を返す。
func (c *Client) ValidateDeckForBattle(ctx context.Context, deckID int64) error {
	resp, err := c.api.ValidateDeckForBattleWithResponse(ctx, deckID)
	if err != nil {
		return fmt.Errorf("apicardclient: ValidateDeckForBattle: %w", err)
	}
	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("apicardclient: ValidateDeckForBattle: %w: %s", ErrDeckInvalid, string(resp.Body))
	}
	return newStatusError("ValidateDeckForBattle", resp.StatusCode())
}

// newStatusError は HTTP status code を sentinel error (errors.Is 分岐可能) に変換する。
// validate-for-battle の 400 は呼出側で先に ErrDeckInvalid に変換するため、ここでは ErrBadRequest にフォールバックする。
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
		return fmt.Errorf("apicardclient: %s: unexpected status %d", op, code)
	}
	return fmt.Errorf("apicardclient: %s: %w", op, sentinel)
}
