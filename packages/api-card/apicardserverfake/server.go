// Package apicardserverfake は card サービスの HTTP 契約を実装するテスト用サーバーを提供する。
// 利用側 (gateway など) が apicardclient を使うハンドラのテストで、実際の card サービスを起動せずに
// REST 呼び出しを確かめるために使う。
//
// 各エンドポイントの応答 (ステータスと本文) は、対応する Fn フィールドに設定した関数で決める。
// Fn が nil のエンドポイントは、成功を表す最小限の既定の応答を返す。
//
// 要求と応答の JSON の形は apicardclient が送受信する形式に合わせる。
package apicardserverfake

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// Server は card サービスの HTTP 契約を実装するテスト用サーバーです。
type Server struct {
	mu                sync.Mutex
	srv               *httptest.Server
	lastRequestHeader http.Header

	// GetHealthFn は GET /health の応答を決定する。
	GetHealthFn func() (int, any)

	// ListCardsWithOwnershipFn は GET /api/v1/cards/cards/with-ownership の応答を決定する。
	ListCardsWithOwnershipFn func() (int, any)

	// ListPlayerCardsFn は GET /api/v1/cards/cards の応答を決定する。
	ListPlayerCardsFn func() (int, any)

	// ListDecksFn は GET /api/v1/cards/decks の応答を決定する。
	ListDecksFn func() (int, any)

	// GetDeckFn は GET /api/v1/cards/decks/{deckID} の応答を決定する。
	GetDeckFn func(deckID string) (int, any)

	// CreateDeckFn は POST /api/v1/cards/decks の応答を決定する。
	CreateDeckFn func(req apicard.DeckCreateRequest) (int, any)

	// UpdateDeckFn は PUT /api/v1/cards/decks/{deckID} の応答を決定する。
	UpdateDeckFn func(deckID string, req apicard.DeckUpdateRequest) (int, any)

	// DeleteDeckFn は DELETE /api/v1/cards/decks/{deckID} の応答を決定する。
	DeleteDeckFn func(deckID string) (int, any)

	// ValidateDeckForBattleFn は POST /api/v1/cards/decks/{deckID}/validate-for-battle の応答を決定する。
	ValidateDeckForBattleFn func(deckID string) (int, any)
}

// NewServer は起動済みの Server を返す。テストの終了時に Close を呼ぶこと。
func NewServer() *Server {
	s := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleGetHealth)
	mux.HandleFunc("GET /api/v1/cards/cards/with-ownership", s.handleListCardsWithOwnership)
	mux.HandleFunc("GET /api/v1/cards/cards", s.handleListPlayerCards)
	mux.HandleFunc("GET /api/v1/cards/decks", s.handleListDecks)
	mux.HandleFunc("GET /api/v1/cards/decks/{deckID}", s.handleGetDeck)
	mux.HandleFunc("POST /api/v1/cards/decks", s.handleCreateDeck)
	mux.HandleFunc("PUT /api/v1/cards/decks/{deckID}", s.handleUpdateDeck)
	mux.HandleFunc("DELETE /api/v1/cards/decks/{deckID}", s.handleDeleteDeck)
	mux.HandleFunc("POST /api/v1/cards/decks/{deckID}/validate-for-battle", s.handleValidateDeckForBattle)
	s.srv = httptest.NewServer(s.recordRequestHeader(mux))
	return s
}

// URL は httptest.Server のベース URL を返す。
func (s *Server) URL() string { return s.srv.URL }

// Close は内部 httptest.Server を閉じる。
func (s *Server) Close() { s.srv.Close() }

// LastRequestHeader は直近に受信したリクエストのヘッダを返す。
// RequestEditorFn が実際に送信されたリクエストへ反映されたかを確認するために使う。
func (s *Server) LastRequestHeader() http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastRequestHeader
}

func (s *Server) recordRequestHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.lastRequestHeader = r.Header.Clone()
		s.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleGetHealth(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fn := s.GetHealthFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.HealthResponse{Status: "ok"})
		return
	}
	status, body := fn()
	writeJSON(w, status, body)
}

func (s *Server) handleListCardsWithOwnership(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fn := s.ListCardsWithOwnershipFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.CardWithOwnership{})
		return
	}
	status, body := fn()
	writeJSON(w, status, body)
}

func (s *Server) handleListPlayerCards(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fn := s.ListPlayerCardsFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.PlayerCardWithDef{})
		return
	}
	status, body := fn()
	writeJSON(w, status, body)
}

func (s *Server) handleListDecks(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	fn := s.ListDecksFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, []*apicard.Deck{})
		return
	}
	status, body := fn()
	writeJSON(w, status, body)
}

func (s *Server) handleGetDeck(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.GetDeckFn
	s.mu.Unlock()
	deckID := r.PathValue("deckID")
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.Deck{})
		return
	}
	status, body := fn(deckID)
	writeJSON(w, status, body)
}

func (s *Server) handleCreateDeck(w http.ResponseWriter, r *http.Request) {
	var req apicard.DeckCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode request body: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	fn := s.CreateDeckFn
	s.mu.Unlock()
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.Deck{})
		return
	}
	status, body := fn(req)
	writeJSON(w, status, body)
}

func (s *Server) handleUpdateDeck(w http.ResponseWriter, r *http.Request) {
	var req apicard.DeckUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode request body: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	fn := s.UpdateDeckFn
	s.mu.Unlock()
	deckID := r.PathValue("deckID")
	if fn == nil {
		writeJSON(w, http.StatusOK, apicard.Deck{})
		return
	}
	status, body := fn(deckID, req)
	writeJSON(w, status, body)
}

func (s *Server) handleDeleteDeck(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.DeleteDeckFn
	s.mu.Unlock()
	deckID := r.PathValue("deckID")
	if fn == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	status, body := fn(deckID)
	writeJSON(w, status, body)
}

func (s *Server) handleValidateDeckForBattle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	fn := s.ValidateDeckForBattleFn
	s.mu.Unlock()
	deckID := r.PathValue("deckID")
	if fn == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	status, body := fn(deckID)
	writeJSON(w, status, body)
}

// writeJSON はステータスを返し、body があれば JSON にして本文として返す。
// body が nil のときは本文を付けない。body を JSON にできないときは、500 で原因を返す。
func writeJSON(w http.ResponseWriter, status int, body any) {
	if body == nil {
		w.WriteHeader(status)
		return
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("encode response body: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(encoded); err != nil {
		slog.Error("apicardserverfake: write response body failed", "error", err)
	}
}
