package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/cache"
	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
)

// fullDeckCardCount is kept as a test-local literal rather than importing the production
// constants.DeckSize: importing it would let a business-rule change silently keep the fixture
// "valid" instead of surfacing as a test failure.
const fullDeckCardCount = 30

// deckFixture is a self-consistent set of deck-related master data and ownership: a faction, a
// product with a routine/special initiative pair, and card entries the player owns exactly.
type deckFixture struct {
	faction   string
	productID string
	routineID string
	specialID string
	entries   []apicard.DeckCardEntry
	owned     []*domain.PlayerCard
}

// newMinimalDeckFixture returns a fixture with a single owned card, valid for any check that
// does not require an exact deck-size total (CreateDeck/UpdateDeck do not require it).
func newMinimalDeckFixture() deckFixture {
	return deckFixture{
		faction:   "SHE",
		productID: "PD-TST01",
		routineID: "IN-TST01",
		specialID: "IN-TST02",
		entries:   []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
		owned:     []*domain.PlayerCard{{PlayerID: testPlayerID, CardID: "TST-0001", ArtNo: 1, Count: 1}},
	}
}

// newFullDeckFixture returns a fixture whose card entries total fullDeckCardCount, spread
// across distinct card IDs so no single card's unlimited-restriction cap is exceeded.
func newFullDeckFixture() deckFixture {
	fx := newMinimalDeckFixture()
	const cardKinds = 10
	const perCard = fullDeckCardCount / cardKinds
	entries := make([]apicard.DeckCardEntry, 0, cardKinds)
	owned := make([]*domain.PlayerCard, 0, cardKinds)
	for i := 1; i <= cardKinds; i++ {
		cardID := fmt.Sprintf("TST-%04d", i)
		entries = append(entries, apicard.DeckCardEntry{CardID: cardID, ArtNo: 1, Count: perCard})
		owned = append(owned, &domain.PlayerCard{PlayerID: testPlayerID, CardID: cardID, ArtNo: 1, Count: perCard})
	}
	fx.entries = entries
	fx.owned = owned
	return fx
}

// cardCache builds a cache.CardCache with one unlimited-restriction card per fixture entry.
func (fx deckFixture) cardCache() *cache.CardCache {
	cards := make([]*domain.Card, len(fx.entries))
	for i, e := range fx.entries {
		cards[i] = newTestCard(e.CardID, fx.faction, "Compute", "unlimited")
	}
	return newTestCardCache(cards...)
}

// productCache builds a cache.ProductCache with the fixture's product.
func (fx deckFixture) productCache(t *testing.T) *cache.ProductCache {
	t.Helper()
	return newTestProductCache(t, domain.Product{ProductID: fx.productID, Faction: fx.faction, ProductName: "テストプロダクト", IsActive: true})
}

// initiativeCache builds a cache.InitiativeCache with the fixture's routine/special initiatives.
func (fx deckFixture) initiativeCache(t *testing.T) *cache.InitiativeCache {
	t.Helper()
	return newTestInitiativeCache(t,
		domain.Initiative{InitiativeID: fx.routineID, ProductID: fx.productID, Kind: domain.InitiativeKindRoutine, Name: "ルーチン施策", InsightCost: 1, EffectText: "効果", Effect: json.RawMessage(`{}`), IsActive: true},
		domain.Initiative{InitiativeID: fx.specialID, ProductID: fx.productID, Kind: domain.InitiativeKindSpecial, Name: "スペシャル施策", InsightCost: 1, EffectText: "効果", Effect: json.RawMessage(`{}`), IsActive: true},
	)
}

// deckCards converts the fixture's request-shape entries into persisted domain.DeckCard rows.
func (fx deckFixture) deckCards(playerID string, deckID int64) []domain.DeckCard {
	cards := make([]domain.DeckCard, len(fx.entries))
	for i, e := range fx.entries {
		cards[i] = domain.DeckCard{PlayerID: playerID, DeckID: deckID, CardID: e.CardID, ArtNo: e.ArtNo, Count: e.Count}
	}
	return cards
}

