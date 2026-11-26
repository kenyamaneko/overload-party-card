package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func TestDeckCreation(t *testing.T) {
	t.Run("[デッキ]デッキ作成", func(t *testing.T) {
		t.Run("デッキを作成する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("デッキを持たないプレイヤーAがデッキを作成すると、プレイヤーAのデッキ一覧が、作成したデッキ1件だけになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())

					created := h.CreateDeck(t, usecasetest.NewDraft())

					details, err := h.Interactor.GetDecks(t.Context(), usecasetest.PlayerA)
					assert.NoError(t, err)
					assert.Len(t, details, 1)
					assert.Equal(t, created.Deck.DeckID, details[0].Deck.DeckID)
				})

				t.Run("時計が2026-01-02T03:04:05Zを返すとき、デッキを作成すると", func(t *testing.T) {
					fixedNow := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

					t.Run("応答の作成日時が2026-01-02T03:04:05Zになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						h.Clock.CurrentTime = fixedNow

						created := h.CreateDeck(t, usecasetest.NewDraft())

						assert.WithinDuration(t, fixedNow, created.Deck.CreatedAt, 0)
					})

					t.Run("応答の更新日時が2026-01-02T03:04:05Zになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						h.Clock.CurrentTime = fixedNow

						created := h.CreateDeck(t, usecasetest.NewDraft())

						assert.WithinDuration(t, fixedNow, created.Deck.UpdatedAt, 0)
					})
				})

				t.Run("カードの合計枚数ごとのis_valid", func(t *testing.T) {
					t.Run("合計が29枚のとき、応答のis_validはfalseになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(29)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))

						created := h.CreateDeck(t, usecasetest.NewDraft(entries...))

						assert.False(t, created.IsValid)
					})

					t.Run("合計が30枚のとき、応答のis_validはtrueになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(30)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))

						created := h.CreateDeck(t, usecasetest.NewDraft(entries...))

						assert.True(t, created.IsValid)
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("プレイヤーがTenkiだけを所持しているとき、宣言陣営にSHEを指定すると、陣営を所持していないことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					setup := usecasetest.NewDefaultDeckSetup()
					setup.OwnedFactions = []string{"Tenki"}
					h := usecasetest.NewDeckHarness(t, setup)

					_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft())

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "is not owned by player")
				})

				t.Run("デッキを持たないプレイヤーが作成に失敗したとき", func(t *testing.T) {
					tests := []struct {
						name          string
						ownedFactions []string
						total         int
					}{
						{name: "合計が31枚のとき、プレイヤーのデッキ一覧が0件になる", ownedFactions: []string{"SHE"}, total: 31},
						{name: "プレイヤーがTenkiだけを所持していて、宣言陣営にSHEを指定したとき、プレイヤーのデッキ一覧が0件になる", ownedFactions: []string{"Tenki"}, total: 0},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := usecasetest.BuildDeckCardEntries(tt.total)
							setup := usecasetest.NewDeckSetupOwning(entries)
							setup.OwnedFactions = tt.ownedFactions
							h := usecasetest.NewDeckHarness(t, setup)

							_, createErr := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(entries...))
							details, listErr := h.Interactor.GetDecks(t.Context(), usecasetest.PlayerA)

							assert.Error(t, createErr)
							assert.NoError(t, listErr)
							assert.Empty(t, details)
						})
					}
				})

				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *usecasetest.DeckHarness, failure error)
					}{
						{
							name:    "所持陣営のaccountサービスが「作成時の陣営照会の失敗」のエラーを返したとき、エラーの原因が「作成時の陣営照会の失敗」のエラーになる",
							failure: errors.New("作成時の陣営照会の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.Factions.Err = failure },
						},
						{
							name:    "所持カードのデータベースが「作成時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「作成時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("作成時の所持カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.PlayerCards.GetErr = failure },
						},
						{
							name:    "デッキのデータベースが保存時に「作成時のデッキ保存の失敗」のエラーを返したとき、エラーの原因が「作成時のデッキ保存の失敗」のエラーになる",
							failure: errors.New("作成時のデッキ保存の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.CreateErr = failure },
						},
						{
							name:    "デッキのデータベースが保存後の構成カード取得時に「作成時の構成カード取得の失敗」のエラーを返したとき、エラーの原因が「作成時の構成カード取得の失敗」のエラーになる",
							failure: errors.New("作成時の構成カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.GetDeckCardsErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 1)}
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							tt.inject(h, tt.failure)

							_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(entries...))

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}
