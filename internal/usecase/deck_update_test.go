package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

const (
	firstTenkiCardNumber     = 2
	firstBulkTenkiCardNumber = 3
)

type failedUpdateScenario struct {
	setup usecasetest.DeckSetup
	draft domain.DeckDraft
}

func newFailedUpdateScenarioWithTotalOfThirtyOne() failedUpdateScenario {
	entries := []domain.DeckCardEntry{usecasetest.NewDeckCardEntry(usecasetest.FormatCardID(firstTenkiCardNumber), 1, 1)}
	for number := firstBulkTenkiCardNumber; number <= usecasetest.CatalogCardCount; number++ {
		entries = append(entries, usecasetest.NewDeckCardEntry(usecasetest.FormatCardID(number), 1, usecasetest.DeckCopiesPerCard))
	}
	setup := usecasetest.NewDefaultDeckSetup()
	setup.Cards = []*domain.Card{usecasetest.NewTestCard("TST-0001", usecasetest.FactionSHE, usecasetest.RestrictionUnlimited)}
	for number := firstTenkiCardNumber; number <= usecasetest.CatalogCardCount; number++ {
		setup.Cards = append(setup.Cards, usecasetest.NewTestCard(usecasetest.FormatCardID(number), usecasetest.FactionTenki, usecasetest.RestrictionUnlimited))
	}
	setup.OwnedCards = append(usecasetest.ToPlayerCards(entries), usecasetest.NewOwnedCard("TST-0001", 1, 2))
	setup.OwnedFactions = []string{usecasetest.FactionSHE, usecasetest.FactionTenki}
	return failedUpdateScenario{setup: setup, draft: newTenkiDraft(entries...)}
}

func newFailedUpdateScenarioWithoutTenki() failedUpdateScenario {
	entries := []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0002", 1, 1)}
	setup := usecasetest.NewDefaultDeckSetup()
	setup.Cards = []*domain.Card{
		usecasetest.NewTestCard("TST-0001", usecasetest.FactionSHE, usecasetest.RestrictionUnlimited),
		usecasetest.NewTestCard("TST-0002", usecasetest.FactionTenki, usecasetest.RestrictionUnlimited),
	}
	setup.OwnedCards = append(usecasetest.ToPlayerCards(entries), usecasetest.NewOwnedCard("TST-0001", 1, 2))
	setup.OwnedFactions = []string{usecasetest.FactionSHE}
	return failedUpdateScenario{setup: setup, draft: newTenkiDraft(entries...)}
}

func newTenkiDraft(entries ...domain.DeckCardEntry) domain.DeckDraft {
	return domain.DeckDraft{
		DeckName:  "更新後デッキ",
		Faction:   usecasetest.FactionTenki,
		ProductID: "TST-0002",
		RoutineID: "TST-1005",
		SpecialID: "TST-1006",
		Cards:     entries,
	}
}

func (s failedUpdateScenario) newHarnessWithSavedDeck(t *testing.T) (*usecasetest.DeckHarness, int64) {
	t.Helper()
	h := usecasetest.NewDeckHarness(t, s.setup)
	beforeUpdate := domain.Deck{
		PlayerID:  usecasetest.PlayerA,
		DeckName:  "更新前デッキ",
		Faction:   usecasetest.FactionSHE,
		ProductID: "TST-0001",
		RoutineID: "TST-1001",
		SpecialID: "TST-1002",
	}
	deckID := h.SeedDeck(t, beforeUpdate, []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 2)})
	return h, deckID
}