// createRequest builds a DeckCreateRequest carrying the fixture's card entries.
func (fx deckFixture) createRequest() apicard.DeckCreateRequest {
	return apicard.DeckCreateRequest{
		DeckName:  "テストデッキ",
		Faction:   fx.faction,
		ProductID: fx.productID,
		RoutineID: fx.routineID,
		SpecialID: fx.specialID,
		Cards:     fx.entries,
	}
}

// updateRequest builds a DeckUpdateRequest carrying the fixture's card entries.
func (fx deckFixture) updateRequest() apicard.DeckUpdateRequest {
	return apicard.DeckUpdateRequest{
		DeckName:  "更新後テストデッキ",
		Faction:   fx.faction,
		ProductID: fx.productID,
		RoutineID: fx.routineID,
		SpecialID: fx.specialID,
		Cards:     fx.entries,
	}
}

// withDeckMasterData applies the fixture's card/product/initiative caches to newTestRouter.
func withDeckMasterData(t *testing.T, fx deckFixture) []testRouterOption {
	t.Helper()
	return []testRouterOption{
		withCardCache(fx.cardCache()),
		withProductCache(fx.productCache(t)),
		withInitiativeCache(fx.initiativeCache(t)),
	}
}

// fakePlayerCardRepoReturning is a fakePlayerCardRepo whose GetPlayerCards always returns cards.
func fakePlayerCardRepoReturning(cards []*domain.PlayerCard) *fakePlayerCardRepo {
	return &fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
		return cards, nil
	}}
}

// fakeFactionClientOwning is a fakeFactionClient whose ListPlayerFactions always returns factions.
func fakeFactionClientOwning(factions ...string) *fakeFactionClient {
	return &fakeFactionClient{ListPlayerFactionsFn: func(ctx context.Context, playerID string) ([]string, error) {
		return factions, nil
	}}
}

