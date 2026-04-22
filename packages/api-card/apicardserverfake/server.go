// Package apicardserverfake は card サービスの HTTP 契約を実装する
// httptest.Server ラッパー。consumer (gateway 等) が cardclient を使う
// handler テストで、実 card サービスを起動せずに REST 呼び出しを検証するための
// テストダブルを提供する。
//
// 各 endpoint は Fn field (func callback) で status + response body を制御する。
// Fn が nil の endpoint は既定値を返す (happy-path を仮定した最低限の応答)。
//
// JSON request / response shape は cardclient が送受する形式に合わせる。
// cardclient が private に保持している deckWithCards wrapper は本パッケージで
// 独立に定義しなおし (DeckWithCardsResponse)、テスト側が typed で組み立てられる
// ようにする。
package apicardserverfake

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// DeckWithCardsResponse は GetDeck endpoint の JSON envelope。
// cardclient 側の private 型と shape 一致している必要があるため、本パッケージで
// export しテストが typed に組み立てられるようにする。
type DeckWithCardsResponse struct {
	Deck  *apicard.Deck      `json:"deck"`
	Cards []apicard.DeckCard `json:"cards"`
}

// Server は card HTTP 契約を実装する httptest.Server wrapper。
type Server struct {
	mu  sync.Mutex
	srv *httptest.Server

	// ListAllCardsFn は GET /internal/v1/cards の応答を決定する。nil なら 200 + 空配列。
	ListAllCardsFn func() (int, any)

	// ListCardsWithOwnershipFn は GET /internal/v1/players/{playerID}/cards/with-ownership の応答を決定する。
	// nil なら 200 + 空配列。
	ListCardsWithOwnershipFn func(playerID string) (int, any)

	// ListPlayerCardsFn は GET /internal/v1/players/{playerID}/cards の応答を決定する。
	// nil なら 200 + 空配列。
	ListPlayerCardsFn func(playerID string) (int, any)

	// ListDecksFn は GET /internal/v1/players/{playerID}/decks の応答を決定する。
	// nil なら 200 + 空配列。
	ListDecksFn func(playerID string) (int, any)

	// GetDeckFn は GET /internal/v1/players/{playerID}/decks/{deckID} の応答を決定する。
	// nil なら 200 + 空の DeckWithCardsResponse。
	GetDeckFn func(playerID, deckID string) (int, any)

	// CreateDeckFn は POST /internal/v1/players/{playerID}/decks の応答を決定する。
	// nil なら 200 + 空の Deck を返す。
	CreateDeckFn func(playerID string, req apicard.DeckCreateRequest) (int, any)

	// UpdateDeckFn は PUT /internal/v1/players/{playerID}/decks/{deckID} の応答を決定する。
	// nil なら 200 + 空の Deck を返す。
	UpdateDeckFn func(playerID, deckID string, req apicard.DeckUpdateRequest) (int, any)

	// DeleteDeckFn は DELETE /internal/v1/players/{playerID}/decks/{deckID} の応答を決定する。
	// nil なら 204 No Content を返す。
	DeleteDeckFn func(playerID, deckID string) (int, any)

	// ValidateDeckForBattleFn は POST /internal/v1/players/{playerID}/decks/{deckID}/validate-for-battle の応答を決定する。
	// nil なら 200 OK を返す (バトル可用を示す)。
	ValidateDeckForBattleFn func(playerID, deckID string) (int, any)
}

// NewServer は起動済み Server を返す。テスト終了時に Close() すること。
func NewServer() *Server {
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/v1/cards", s.handleListAllCards)
	mux.HandleFunc("GET /internal/v1/players/{playerID}/cards/with-ownership", s.handleListCardsWithOwnership)
	mux.HandleFunc("GET /internal/v1/players/{playerID}/cards", s.handleListPlayerCards)
	mux.HandleFunc("GET /internal/v1/players/{playerID}/decks", s.handleListDecks)
	mux.HandleFunc("GET /internal/v1/players/{playerID}/decks/{deckID}", s.handleGetDeck)
	mux.HandleFunc("POST /internal/v1/players/{playerID}/decks", s.handleCreateDeck)
	mux.HandleFunc("PUT /internal/v1/players/{playerID}/decks/{deckID}", s.handleUpdateDeck)
	mux.HandleFunc("DELETE /internal/v1/players/{playerID}/decks/{deckID}", s.handleDeleteDeck)
	mux.HandleFunc("POST /internal/v1/players/{playerID}/decks/{deckID}/validate-for-battle", s.handleValidateDeckForBattle)
	s.srv = httptest.NewServer(mux)
	return s
}

// URL は httptest.Server のベース URL を返す。
func (s *Server) URL() string { return s.srv.URL }

// Close は内部 httptest.Server を閉じる。
func (s *Server) Close() { s.srv.Close() }

func (s *Server) handleListAllCards(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fn := s.ListAllCardsFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.CardDefinition{})
		return
	}
	status, body := fn()
	writeJSON(w, status, body)
}

func (s *Server) handleListCardsWithOwnership(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.ListCardsWithOwnershipFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.CardWithOwnership{})
		return
	}
	status, body := fn(playerID)
	writeJSON(w, status, body)
}

func (s *Server) handleListPlayerCards(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.ListPlayerCardsFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.PlayerCardWithDef{})
		return
	}
	status, body := fn(playerID)
	writeJSON(w, status, body)
}

func (s *Server) handleListDecks(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.ListDecksFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.Deck{})
		return
	}
	status, body := fn(playerID)
	writeJSON(w, status, body)
}

func (s *Server) handleGetDeck(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.GetDeckFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	deckID := r.PathValue("deckID")
	if fn == nil {
		writeJSON(w, http.StatusOK, DeckWithCardsResponse{Cards: []apicard.DeckCard{}})
		return
	}
	status, body := fn(playerID, deckID)
	writeJSON(w, status, body)
}

func (s *Server) handleCreateDeck(w http.ResponseWriter, r *http.Request) {
	var req apicard.DeckCreateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	fn := s.CreateDeckFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.Deck{})
		return
	}
	status, body := fn(playerID, req)
	writeJSON(w, status, body)
}

func (s *Server) handleUpdateDeck(w http.ResponseWriter, r *http.Request) {
	var req apicard.DeckUpdateRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.Lock()
	fn := s.UpdateDeckFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	deckID := r.PathValue("deckID")
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.Deck{})
		return
	}
	status, body := fn(playerID, deckID, req)
	writeJSON(w, status, body)
}

func (s *Server) handleDeleteDeck(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.DeleteDeckFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	deckID := r.PathValue("deckID")
	if fn == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	status, body := fn(playerID, deckID)
	writeJSON(w, status, body)
}

func (s *Server) handleValidateDeckForBattle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.ValidateDeckForBattleFn
	s.mu.Unlock()
	playerID := r.PathValue("playerID")
	deckID := r.PathValue("deckID")
	if fn == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	status, body := fn(playerID, deckID)
	writeJSON(w, status, body)
}

// writeJSON は status code を書き、body が非 nil なら Content-Type: application/json
// で JSON encode して送る。body が nil の場合は body 無しでレスポンスを終わる
// (cardclient は 4xx/5xx のエラー body を一部しか読まないため、body=nil でも応答は成立する)。
func writeJSON(w http.ResponseWriter, status int, body any) {
	if body == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