func TestDeckUpdate(t *testing.T) {
	t.Run("[デッキ]デッキ更新", func(t *testing.T) {
		t.Run("デッキを更新する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("時計が2026-01-02T03:04:05Zを返すときに作成したデッキを、時計が2026-02-03T04:05:06Zを返すときに更新すると", func(t *testing.T) {
					createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
					updatedAt := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

					t.Run("応答の作成日時が2026-01-02T03:04:05Zになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						h.Clock.CurrentTime = createdAt
						created := h.CreateDeck(t, usecasetest.NewDraft())
						h.Clock.CurrentTime = updatedAt

						updated, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, created.Deck.DeckID, usecasetest.NewDraft())

						assert.NoError(t, err)
						assert.WithinDuration(t, createdAt, updated.Deck.CreatedAt, 0)
					})

					t.Run("応答の更新日時が2026-02-03T04:05:06Zになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						h.Clock.CurrentTime = createdAt
						created := h.CreateDeck(t, usecasetest.NewDraft())
						h.Clock.CurrentTime = updatedAt

						updated, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, created.Deck.DeckID, usecasetest.NewDraft())

						assert.NoError(t, err)
						assert.WithinDuration(t, updatedAt, updated.Deck.UpdatedAt, 0)
					})
				})

				t.Run("更新後のカードの合計枚数ごとのis_valid", func(t *testing.T) {
					t.Run("合計が29枚のとき、応答のis_validはfalseになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(29)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
						created := h.CreateDeck(t, usecasetest.NewDraft())

						updated, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, created.Deck.DeckID, usecasetest.NewDraft(entries...))

						assert.NoError(t, err)
						assert.False(t, updated.IsValid)
					})

					t.Run("合計が30枚のとき、応答のis_validはtrueになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(30)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
						created := h.CreateDeck(t, usecasetest.NewDraft())

						updated, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, created.Deck.DeckID, usecasetest.NewDraft(entries...))

						assert.NoError(t, err)
						assert.True(t, updated.IsValid)
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("合計が31枚のとき、枚数超過を示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					entries := usecasetest.BuildDeckCardEntries(31)
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
					deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), nil)

					_, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, usecasetest.NewDraft(entries...))

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "cannot exceed")
				})

				t.Run("存在しないプロダクトTST-9999を指定したとき、プロダクトが無いことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
					deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), nil)
					draft := usecasetest.NewDraft()
					draft.ProductID = "TST-9999"

					_, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, draft)

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "TST-9999", "not found")
				})

				t.Run("プレイヤーがTenkiだけを所持しているとき、宣言陣営にSHEを指定すると、陣営を所持していないことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					setup := usecasetest.NewDefaultDeckSetup()
					setup.OwnedFactions = []string{"Tenki"}
					h := usecasetest.NewDeckHarness(t, setup)
					deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), nil)

					_, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, usecasetest.NewDraft())

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "is not owned by player")
				})

				t.Run("更新前のデッキが、デッキ名「更新前デッキ」、宣言陣営SHE、プロダクトTST-0001、ルーチン施策TST-1001、スペシャル施策TST-1002、TST-0001のアート1が2枚の構成のとき", func(t *testing.T) {
					t.Run("デッキ名「更新後デッキ」、宣言陣営Tenki、プロダクトTST-0002、ルーチン施策TST-1005、スペシャル施策TST-1006、TST-0002のアート1を1枚含む構成への更新に失敗したあと、同じdeck_idでデッキを取得するとき", func(t *testing.T) {
						t.Run("deck_name", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのdeck_nameが、「更新前デッキ」になる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのdeck_nameが、「更新前デッキ」になる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, "更新前デッキ", fetched.Deck.DeckName)
								})
							}
						})

						t.Run("faction", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのfactionが、SHEになる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのfactionが、SHEになる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, "SHE", fetched.Deck.Faction)
								})
							}
						})

						t.Run("product_id", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのproduct_idが、TST-0001になる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのproduct_idが、TST-0001になる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, "TST-0001", fetched.Deck.ProductID)
								})
							}
						})

						t.Run("routine_id", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのroutine_idが、TST-1001になる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのroutine_idが、TST-1001になる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, "TST-1001", fetched.Deck.RoutineID)
								})
							}
						})

						t.Run("special_id", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのspecial_idが、TST-1002になる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのspecial_idが、TST-1002になる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, "TST-1002", fetched.Deck.SpecialID)
								})
							}
						})

						t.Run("deck_cards", func(t *testing.T) {
							tests := []struct {
								name     string
								scenario failedUpdateScenario
							}{
								{name: "合計が31枚の構成のとき、取得したデッキのdeck_cardsが、TST-0001のアート1が2枚だけになる", scenario: newFailedUpdateScenarioWithTotalOfThirtyOne()},
								{name: "プレイヤーが宣言陣営Tenkiを所持していないとき、取得したデッキのdeck_cardsが、TST-0001のアート1が2枚だけになる", scenario: newFailedUpdateScenarioWithoutTenki()},
							}
							for _, tt := range tests {
								t.Run(tt.name, func(t *testing.T) {
									h, deckID := tt.scenario.newHarnessWithSavedDeck(t)
									_, updateErr := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, tt.scenario.draft)
									require.Error(t, updateErr)

									fetched := h.FetchDeck(t, deckID)

									assert.Equal(t, map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 2}, usecasetest.CountByCardArt(fetched.Cards))
								})
							}
						})
					})
				})

				t.Run("更新対象のdeck_idのデッキが存在しないとき、見つからないことを示すエラーになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())

					_, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NonexistentDeckID, usecasetest.NewDraft())

					assert.ErrorIs(t, err, port.ErrNotFound)
				})

				t.Run("依存先がエラーを返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						failure error
						inject  func(h *usecasetest.DeckHarness, failure error)
					}{
						{
							name:    "所持陣営のaccountサービスが「更新時の陣営照会の失敗」のエラーを返したとき、エラーの原因が「更新時の陣営照会の失敗」のエラーになる",
							failure: errors.New("更新時の陣営照会の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.Factions.Err = failure },
						},
						{
							name:    "所持カードのデータベースが「更新時の所持カード取得の失敗」のエラーを返したとき、エラーの原因が「更新時の所持カード取得の失敗」のエラーになる",
							failure: errors.New("更新時の所持カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.PlayerCards.GetErr = failure },
						},
						{
							name:    "デッキのデータベースが更新時に「更新時のデッキ更新の失敗」のエラーを返したとき、エラーの原因が「更新時のデッキ更新の失敗」のエラーになる",
							failure: errors.New("更新時のデッキ更新の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.UpdateErr = failure },
						},
						{
							name:    "デッキのデータベースが更新後のデッキ取得時に「更新時の更新後デッキ取得の失敗」のエラーを返したとき、エラーの原因が「更新時の更新後デッキ取得の失敗」のエラーになる",
							failure: errors.New("更新時の更新後デッキ取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.FindByIDErr = failure },
						},
						{
							name:    "デッキのデータベースが更新後の構成カード取得時に「更新時の構成カード取得の失敗」のエラーを返したとき、エラーの原因が「更新時の構成カード取得の失敗」のエラーになる",
							failure: errors.New("更新時の構成カード取得の失敗"),
							inject:  func(h *usecasetest.DeckHarness, failure error) { h.DeckRepo.GetDeckCardsErr = failure },
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 1)}
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							deckID := h.SeedDeck(t, usecasetest.NewSavedDeck(), nil)
							tt.inject(h, tt.failure)

							_, err := h.Interactor.UpdateDeck(t.Context(), usecasetest.PlayerA, deckID, usecasetest.NewDraft(entries...))

							assert.ErrorIs(t, err, tt.failure)
						})
					}
				})
			})
		})
	})
}
