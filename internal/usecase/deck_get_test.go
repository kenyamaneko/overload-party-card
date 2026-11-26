package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func TestDeckRetrieval(t *testing.T) {
	t.Run("[デッキ]デッキ取得", func(t *testing.T) {
		t.Run("デッキを取得する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("is_validの判定", func(t *testing.T) {
					t.Run("宣言陣営がSHEで、SHE陣営のunlimitedのカードTST-0001からTST-0010のアート1を各3枚、合計30枚入れたデッキがあり、10種とも3枚ずつ所持しているとき、is_validはtrueになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(30)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
						deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)

						fetched, err := h.Interactor.GetDeck(t.Context(), usecasetest.PlayerA, deckID)

						assert.NoError(t, err)
						assert.True(t, fetched.IsValid)
					})

					t.Run("デッキ構成ルールを満たさないとき", func(t *testing.T) {
						tests := []struct {
							name    string
							setup   usecasetest.DeckSetup
							entries []domain.DeckCardEntry
						}{
							{
								name:    "合計が29枚のとき、is_validはfalseになる",
								setup:   usecasetest.NewDeckSetupOwning(usecasetest.BuildDeckCardEntries(29)),
								entries: usecasetest.BuildDeckCardEntries(29),
							},
							{
								name:    "合計が31枚のとき、is_validはfalseになる",
								setup:   usecasetest.NewDeckSetupOwning(usecasetest.BuildDeckCardEntries(31)),
								entries: usecasetest.BuildDeckCardEntries(31),
							},
							{
								name:    "デッキにTST-0001のアート1が3枚入っていて、保存後にTST-0001のアート1の所持枚数が3枚から2枚に減っているとき、is_validはfalseになる",
								setup:   usecasetest.NewDeckSetupWithReducedOwnership(),
								entries: usecasetest.BuildDeckCardEntries(30),
							},
							{
								name:    "デッキのプロダクトがTST-0001で、保存後にプロダクトTST-0001が存在しなくなったとき、is_validはfalseになる",
								setup:   usecasetest.NewDeckSetupWithoutProduct(),
								entries: usecasetest.BuildDeckCardEntries(30),
							},
						}
						for _, tt := range tests {
							t.Run(tt.name, func(t *testing.T) {
								h := usecasetest.NewDeckHarness(t, tt.setup)
								deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), tt.entries)

								fetched, err := h.Interactor.GetDeck(t.Context(), usecasetest.PlayerA, deckID)

								assert.NoError(t, err)
								assert.False(t, fetched.IsValid)
							})
						}
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("指定したdeck_idのデッキが存在しないとき、見つからないことを示すエラーになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())

					_, err := h.Interactor.GetDeck(t.Context(), usecasetest.PlayerA, usecasetest.NonexistentDeckID)

					assert.ErrorIs(t, err, port.ErrNotFound)
				})

				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *usecasetest.DeckHarness, failure error)
					}{
						{
							name:    "デッキのデータベースがデッキ取得時に「取得時のデッキ取得の失敗」のエラーを返したとき、エラーの原因が「取得時のデッキ取得の失敗」のエラーになる",
							failure: errors.New("取得時のデッキ取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.FindByIDErr = failure },
						},
						{
							name:    "デッキのデータベースが構成カード取得時に「取得時の構成カード取得の失敗」のエラーを返したとき、エラーの原因が「取得時の構成カード取得の失敗」のエラーになる",
							failure: errors.New("取得時の構成カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.GetDeckCardsErr = failure },
						},
						{
							name:    "所持カードのデータベースが「取得時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「取得時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("取得時の所持カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.PlayerCards.GetErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := usecasetest.BuildDeckCardEntries(30)
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)
							tt.inject(h, tt.failure)

							_, err := h.Interactor.GetDeck(t.Context(), usecasetest.PlayerA, deckID)

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}

func TestDeckListRetrieval(t *testing.T) {
	t.Run("[デッキ]デッキ一覧取得", func(t *testing.T) {
		t.Run("プレイヤーのデッキ一覧を取得する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("デッキが0件のとき、空の一覧が返る", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())

					details, err := h.Interactor.GetDecks(t.Context(), usecasetest.PlayerA)

					assert.NoError(t, err)
					assert.NotNil(t, details)
					assert.Empty(t, details)
				})

				t.Run("is_validの判定", func(t *testing.T) {
					t.Run("合計が30枚のデッキと合計が29枚のデッキがあるとき", func(t *testing.T) {
						t.Run("合計が30枚のデッキのis_validはtrueになる", func(t *testing.T) {
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(usecasetest.BuildDeckCardEntries(30)))
							thirtyDeckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(30))
							h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(29))

							assert.True(t, h.FindListedDeck(t, thirtyDeckID).IsValid)
						})

						t.Run("合計が29枚のデッキのis_validはfalseになる", func(t *testing.T) {
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(usecasetest.BuildDeckCardEntries(30)))
							h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(30))
							twentyNineDeckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(29))

							assert.False(t, h.FindListedDeck(t, twentyNineDeckID).IsValid)
						})
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *usecasetest.DeckHarness, failure error)
					}{
						{
							name:    "デッキのデータベースが一覧取得時に「一覧取得時のデッキ一覧取得の失敗」のエラーを返したとき、エラーの原因が「一覧取得時のデッキ一覧取得の失敗」のエラーになる",
							failure: errors.New("一覧取得時のデッキ一覧取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.FindByPlayerIDErr = failure },
						},
						{
							name:    "所持カードのデータベースが「一覧取得時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「一覧取得時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("一覧取得時の所持カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.PlayerCards.GetErr = failure },
						},
						{
							name:    "デッキのデータベースがいずれかのデッキの構成カード取得時に「一覧取得時の構成カード取得の失敗」のエラーを返したとき、エラーの原因が「一覧取得時の構成カード取得の失敗」のエラーになる",
							failure: errors.New("一覧取得時の構成カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.GetDeckCardsErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := usecasetest.BuildDeckCardEntries(30)
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)
							tt.inject(h, tt.failure)

							_, err := h.Interactor.GetDecks(t.Context(), usecasetest.PlayerA)

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}

func TestDeckValidationBeforeBattle(t *testing.T) {
	t.Run("[デッキ]バトル前検証", func(t *testing.T) {
		t.Run("デッキをバトルに使えるか検証する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("宣言陣営がSHEで、SHE陣営のunlimitedのカードTST-0001からTST-0010のアート1を各3枚、合計30枚入れたデッキがあり、10種とも3枚ずつ所持しているとき、エラーにならない", func(t *testing.T) {
					entries := usecasetest.BuildDeckCardEntries(30)
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
					deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)

					err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, deckID)

					assert.NoError(t, err)
				})

				t.Run("保存後に構成を満たさなくなったデッキ", func(t *testing.T) {
					t.Run("カードの合計枚数", func(t *testing.T) {
						tests := []struct {
							name  string
							total int
						}{
							{name: "合計が29枚のとき、枚数の不一致を示すデッキ構成の不正のエラーになる", total: 29},
							{name: "合計が31枚のとき、枚数の不一致を示すデッキ構成の不正のエラーになる", total: 31},
						}
						for _, tt := range tests {
							t.Run(tt.name, func(t *testing.T) {
								entries := usecasetest.BuildDeckCardEntries(tt.total)
								h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
								deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)

								err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, deckID)

								assert.ErrorIs(t, err, port.ErrInvalidDeck)
								usecasetest.AssertErrorContainsAll(t, err, "need exactly")
							})
						}
					})

					t.Run("デッキにTST-0001のアート1が3枚入っていて、保存後にTST-0001のアート1の所持枚数が3枚から2枚に減っているとき、所持枚数の不足を示すエラーになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupWithReducedOwnership())
						deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(30))

						err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, deckID)

						assert.ErrorIs(t, err, port.ErrUnowned)
						usecasetest.AssertErrorContainsAll(t, err, "not enough owned")
					})

					t.Run("デッキのプロダクトがTST-0001で、保存後にプロダクトTST-0001が存在しなくなったとき、プロダクトが無いことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupWithoutProduct())
						deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), usecasetest.BuildDeckCardEntries(30))

						err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, deckID)

						assert.ErrorIs(t, err, port.ErrInvalidDeck)
						usecasetest.AssertErrorContainsAll(t, err, "TST-0001", "not found")
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("指定したdeck_idのデッキが存在しないとき、見つからないことを示すエラーになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())

					err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, usecasetest.NonexistentDeckID)

					assert.ErrorIs(t, err, port.ErrNotFound)
				})

				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *usecasetest.DeckHarness, failure error)
					}{
						{
							name:    "デッキのデータベースがデッキ取得時に「バトル前検証時のデッキ取得の失敗」のエラーを返したとき、エラーの原因が「バトル前検証時のデッキ取得の失敗」のエラーになる",
							failure: errors.New("バトル前検証時のデッキ取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.FindByIDErr = failure },
						},
						{
							name:    "デッキのデータベースが構成カード取得時に「バトル前検証時の構成カード取得の失敗」のエラーを返したとき、エラーの原因が「バトル前検証時の構成カード取得の失敗」のエラーになる",
							failure: errors.New("バトル前検証時の構成カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.GetDeckCardsErr = failure },
						},
						{
							name:    "所持カードのデータベースが「バトル前検証時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「バトル前検証時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("バトル前検証時の所持カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.PlayerCards.GetErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := usecasetest.BuildDeckCardEntries(30)
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), entries)
							tt.inject(h, tt.failure)

							err := h.Interactor.ValidateDeckForBattle(t.Context(), usecasetest.PlayerA, deckID)

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}