func TestDeckHandlerGetDecks(t *testing.T) {
	t.Run("[デッキAPI] デッキ一覧取得", func(t *testing.T) {
		t.Run("プレイヤーのデッキ一覧と所持カード一覧の両方が取得できるとき、200になりレスポンスボディはデッキの配列になる", func(t *testing.T) {
			deck := &domain.Deck{PlayerID: testPlayerID, DeckID: 1, DeckName: "デッキ1", Faction: "SHE", ProductID: "PD-TST01", RoutineID: "IN-TST01", SpecialID: "IN-TST02"}
			engine := newTestRouter(t,
				withDeckRepo(&fakeDeckRepo{
					FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) { return []*domain.Deck{deck}, nil },
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(nil)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.Deck
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			assert.Equal(t, deck.DeckName, body[0].DeckName)
		})

		errorCases := []struct {
			name     string
			deckRepo *fakeDeckRepo
			pcRepo   *fakePlayerCardRepo
		}{
			{
				name: "プレイヤーのデッキ一覧の取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる",
				deckRepo: &fakeDeckRepo{FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) {
					return nil, errors.New("find by player id failed")
				}},
				pcRepo: &fakePlayerCardRepo{},
			},
			{
				name: "デッキ一覧の取得は成功し、プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる",
				deckRepo: &fakeDeckRepo{FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) {
					return []*domain.Deck{{PlayerID: testPlayerID, DeckID: 1}}, nil
				}},
				pcRepo: &fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return nil, errors.New("get player cards failed")
				}},
			},
			{
				name: "あるデッキの構成カードの取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる",
				deckRepo: &fakeDeckRepo{
					FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) {
						return []*domain.Deck{{PlayerID: testPlayerID, DeckID: 1}}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, errors.New("get deck cards failed")
					},
				},
				pcRepo: fakePlayerCardRepoReturning(nil),
			},
		}
		for _, tc := range errorCases {
			t.Run(tc.name, func(t *testing.T) {
				engine := newTestRouter(t, withDeckRepo(tc.deckRepo), withPlayerCardRepo(tc.pcRepo))

				rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks", nil)

				require.Equal(t, http.StatusInternalServerError, rr.Code)
				assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
			})
		}

		t.Run("構成カードが1件もないデッキがあるとき、レスポンスのそのデッキの要素にはdeck_cardsキー自体が含まれない", func(t *testing.T) {
			engine := newTestRouter(t,
				withDeckRepo(&fakeDeckRepo{
					FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) {
						return []*domain.Deck{{PlayerID: testPlayerID, DeckID: 1, DeckName: "空デッキ"}}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(nil)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []map[string]any
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			_, hasKey := body[0]["deck_cards"]
			assert.False(t, hasKey, "deck_cards キーが含まれない")
		})

		t.Run("構成カードが1件以上あるデッキがあるとき、レスポンスのそのデッキの要素のdeck_cards配列にはそれらの構成カードが含まれる", func(t *testing.T) {
			deckCards := []domain.DeckCard{{PlayerID: testPlayerID, DeckID: 1, CardID: "TST-0001", ArtNo: 1, Count: 1}}
			engine := newTestRouter(t,
				withDeckRepo(&fakeDeckRepo{
					FindByPlayerIDFn: func(ctx context.Context, playerID string) ([]*domain.Deck, error) {
						return []*domain.Deck{{PlayerID: testPlayerID, DeckID: 1, DeckName: "デッキ1"}}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return deckCards, nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(nil)),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body []apicard.Deck
			decodeJSON(t, rr, &body)
			require.Len(t, body, 1)
			require.NotNil(t, body[0].DeckCards)
			require.Len(t, *body[0].DeckCards, 1)
			assert.Equal(t, "TST-0001", (*body[0].DeckCards)[0].CardID)
		})
	})
}

func TestDeckHandlerGetDeck(t *testing.T) {
	t.Run("[デッキAPI] デッキ詳細取得", func(t *testing.T) {
		t.Run("deckIdパスパラメータが数値として解釈できないとき、400になりボディのerrorフィールドはinvalid deck_idになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/abc", nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "invalid deck_id", decodeErrorMessage(t, rr))
		})

		t.Run("指定したdeckIdのデッキが存在しないとき、404になりボディのerrorフィールドはnot foundを含む文字列になる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{
				FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
					return nil, port.ErrNotFound
				},
			}))

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusNotFound, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "not found")
		})

		t.Run("デッキの取得で想定外のエラーが発生するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{
				FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
					return nil, errors.New("find by id failed")
				},
			}))

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキ本体の取得は成功し、構成カードの取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{
				FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
					return &domain.Deck{PlayerID: playerID, DeckID: deckID}, nil
				},
				GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
					return nil, errors.New("get deck cards failed")
				},
			}))

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキ本体と構成カードは取得できたが、プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t,
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, nil
					},
				}),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return nil, errors.New("get player cards failed")
				}}),
			)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("指定したdeckIdのデッキが存在し、その構成カードとプレイヤーの所持カードがすべて取得できるとき、200になりレスポンスボディ直下がデッキ本体になり、取得した構成カード一覧はdeck_cardsフィールドにのみ含まれる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, DeckName: "デッキ1", Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body map[string]any
			decodeJSON(t, rr, &body)
			assert.Equal(t, "デッキ1", body["deck_name"])
			_, hasCardsKey := body["cards"]
			assert.False(t, hasCardsKey, "cards キーは含まれない")
			deckCards, ok := body["deck_cards"].([]any)
			require.True(t, ok, "deck_cards キーが配列で含まれる")
			require.Len(t, deckCards, 1)
		})

		t.Run("指定したdeckIdのデッキが存在し、デッキ内容検証・施策整合検証をともに満たし、構成カードの合計枚数がデッキ規定枚数と一致するとき、is_validはtrueになる", func(t *testing.T) {
			fx := newFullDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body apicard.Deck
			decodeJSON(t, rr, &body)
			assert.True(t, body.IsValid)
		})

		t.Run("指定したdeckIdのデッキが存在し、デッキ内容検証・施策整合検証をともに満たすが、構成カードの合計枚数がデッキ規定枚数と一致しないとき、is_validはfalseになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodGet, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusOK, rr.Code)
			var body apicard.Deck
			decodeJSON(t, rr, &body)
			assert.False(t, body.IsValid)
		})
	})
}

