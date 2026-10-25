package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/usecase/usecasetest"
)

func TestDeckCompositionRules(t *testing.T) {
	t.Run("[デッキ]デッキ構成ルール", func(t *testing.T) {
		t.Run("デッキを作成する", func(t *testing.T) {
			t.Run("正常系", func(t *testing.T) {
				t.Run("宣言陣営がSHEのとき、デッキ一覧で、作成したデッキの宣言陣営がSHEになる", func(t *testing.T) {
					h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
					created := h.CreateDeck(t, usecasetest.NewDraft())

					listed := h.FindListedDeck(t, created.Deck.DeckID)

					assert.Equal(t, "SHE", listed.Deck.Faction)
				})

				t.Run("カードの合計枚数", func(t *testing.T) {
					tests := []struct {
						name  string
						total int
					}{
						{name: "合計が0枚のとき、デッキ一覧で、作成したデッキの構成カードの合計が0枚になる", total: 0},
						{name: "合計が30枚のとき、デッキ一覧で、作成したデッキの構成カードの合計が30枚になる", total: 30},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							entries := usecasetest.BuildDeckCardEntries(tt.total)
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))
							created := h.CreateDeck(t, usecasetest.NewDraft(entries...))

							listed := h.FindListedDeck(t, created.Deck.DeckID)

							assert.Equal(t, tt.total, usecasetest.SumDeckCardCounts(listed.Cards))
						})
					}
				})

				t.Run("所持枚数", func(t *testing.T) {
					tests := []struct {
						name    string
						owned   []domain.PlayerCard
						entries []domain.DeckCardEntry
						want    map[usecasetest.CardArtKey]int
					}{
						{
							name:    "TST-0001のアート1を3枚所持していて、3枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が3枚含まれる",
							owned:   []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3)},
							entries: []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 3)},
							want:    map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 3},
						},
						{
							name:    "TST-0001のアート1を3枚、アート2を2枚所持していて、アート1を2枚、アート2を1枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が2枚、アート2が1枚含まれる",
							owned:   []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3), usecasetest.NewOwnedCard("TST-0001", 2, 2)},
							entries: []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 2), usecasetest.NewDeckCardEntry("TST-0001", 2, 1)},
							want:    map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 2, {CardID: "TST-0001", ArtNo: 2}: 1},
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							setup := usecasetest.NewDefaultDeckSetup()
							setup.OwnedCards = tt.owned
							h := usecasetest.NewDeckHarness(t, setup)
							created := h.CreateDeck(t, usecasetest.NewDraft(tt.entries...))

							listed := h.FindListedDeck(t, created.Deck.DeckID)

							got := usecasetest.CountByCardArt(listed.Cards)
							for key, count := range tt.want {
								assert.Equal(t, count, got[key])
							}
						})
					}
				})

				t.Run("宣言陣営がSHEで、カードの陣営ごと", func(t *testing.T) {
					tests := []struct {
						name        string
						cardFaction string
					}{
						{name: "SHE陣営のカードTST-0001のアート1を1枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が1枚含まれる", cardFaction: "SHE"},
						{name: "Neutral陣営のカードTST-0001のアート1を1枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が1枚含まれる", cardFaction: "Neutral"},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							setup := usecasetest.NewDefaultDeckSetup()
							setup.Cards = []*domain.Card{usecasetest.NewTestCard("TST-0001", tt.cardFaction, usecasetest.RestrictionUnlimited)}
							setup.OwnedCards = []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1)}
							h := usecasetest.NewDeckHarness(t, setup)
							created := h.CreateDeck(t, usecasetest.NewDraft(usecasetest.NewDeckCardEntry("TST-0001", 1, 1)))

							listed := h.FindListedDeck(t, created.Deck.DeckID)

							assert.Equal(t, 1, usecasetest.CountByCardArt(listed.Cards)[usecasetest.CardArtKey{CardID: "TST-0001", ArtNo: 1}])
						})
					}
				})

				t.Run("制限区分ごとの同一カードの枚数", func(t *testing.T) {
					tests := []struct {
						name        string
						restriction string
						owned       []domain.PlayerCard
						entries     []domain.DeckCardEntry
						want        map[usecasetest.CardArtKey]int
					}{
						{
							name:        "制限区分がunlimitedのカードTST-0001のアート1を3枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が3枚含まれる",
							restriction: "unlimited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 3)},
							want:        map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 3},
						},
						{
							name:        "制限区分がsemi_limitedのカードTST-0001のアート1を2枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が2枚含まれる",
							restriction: "semi_limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 2)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 2)},
							want:        map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 2},
						},
						{
							name:        "制限区分がlimitedのカードTST-0001のアート1を1枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が1枚含まれる",
							restriction: "limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 1)},
							want:        map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 1},
						},
						{
							name:        "制限区分がsemi_limitedのカードTST-0001のアート1を1枚、アート2を1枚入れるとき、デッキ一覧で、作成したデッキの構成カードに、TST-0001のアート1が1枚、アート2が1枚含まれる",
							restriction: "semi_limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1), usecasetest.NewOwnedCard("TST-0001", 2, 1)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 1), usecasetest.NewDeckCardEntry("TST-0001", 2, 1)},
							want:        map[usecasetest.CardArtKey]int{{CardID: "TST-0001", ArtNo: 1}: 1, {CardID: "TST-0001", ArtNo: 2}: 1},
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							setup := usecasetest.NewDefaultDeckSetup()
							setup.Cards = []*domain.Card{usecasetest.NewTestCard("TST-0001", usecasetest.FactionSHE, tt.restriction)}
							setup.OwnedCards = tt.owned
							h := usecasetest.NewDeckHarness(t, setup)
							created := h.CreateDeck(t, usecasetest.NewDraft(tt.entries...))

							listed := h.FindListedDeck(t, created.Deck.DeckID)

							got := usecasetest.CountByCardArt(listed.Cards)
							for key, count := range tt.want {
								assert.Equal(t, count, got[key])
							}
						})
					}
				})

				t.Run("宣言陣営がSHEで、SHE陣営のプロダクトTST-0001と、そのプロダクトのルーチン施策TST-1001、スペシャル施策TST-1002を指定するとき", func(t *testing.T) {
					t.Run("デッキ一覧で、作成したデッキのproduct_idが、TST-0001になる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						created := h.CreateDeck(t, usecasetest.NewDraft())

						listed := h.FindListedDeck(t, created.Deck.DeckID)

						assert.Equal(t, "TST-0001", listed.Deck.ProductID)
					})

					t.Run("デッキ一覧で、作成したデッキのroutine_idが、TST-1001になる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						created := h.CreateDeck(t, usecasetest.NewDraft())

						listed := h.FindListedDeck(t, created.Deck.DeckID)

						assert.Equal(t, "TST-1001", listed.Deck.RoutineID)
					})

					t.Run("デッキ一覧で、作成したデッキのspecial_idが、TST-1002になる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						created := h.CreateDeck(t, usecasetest.NewDraft())

						listed := h.FindListedDeck(t, created.Deck.DeckID)

						assert.Equal(t, "TST-1002", listed.Deck.SpecialID)
					})
				})
			})

			t.Run("異常系", func(t *testing.T) {
				t.Run("宣言陣営", func(t *testing.T) {
					tests := []struct {
						name    string
						faction string
					}{
						{name: "宣言陣営がNeutralのとき、選択できない陣営であることを示すデッキ構成の不正のエラーになる", faction: "Neutral"},
						{name: "宣言陣営が空文字のとき、選択できない陣営であることを示すデッキ構成の不正のエラーになる", faction: ""},
						{name: "宣言陣営が未定義の陣営名Unknownのとき、選択できない陣営であることを示すデッキ構成の不正のエラーになる", faction: "Unknown"},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
							draft := usecasetest.NewDraft()
							draft.Faction = tt.faction

							_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, draft)

							assert.ErrorIs(t, err, port.ErrInvalidDeck)
							usecasetest.AssertErrorContainsAll(t, err, "is not selectable")
						})
					}
				})

				t.Run("カードの枚数", func(t *testing.T) {
					t.Run("リクエストに、枚数が0枚のカードの指定を含むとき、枚数が正でないことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
						setup := usecasetest.NewDefaultDeckSetup()
						setup.OwnedCards = []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3)}
						h := usecasetest.NewDeckHarness(t, setup)

						_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(usecasetest.NewDeckCardEntry("TST-0001", 1, 0)))

						assert.ErrorIs(t, err, port.ErrInvalidDeck)
						usecasetest.AssertErrorContainsAll(t, err, "count must be positive")
					})

					t.Run("合計が31枚のとき、枚数超過を示すデッキ構成の不正のエラーになる", func(t *testing.T) {
						entries := usecasetest.BuildDeckCardEntries(31)
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDeckSetupOwning(entries))

						_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(entries...))

						assert.ErrorIs(t, err, port.ErrInvalidDeck)
						usecasetest.AssertErrorContainsAll(t, err, "cannot exceed")
					})
				})

				t.Run("所持枚数", func(t *testing.T) {
					tests := []struct {
						name    string
						owned   []domain.PlayerCard
						entries []domain.DeckCardEntry
					}{
						{
							name:    "TST-0001のアート1を3枚所持していて、4枚入れるとき、所持枚数の不足を示すエラーになる",
							owned:   []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3)},
							entries: []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 4)},
						},
						{
							name:    "TST-0001のアート1を1枚だけ所持していて、TST-0001のアート2を1枚入れるとき、所持枚数の不足を示すエラーになる",
							owned:   []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1)},
							entries: []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 2, 1)},
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							setup := usecasetest.NewDefaultDeckSetup()
							setup.OwnedCards = tt.owned
							h := usecasetest.NewDeckHarness(t, setup)

							_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(tt.entries...))

							assert.ErrorIs(t, err, port.ErrUnowned)
							usecasetest.AssertErrorContainsAll(t, err, "not enough owned")
						})
					}
				})

				t.Run("TST-9999のアート1を1枚所持しているが、カード定義にTST-9999が無いとき、TST-9999のアート1を1枚入れると、カード定義が無いことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					setup := usecasetest.NewDefaultDeckSetup()
					setup.OwnedCards = []domain.PlayerCard{usecasetest.NewOwnedCard("TST-9999", 1, 1)}
					h := usecasetest.NewDeckHarness(t, setup)

					_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(usecasetest.NewDeckCardEntry("TST-9999", 1, 1)))

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "not found in card definitions")
				})

				t.Run("宣言陣営がSHEで、Tenki陣営のカードTST-0001のアート1を1枚所持していて、1枚入れるとき、陣営の不一致を示すデッキ構成の不正のエラーになる", func(t *testing.T) {
					setup := usecasetest.NewDefaultDeckSetup()
					setup.Cards = []*domain.Card{usecasetest.NewTestCard("TST-0001", usecasetest.FactionTenki, usecasetest.RestrictionUnlimited)}
					setup.OwnedCards = []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1)}
					h := usecasetest.NewDeckHarness(t, setup)

					_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(usecasetest.NewDeckCardEntry("TST-0001", 1, 1)))

					assert.ErrorIs(t, err, port.ErrInvalidDeck)
					usecasetest.AssertErrorContainsAll(t, err, "deck faction is")
				})

				t.Run("制限区分ごとの同一カードの枚数", func(t *testing.T) {
					tests := []struct {
						name        string
						restriction string
						owned       []domain.PlayerCard
						entries     []domain.DeckCardEntry
					}{
						{
							name:        "制限区分がunlimitedのカードTST-0001のアート1を4枚所持していて、4枚入れるとき、制限枚数の超過を示すエラーになる",
							restriction: "unlimited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 4)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 4)},
						},
						{
							name:        "制限区分がsemi_limitedのカードTST-0001のアート1を3枚所持していて、3枚入れるとき、制限枚数の超過を示すエラーになる",
							restriction: "semi_limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 3)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 3)},
						},
						{
							name:        "制限区分がlimitedのカードTST-0001のアート1を2枚所持していて、2枚入れるとき、制限枚数の超過を示すエラーになる",
							restriction: "limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 2)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 2)},
						},
						{
							name:        "制限区分がforbiddenのカードTST-0001のアート1を1枚所持していて、1枚入れるとき、制限枚数の超過を示すエラーになる",
							restriction: "forbidden",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 1)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 1)},
						},
						{
							name:        "制限区分がsemi_limitedのカードTST-0001のアート1を2枚、アート2を1枚所持していて、アート1を2枚、アート2を1枚入れるとき、制限枚数の超過を示すエラーになる",
							restriction: "semi_limited",
							owned:       []domain.PlayerCard{usecasetest.NewOwnedCard("TST-0001", 1, 2), usecasetest.NewOwnedCard("TST-0001", 2, 1)},
							entries:     []domain.DeckCardEntry{usecasetest.NewDeckCardEntry("TST-0001", 1, 2), usecasetest.NewDeckCardEntry("TST-0001", 2, 1)},
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							setup := usecasetest.NewDefaultDeckSetup()
							setup.Cards = []*domain.Card{usecasetest.NewTestCard("TST-0001", usecasetest.FactionSHE, tt.restriction)}
							setup.OwnedCards = tt.owned
							h := usecasetest.NewDeckHarness(t, setup)

							_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, usecasetest.NewDraft(tt.entries...))

							assert.ErrorIs(t, err, port.ErrRestrictionExceeded)
							usecasetest.AssertErrorContainsAll(t, err, "exceeds restriction limit")
						})
					}
				})

				t.Run("施策", func(t *testing.T) {
					t.Run("存在しないプロダクトTST-9999を指定したとき、プロダクトが無いことを示すデッキ構成の不正のエラーになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						draft := usecasetest.NewDraft()
						draft.ProductID = "TST-9999"

						_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, draft)

						assert.ErrorIs(t, err, port.ErrInvalidDeck)
						usecasetest.AssertErrorContainsAll(t, err, "TST-9999", "not found")
					})

					t.Run("宣言陣営がSHEのとき、Tenki陣営に属するプロダクトTST-0002を指定すると、陣営の不一致を示すデッキ構成の不正のエラーになる", func(t *testing.T) {
						h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
						draft := usecasetest.NewDraft()
						draft.ProductID = "TST-0002"
						draft.RoutineID = "TST-1005"
						draft.SpecialID = "TST-1006"

						_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, draft)

						assert.ErrorIs(t, err, port.ErrInvalidDeck)
						usecasetest.AssertErrorContainsAll(t, err, "belongs to faction")
					})

					t.Run("宣言陣営がSHEで、SHE陣営のプロダクトTST-0001を指定していて、ルーチン施策とスペシャル施策の片方だけを誤って指定するとき", func(t *testing.T) {
						tests := []struct {
							name              string
							routineID         string
							specialID         string
							wrongInitiativeID string
						}{
							{name: "存在しないルーチン施策TST-9998を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-9998", specialID: "TST-1002", wrongInitiativeID: "TST-9998"},
							{name: "別のプロダクトTST-0003のルーチン施策TST-1003を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-1003", specialID: "TST-1002", wrongInitiativeID: "TST-1003"},
							{name: "ルーチン施策にスペシャル施策TST-1002を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-1002", specialID: "TST-1002", wrongInitiativeID: "TST-1002"},
							{name: "存在しないスペシャル施策TST-9999を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-1001", specialID: "TST-9999", wrongInitiativeID: "TST-9999"},
							{name: "別のプロダクトTST-0003のスペシャル施策TST-1004を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-1001", specialID: "TST-1004", wrongInitiativeID: "TST-1004"},
							{name: "スペシャル施策にルーチン施策TST-1001を指定したとき、施策の不一致を示すデッキ構成の不正のエラーになる", routineID: "TST-1001", specialID: "TST-1001", wrongInitiativeID: "TST-1001"},
						}
						for _, tt := range tests {
							t.Run(tt.name, func(t *testing.T) {
								h := usecasetest.NewDeckHarness(t, usecasetest.NewDefaultDeckSetup())
								draft := usecasetest.NewDraft()
								draft.RoutineID = tt.routineID
								draft.SpecialID = tt.specialID

								_, err := h.Interactor.CreateDeck(t.Context(), usecasetest.PlayerA, draft)

								assert.ErrorIs(t, err, port.ErrInvalidDeck)
								usecasetest.AssertErrorContainsAll(t, err, tt.wrongInitiativeID, "of product")
							})
						}
					})
				})
			})
		})
	})
}