func TestDeckHandlerCreateDeck(t *testing.T) {
	t.Run("[デッキAPI] デッキ作成", func(t *testing.T) {
		t.Run("リクエストボディが妥当なJSONとして解釈できないとき、400になりボディのerrorフィールドはmalformed request bodyになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", `{"cards":`)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "malformed request body", decodeErrorMessage(t, rr))
		})

		t.Run("宣言陣営が選択可能陣営のいずれでもないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			req := fx.createRequest()
			req.Faction = "NotASelectableFaction"
			engine := newTestRouter(t, withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)))

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("あるカードの制限区分の投入上限を超える枚数が指定されているとき、400になりボディのerrorフィールドはrestriction exceededを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			limitedCardID := "TST-0002"
			fx.entries = []apicard.DeckCardEntry{{CardID: limitedCardID, ArtNo: 1, Count: 2}}
			fx.owned = []*domain.PlayerCard{{PlayerID: testPlayerID, CardID: limitedCardID, ArtNo: 1, Count: 2}}
			engine := newTestRouter(t,
				withCardCache(newTestCardCache(newTestCard(limitedCardID, fx.faction, "Compute", "limited"))),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "restriction exceeded")
		})

		t.Run("プレイヤーの所持カード一覧に含まれないカードが指定されているとき、403になりボディのerrorフィールドはunownedを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withPlayerCardRepo(fakePlayerCardRepoReturning(nil)))

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusForbidden, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "unowned")
		})

		t.Run("プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
				return nil, errors.New("get player cards failed")
			}}))

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("宣言陣営がプレイヤーの所持陣営に含まれないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning("Tenki")),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("プレイヤーの所持陣営取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(&fakeFactionClient{ListPlayerFactionsFn: func(ctx context.Context, playerID string) ([]string, error) {
					return nil, errors.New("list player factions failed")
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキの作成に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{CreateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
					return 0, errors.New("create failed")
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキの作成は成功したが、作成後の構成カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{
					CreateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
						return 1, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, errors.New("get deck cards failed")
					},
				}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", fx.createRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("所持・陣営所持・施策整合の検証をすべて満たすとき、201になりレスポンスボディは作成されたデッキ(リクエストで指定した値と一致するデッキ)になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			req := fx.createRequest()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{
					CreateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) (int64, error) {
						return 42, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks", req)

			require.Equal(t, http.StatusCreated, rr.Code)
			var body apicard.Deck
			decodeJSON(t, rr, &body)
			assert.Equal(t, req.DeckName, body.DeckName)
			assert.Equal(t, req.Faction, body.Faction)
			assert.Equal(t, req.ProductID, body.ProductID)
			assert.Equal(t, req.RoutineID, body.RoutineID)
			assert.Equal(t, req.SpecialID, body.SpecialID)
			assert.Equal(t, req.PlaymatNo, body.PlaymatNo)
			assert.Equal(t, req.SleeveNo, body.SleeveNo)
		})
	})
}

func TestDeckHandlerUpdateDeck(t *testing.T) {
	t.Run("[デッキAPI] デッキ更新", func(t *testing.T) {
		t.Run("deckIdパスパラメータが数値として解釈できないとき、400になりボディのerrorフィールドはinvalid deck_idになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/abc", newMinimalDeckFixture().updateRequest())

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "invalid deck_id", decodeErrorMessage(t, rr))
		})

		t.Run("deckIdが数値で、リクエストボディが妥当なJSONとして解釈できないとき、400になりボディのerrorフィールドはmalformed request bodyになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", `{"cards":`)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "malformed request body", decodeErrorMessage(t, rr))
		})

		t.Run("宣言陣営が選択可能陣営のいずれでもないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			req := fx.updateRequest()
			req.Faction = "NotASelectableFaction"
			engine := newTestRouter(t, withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)))

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("あるカードの制限区分の投入上限を超える枚数が指定されているとき、400になりボディのerrorフィールドはrestriction exceededを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			limitedCardID := "TST-0002"
			fx.entries = []apicard.DeckCardEntry{{CardID: limitedCardID, ArtNo: 1, Count: 2}}
			fx.owned = []*domain.PlayerCard{{PlayerID: testPlayerID, CardID: limitedCardID, ArtNo: 1, Count: 2}}
			engine := newTestRouter(t,
				withCardCache(newTestCardCache(newTestCard(limitedCardID, fx.faction, "Compute", "limited"))),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "restriction exceeded")
		})

		t.Run("プレイヤーの所持カード一覧に含まれないカードが指定されているとき、403になりボディのerrorフィールドはunownedを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withPlayerCardRepo(fakePlayerCardRepoReturning(nil)))

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusForbidden, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "unowned")
		})

		t.Run("プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
				return nil, errors.New("get player cards failed")
			}}))

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("宣言陣営がプレイヤーの所持陣営に含まれないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning("Tenki")),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("プレイヤーの所持陣営取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(&fakeFactionClient{ListPlayerFactionsFn: func(ctx context.Context, playerID string) ([]string, error) {
					return nil, errors.New("list player factions failed")
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("更新対象のdeckIdのデッキが存在しないとき、404になりボディのerrorフィールドはnot foundを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{UpdateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
					return port.ErrNotFound
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusNotFound, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "not found")
		})

		t.Run("デッキの更新で想定外のエラーが発生するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{UpdateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error {
					return errors.New("update failed")
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキの更新は成功したが、更新後の構成カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{
					UpdateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error { return nil },
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return nil, errors.New("get deck cards failed")
					},
				}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", fx.updateRequest())

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("所持・陣営所持・施策整合の検証をすべて満たすとき、200になりレスポンスボディは更新後のデッキ(リクエストで指定した値と一致するデッキ)になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			req := fx.updateRequest()
			opts := append(withDeckMasterData(t, fx),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
				withFactionClient(fakeFactionClientOwning(fx.faction)),
				withDeckRepo(&fakeDeckRepo{
					UpdateFn: func(ctx context.Context, deck domain.Deck, entries []domain.DeckCardEntry) error { return nil },
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPut, "/api/v1/cards/decks/1", req)

			require.Equal(t, http.StatusOK, rr.Code)
			var body apicard.Deck
			decodeJSON(t, rr, &body)
			assert.Equal(t, req.DeckName, body.DeckName)
			assert.Equal(t, req.Faction, body.Faction)
			assert.Equal(t, req.ProductID, body.ProductID)
			assert.Equal(t, req.RoutineID, body.RoutineID)
			assert.Equal(t, req.SpecialID, body.SpecialID)
			assert.Equal(t, req.PlaymatNo, body.PlaymatNo)
			assert.Equal(t, req.SleeveNo, body.SleeveNo)
		})
	})
}

func TestDeckHandlerDeleteDeck(t *testing.T) {
	t.Run("[デッキAPI] デッキ削除", func(t *testing.T) {
		t.Run("deckIdパスパラメータが数値として解釈できないとき、400になりボディのerrorフィールドはinvalid deck_idになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodDelete, "/api/v1/cards/decks/abc", nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "invalid deck_id", decodeErrorMessage(t, rr))
		})

		t.Run("deckIdが数値でデッキの削除が成功するとき、204になりレスポンスボディは空になる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{DeleteFn: func(ctx context.Context, playerID string, deckID int64) error {
				return nil
			}}))

			rr := doAuthedRequest(t, engine, http.MethodDelete, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusNoContent, rr.Code)
			assert.Empty(t, rr.Body.String())
		})

		t.Run("削除対象のdeckIdのデッキが存在しないとき、404になりボディのerrorフィールドはnot foundを含む文字列になる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{DeleteFn: func(ctx context.Context, playerID string, deckID int64) error {
				return port.ErrNotFound
			}}))

			rr := doAuthedRequest(t, engine, http.MethodDelete, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusNotFound, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "not found")
		})

		t.Run("デッキの削除で想定外のエラーが発生するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{DeleteFn: func(ctx context.Context, playerID string, deckID int64) error {
				return errors.New("delete failed")
			}}))

			rr := doAuthedRequest(t, engine, http.MethodDelete, "/api/v1/cards/decks/1", nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})
	})
}

func TestDeckHandlerValidateDeckForBattle(t *testing.T) {
	const path = "/api/v1/cards/decks/1/validate-for-battle"

	t.Run("[デッキAPI] デッキのバトル可否検証", func(t *testing.T) {
		t.Run("deckIdパスパラメータが数値として解釈できないとき、400になりボディのerrorフィールドはinvalid deck_idになる", func(t *testing.T) {
			engine := newTestRouter(t)

			rr := doAuthedRequest(t, engine, http.MethodPost, "/api/v1/cards/decks/abc/validate-for-battle", nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Equal(t, "invalid deck_id", decodeErrorMessage(t, rr))
		})

		t.Run("deckIdが数値で、検証対象のデッキが存在しないとき、404になりボディのerrorフィールドはnot foundを含む文字列になる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
				return nil, port.ErrNotFound
			}}))

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusNotFound, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "not found")
		})

		t.Run("deckIdが数値で、デッキの取得中に想定外のエラーが発生するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
				return nil, errors.New("find by id failed")
			}}))

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("デッキ本体の取得は成功し、構成カードの取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{
				FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
					return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
				},
				GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
					return nil, errors.New("get deck cards failed")
				},
			}))

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("構成カードの合計枚数がデッキ規定枚数と一致しないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newMinimalDeckFixture()
			engine := newTestRouter(t, withDeckRepo(&fakeDeckRepo{
				FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
					return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
				},
				GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
					return fx.deckCards(playerID, deckID), nil
				},
			}))

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("構成カードの合計枚数がデッキ規定枚数と一致し、プレイヤーの所持カード取得に失敗するとき、500になりボディのerrorフィールドはinternal server errorになる", func(t *testing.T) {
			fx := newFullDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(&fakePlayerCardRepo{GetPlayerCardsFn: func(ctx context.Context, playerID string) ([]*domain.PlayerCard, error) {
					return nil, errors.New("get player cards failed")
				}}),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Equal(t, "internal server error", decodeErrorMessage(t, rr))
		})

		t.Run("構成カードの合計枚数がデッキ規定枚数と一致し、カード構成にプレイヤーの所持カード一覧に含まれないカードがあるとき、403になりボディのerrorフィールドはunownedを含む文字列になる", func(t *testing.T) {
			fx := newFullDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(nil)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusForbidden, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "unowned")
		})

		t.Run("構成カードの合計枚数がデッキ規定枚数と一致し、あるカードの制限区分の投入上限を超える枚数が含まれるとき、400になりボディのerrorフィールドはrestriction exceededを含む文字列になる", func(t *testing.T) {
			fx := newFullDeckFixture()
			limitedCardID := fx.entries[0].CardID
			cardCache := fx.cardCache()
			cardCache.InjectForTest(limitedCardID, newTestCard(limitedCardID, fx.faction, "Compute", "limited"))
			opts := []testRouterOption{
				withCardCache(cardCache),
				withProductCache(fx.productCache(t)),
				withInitiativeCache(fx.initiativeCache(t)),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			}
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "restriction exceeded")
		})

		t.Run("構成カードの合計枚数がデッキ規定枚数と一致し、選択プロダクト・ルーチン施策・スペシャル施策の組み合わせが整合しないとき、400になりボディのerrorフィールドはinvalid deckを含む文字列になる", func(t *testing.T) {
			fx := newFullDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: "IN-NOT-REGISTERED", SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, decodeErrorMessage(t, rr), "invalid deck")
		})

		t.Run("デッキ内容検証・施策整合検証をともに満たし、構成カードの合計枚数がデッキ規定枚数と一致するとき、200になりレスポンスボディは空になる", func(t *testing.T) {
			fx := newFullDeckFixture()
			opts := append(withDeckMasterData(t, fx),
				withDeckRepo(&fakeDeckRepo{
					FindByIDFn: func(ctx context.Context, playerID string, deckID int64) (*domain.Deck, error) {
						return &domain.Deck{PlayerID: playerID, DeckID: deckID, Faction: fx.faction, ProductID: fx.productID, RoutineID: fx.routineID, SpecialID: fx.specialID}, nil
					},
					GetDeckCardsFn: func(ctx context.Context, playerID string, deckID int64) ([]domain.DeckCard, error) {
						return fx.deckCards(playerID, deckID), nil
					},
				}),
				withPlayerCardRepo(fakePlayerCardRepoReturning(fx.owned)),
			)
			engine := newTestRouter(t, opts...)

			rr := doAuthedRequest(t, engine, http.MethodPost, path, nil)

			require.Equal(t, http.StatusOK, rr.Code)
			assert.Empty(t, rr.Body.String())
		})
	})
}
