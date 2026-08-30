package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kenyamaneko/overload-party-card/internal/domain"
	"github.com/kenyamaneko/overload-party-card/internal/port"
	"github.com/kenyamaneko/overload-party-card/internal/presenter"
	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	gamedesign "github.com/kenyamaneko/overload-party-common/packages/game-design-constants"
)

// baselineDeckCards は baselineCards を永続化済み DeckCard 形式に詰め替えて返す。
func baselineDeckCards(playerID string, deckID int64) []domain.DeckCard {
	return entriesToDeckCards(playerID, deckID, presenter.DeckCardEntriesFromRequest(baselineCards()))
}

func TestDeckContentValidation(t *testing.T) {
	t.Run("[デッキ]デッキ内容検証", func(t *testing.T) {
		t.Run("宣言陣営が選択可能陣営のいずれでもない(例:Neutral)とき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: gamedesign.FactionNeutral,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		countCases := []struct {
			name  string
			count int
		}{
			{"カードエントリの指定枚数が0のとき、デッキを作成するとエラーになる", 0},
			{"カードエントリの指定枚数が負の数(-1)のとき、デッキを作成するとエラーになる", -1},
		}
		for _, tc := range countCases {
			t.Run(tc.name, func(t *testing.T) {
				interactor, _, playerCardRepo, _ := newDeckFixture(t)
				playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 3})
				req := apicard.DeckCreateRequest{
					DeckName: "Test Deck", Faction: fxFaction,
					ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
					Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: tc.count}},
				}

				_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

				require.Error(t, err)
				assert.ErrorIs(t, err, port.ErrInvalidDeck)
			})
		}

		t.Run("カードエントリの指定枚数が1のとき、デッキを作成してもこの条件によるエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})

		t.Run("全カードエントリの合計投入枚数がデッキ上限枚数を超え31枚のとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			cards := append(baselineCards(), apicard.DeckCardEntry{CardID: "TST-0011", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: cards,
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("全カードエントリの合計投入枚数がデッキ上限枚数ちょうど30枚のとき、デッキを作成してもこの条件によるエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})

		t.Run("指定したカード・アート番号の組について、プレイヤーの所持枚数が要求枚数より少ないとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 2})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 3}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrUnowned)
		})

		t.Run("指定したカード・アート番号の組について、プレイヤーがその組を1件も所持していない(所持記録が無い)とき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, _, _ := newDeckFixture(t)
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrUnowned)
		})

		t.Run("指定したカード・アート番号の組について、プレイヤーの所持枚数が要求枚数とちょうど同じとき、デッキを作成してもこの条件によるエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 2})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 2}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})

		t.Run("同一カードIDに対しアート番号が異なる複数のエントリを指定し、合算した投入枚数が制限区分の上限を超えるとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID,
				&domain.PlayerCard{CardID: "TST-LIMITED", ArtNo: 1, Count: 1},
				&domain.PlayerCard{CardID: "TST-LIMITED", ArtNo: 2, Count: 1},
			)
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{
					{CardID: "TST-LIMITED", ArtNo: 1, Count: 1},
					{CardID: "TST-LIMITED", ArtNo: 2, Count: 1},
				},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrRestrictionExceeded)
		})

		t.Run("指定したカードIDがカード定義に存在しないとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-UNDEFINED", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-UNDEFINED", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		factionCases := []struct {
			name   string
			cardID string
		}{
			{"カードの陣営が宣言陣営(SHE)と一致するとき、デッキを作成してもエラーにならない", "TST-0001"},
			{"カードの陣営がNeutralのとき、宣言陣営が何であってもデッキを作成してもエラーにならない", "TST-NEUTRAL"},
		}
		for _, tc := range factionCases {
			t.Run(tc.name, func(t *testing.T) {
				interactor, _, playerCardRepo, _ := newDeckFixture(t)
				playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: tc.cardID, ArtNo: 1, Count: 1})
				req := apicard.DeckCreateRequest{
					DeckName: "Test Deck", Faction: fxFaction,
					ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
					Cards: []apicard.DeckCardEntry{{CardID: tc.cardID, ArtNo: 1, Count: 1}},
				}

				_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

				require.NoError(t, err)
			})
		}

		t.Run("カードの陣営が宣言陣営ともNeutralとも異なる(他の選択可能陣営である)とき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-OTHERFACTION", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-OTHERFACTION", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("カードの制限区分に対応する投入上限が制限区分ごとの上限テーブルに定義されていない(未知の制限区分値)とき、デッキを作成するとエラーになり、そのエラーはデッキ内容検証の違反や制限超過とは異なる種別になる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-UNKNOWNRESTRICTION", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-UNKNOWNRESTRICTION", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.NotErrorIs(t, err, port.ErrInvalidDeck)
			assert.NotErrorIs(t, err, port.ErrRestrictionExceeded)
			assert.NotErrorIs(t, err, port.ErrUnowned)
			assert.ErrorContains(t, err, "restriction")
		})

		t.Run("制限区分がlimited(投入上限1枚)のカードについて、同一カードIDの合計投入枚数が2枚のとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-LIMITED", ArtNo: 1, Count: 2})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-LIMITED", ArtNo: 1, Count: 2}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrRestrictionExceeded)
		})

		t.Run("制限区分がlimited(投入上限1枚)のカードについて、同一カードIDの合計投入枚数が1枚のとき、デッキを作成してもこの条件によるエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-LIMITED", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-LIMITED", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})

		t.Run("制限区分がforbidden(投入上限0枚)のカードを1枚でも投入したとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-FORBIDDEN", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-FORBIDDEN", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrRestrictionExceeded)
		})

		t.Run("宣言陣営が選択可能陣営のいずれかであり、各カードエントリの指定枚数が正の数で合計がデッキ上限枚数以内であり、指定したカード・アート番号の組をすべて要求枚数以上所持しており、指定したカードIDがすべてカード定義に存在し陣営が宣言陣営またはNeutralであり、同一カードIDごとの合計投入枚数がすべて制限区分の上限以内であるとき、デッキを作成してもエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})
	})
}

func TestFactionOwnershipValidation(t *testing.T) {
	t.Run("[デッキ]陣営所持検証", func(t *testing.T) {
		t.Run("プレイヤーの所持陣営一覧の取得元がエラーを返すとき、そのエラーをそのまま返す", func(t *testing.T) {
			interactor, _, playerCardRepo, factionClient := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			injected := errors.New("faction client unavailable")
			factionClient.err = injected
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("プレイヤーの所持陣営一覧に宣言陣営が含まれないとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, factionClient := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			factionClient.factions = []string{fxFaction2}
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("プレイヤーの所持陣営一覧に宣言陣営が含まれるとき、デッキを作成してもエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, factionClient := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			factionClient.factions = []string{fxFaction}
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})
	})
}

func TestInitiativeConsistencyValidation(t *testing.T) {
	t.Run("[デッキ]施策整合検証", func(t *testing.T) {
		t.Run("指定したプロダクトIDがプロダクト定義に存在しないとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: "PD-UNKNOWN", RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("指定したプロダクトの陣営が宣言陣営と異なるとき、デッキを作成するとエラーになる", func(t *testing.T) {
			interactor, _, playerCardRepo, factionClient := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-OTHERFACTION", ArtNo: 1, Count: 1})
			factionClient.factions = []string{fxFaction2}
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction2,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-OTHERFACTION", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		routineCases := []struct {
			name      string
			routineID string
		}{
			{"指定したルーチン施策IDが施策定義に存在しないとき、デッキを作成するとエラーになる", "IN-9999-X"},
			{"指定したルーチン施策IDが施策定義には存在するが指定プロダクトに属さないとき、デッキを作成するとエラーになる", "IN-0002-R"},
			{"指定したルーチン施策IDが施策定義には存在し指定プロダクトにも属するがルーチン区分でないとき、デッキを作成するとエラーになる", "IN-0001-S"},
		}
		for _, tc := range routineCases {
			t.Run(tc.name, func(t *testing.T) {
				interactor, _, playerCardRepo, _ := newDeckFixture(t)
				playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
				req := apicard.DeckCreateRequest{
					DeckName: "Test Deck", Faction: fxFaction,
					ProductID: fxProductID, RoutineID: tc.routineID, SpecialID: fxSpecialID,
					Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
				}

				_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

				require.Error(t, err)
				assert.ErrorIs(t, err, port.ErrInvalidDeck)
			})
		}

		specialCases := []struct {
			name      string
			specialID string
		}{
			{"指定したスペシャル施策IDが施策定義に存在しないとき、デッキを作成するとエラーになる", "IN-9999-Y"},
			{"指定したスペシャル施策IDが施策定義には存在するが指定プロダクトに属さないとき、デッキを作成するとエラーになる", "IN-0002-S"},
			{"指定したスペシャル施策IDが施策定義には存在し指定プロダクトにも属するがスペシャル区分でないとき、デッキを作成するとエラーになる", "IN-0001-R"},
		}
		for _, tc := range specialCases {
			t.Run(tc.name, func(t *testing.T) {
				interactor, _, playerCardRepo, _ := newDeckFixture(t)
				playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
				req := apicard.DeckCreateRequest{
					DeckName: "Test Deck", Faction: fxFaction,
					ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: tc.specialID,
					Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
				}

				_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

				require.Error(t, err)
				assert.ErrorIs(t, err, port.ErrInvalidDeck)
			})
		}

		t.Run("指定したプロダクトIDがプロダクト定義に存在しその陣営が宣言陣営と一致し、指定したルーチン施策IDがそのプロダクトに属するルーチン区分の施策であり、指定したスペシャル施策IDがそのプロダクトに属するスペシャル区分の施策であるとき、デッキを作成してもエラーにはならない", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
		})
	})
}

func TestCreateDeck(t *testing.T) {
	t.Run("[デッキ]デッキ作成", func(t *testing.T) {
		t.Run("プレイヤーの所持カード取得元がエラーを返すとき、そのエラーを返しデッキは作成されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
			assert.Empty(t, deckRepo.decks)
		})

		t.Run("リクエストのカード構成がデッキ内容検証のいずれかの条件に違反するとき、その検証が返すエラーをそのまま返し、デッキは作成されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 0}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Empty(t, deckRepo.decks)
		})

		t.Run("リクエストの宣言陣営が陣営所持検証の条件に違反する(プレイヤーが宣言陣営を所持していない)とき、その検証が返すエラーをそのまま返し、デッキは作成されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, factionClient := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			factionClient.factions = []string{fxFaction2}
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Empty(t, deckRepo.decks)
		})

		t.Run("リクエストのプロダクト・ルーチン施策・スペシャル施策の組み合わせが施策整合検証のいずれかの条件に違反するとき、その検証が返すエラーをそのまま返し、デッキは作成されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: "PD-UNKNOWN", RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Empty(t, deckRepo.decks)
		})

		t.Run("デッキ内容検証・陣営所持検証・施策整合検証をすべて満たし、カードエントリの合計投入枚数がデッキ上限枚数未満(29枚)のとき、エラーを返さずデッキを作成し、作成されたデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			cards := baselineCards()
			cards[0].Count = 2 // 合計を 30 から 29 に減らす
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: cards,
			}

			resp, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
			assert.False(t, resp.IsValid)
			assert.Len(t, deckRepo.decks, 1)
		})

		t.Run("デッキ内容検証・陣営所持検証・施策整合検証をすべて満たし、カードエントリの合計投入枚数がデッキ上限枚数ちょうど(30枚)のとき、エラーを返さずデッキを作成し、作成されたデッキのis_validはtrueになる", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			resp, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
			assert.True(t, resp.IsValid)
		})

		t.Run("デッキの永続化処理がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			injected := errors.New("deck repo unavailable")
			deckRepo.createErr = injected
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ作成後のデッキ構成カード再取得がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			injected := errors.New("deck repo unavailable")
			deckRepo.getDeckCardsErr = injected
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ作成が成功したとき、返るデッキのdeck_idは永続化処理が採番したIDと一致する", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			deckRepo.nextDeckID = 42
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			resp, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
			assert.Equal(t, int64(42), resp.DeckID)
		})

		t.Run("デッキ作成が成功したとき、返るデッキのデッキ名・陣営・プロダクトID・ルーチン施策ID・スペシャル施策ID・プレイマット番号・スリーブ番号は、リクエストで指定した値と一致する", func(t *testing.T) {
			interactor, _, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			playmatNo := int64(5)
			sleeveNo := int64(9)
			req := apicard.DeckCreateRequest{
				DeckName: "My Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				PlaymatNo: &playmatNo, SleeveNo: &sleeveNo,
				Cards: baselineCards(),
			}

			resp, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
			assert.Equal(t, "My Deck", resp.DeckName)
			assert.Equal(t, fxFaction, resp.Faction)
			assert.Equal(t, fxProductID, resp.ProductID)
			assert.Equal(t, fxRoutineID, resp.RoutineID)
			assert.Equal(t, fxSpecialID, resp.SpecialID)
			assert.Equal(t, &playmatNo, resp.PlaymatNo)
			assert.Equal(t, &sleeveNo, resp.SleeveNo)
		})

		t.Run("デッキ作成が成功したとき、返るデッキのdeck_cardsは、作成後に再取得したデッキ構成カードの内容になる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			refetched := []domain.DeckCard{{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0099", ArtNo: 7, Count: 4}}
			deckRepo.getDeckCardsOverride = refetched
			req := apicard.DeckCreateRequest{
				DeckName: "Test Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			resp, err := interactor.CreateDeck(context.Background(), fxPlayerID, req)

			require.NoError(t, err)
			require.NotNil(t, resp.DeckCards)
			want := []apicard.DeckCard{{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0099", ArtNo: 7, Count: 4}}
			assert.Equal(t, want, *resp.DeckCards)
		})
	})
}

func TestGetDecks(t *testing.T) {
	t.Run("[デッキ]デッキ一覧取得", func(t *testing.T) {
		t.Run("プレイヤーのデッキ一覧の取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			injected := errors.New("deck repo unavailable")
			deckRepo.findByPlayerErr = injected

			_, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ一覧の取得が成功し、プレイヤーの所持カード取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected

			_, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("プレイヤーがデッキを1件も持たないとき、空の一覧を返す", func(t *testing.T) {
			interactor, _, _, _ := newDeckFixture(t)

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			assert.Empty(t, result)
		})

		t.Run("いずれかのデッキについてデッキ構成カードの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			injected := errors.New("deck repo unavailable")
			deckRepo.getDeckCardsErr = injected

			_, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("あるデッキの構成カード枚数の合計がデッキ上限枚数(30枚)と一致しないとき、そのデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				[]domain.DeckCard{{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0001", ArtNo: 1, Count: 5}},
			)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 5})

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.False(t, result[0].IsValid)
		})

		t.Run("合計がちょうど30枚で、デッキ内容検証・施策整合検証をともに満たすとき、そのデッキのis_validはtrueになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.True(t, result[0].IsValid)
		})

		t.Run("合計が30枚ちょうどでも、デッキ内容検証の条件に違反する(例:制限枚数超過のカードを含む)とき、そのデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			cards := baselineDeckCards(fxPlayerID, 1)
			cards[0].CardID = "TST-LIMITED" // 制限区分 limited (投入上限1枚) のカードを3枚投入し、超過させる
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				cards,
			)
			owned := baselineOwnedCards()
			owned[0].CardID = "TST-LIMITED"
			playerCardRepo.seed(fxPlayerID, owned...)

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.False(t, result[0].IsValid)
		})

		t.Run("合計が30枚ちょうどでも、施策整合検証の条件に違反する(例:ルーチン施策が選択プロダクトと別のプロダクトに属する)とき、そのデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: "IN-0002-R", SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.False(t, result[0].IsValid)
		})

		t.Run("プレイヤーが宣言陣営を現在所持していなくても、構成カード枚数の合計が30枚ちょうどでデッキ内容検証・施策整合検証をともに満たすなら、そのデッキのis_validはtrueになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, factionClient := newDeckFixture(t)
			factionClient.factions = nil
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			result, err := interactor.GetDecks(context.Background(), fxPlayerID)

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.True(t, result[0].IsValid)
		})
	})
}

func TestGetDeck(t *testing.T) {
	t.Run("[デッキ]デッキ詳細取得", func(t *testing.T) {
		t.Run("指定したデッキの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			injected := errors.New("deck repo unavailable")
			deckRepo.findByIDErr = injected

			_, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ構成カードの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			injected := errors.New("deck repo unavailable")
			deckRepo.getDeckCardsErr = injected

			_, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("プレイヤーの所持カード取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected

			_, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("取得したデッキがデッキ内容検証・施策整合検証をともに満たし、構成カードの合計投入枚数がデッキ上限枚数ちょうど(30枚)のとき、返るデッキのis_validはtrueになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			resp, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
			assert.True(t, resp.IsValid)
		})

		t.Run("取得したデッキがデッキ内容検証・施策整合検証をともに満たすが、構成カード枚数の合計がデッキ上限枚数と一致しないとき、返るデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			cards := baselineDeckCards(fxPlayerID, 1)
			cards[0].Count = 2 // 合計を 29 に減らし、デッキ上限枚数と不一致にする
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				cards,
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			resp, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
			assert.False(t, resp.IsValid)
		})

		t.Run("取得に成功したとき、返るデッキのdeck_cardsは取得したデッキ構成カードの内容になる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			cards := []domain.DeckCard{{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0001", ArtNo: 1, Count: 1}}
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, cards)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})

			resp, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
			require.NotNil(t, resp.DeckCards)
			want := []apicard.DeckCard{{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0001", ArtNo: 1, Count: 1}}
			assert.Equal(t, want, *resp.DeckCards)
		})

		t.Run("取得に成功したとき、返るデッキのデッキ名・陣営・プロダクトID・ルーチン施策ID・スペシャル施策IDは、取得したデッキの内容と一致する", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, DeckName: "My Deck", Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				nil,
			)

			resp, err := interactor.GetDeck(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
			assert.Equal(t, "My Deck", resp.DeckName)
			assert.Equal(t, fxFaction, resp.Faction)
			assert.Equal(t, fxProductID, resp.ProductID)
			assert.Equal(t, fxRoutineID, resp.RoutineID)
			assert.Equal(t, fxSpecialID, resp.SpecialID)
		})
	})
}

func TestUpdateDeck(t *testing.T) {
	t.Run("[デッキ]デッキ更新", func(t *testing.T) {
		t.Run("プレイヤーの所持カード取得元がエラーを返すとき、そのエラーを返しデッキは更新されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			original := domain.Deck{PlayerID: fxPlayerID, DeckID: 1, DeckName: "Original", Faction: fxFaction}
			deckRepo.seed(original, nil)
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
			assert.Equal(t, original, deckRepo.decks[1])
		})

		t.Run("リクエストのカード構成がデッキ内容検証のいずれかの条件に違反するとき、その検証が返すエラーをそのまま返し、デッキは更新されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			original := domain.Deck{PlayerID: fxPlayerID, DeckID: 1, DeckName: "Original", Faction: fxFaction}
			deckRepo.seed(original, nil)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 0}},
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Equal(t, original, deckRepo.decks[1])
		})

		t.Run("リクエストの宣言陣営が陣営所持検証の条件に違反する(プレイヤーが宣言陣営を所持していない)とき、その検証が返すエラーをそのまま返し、デッキは更新されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, factionClient := newDeckFixture(t)
			original := domain.Deck{PlayerID: fxPlayerID, DeckID: 1, DeckName: "Original", Faction: fxFaction}
			deckRepo.seed(original, nil)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			factionClient.factions = []string{fxFaction2}
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Equal(t, original, deckRepo.decks[1])
		})

		t.Run("リクエストのプロダクト・ルーチン施策・スペシャル施策の組み合わせが施策整合検証のいずれかの条件に違反するとき、その検証が返すエラーをそのまま返し、デッキは更新されない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			original := domain.Deck{PlayerID: fxPlayerID, DeckID: 1, DeckName: "Original", Faction: fxFaction}
			deckRepo.seed(original, nil)
			playerCardRepo.seed(fxPlayerID, &domain.PlayerCard{CardID: "TST-0001", ArtNo: 1, Count: 1})
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: "PD-UNKNOWN", RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: []apicard.DeckCardEntry{{CardID: "TST-0001", ArtNo: 1, Count: 1}},
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
			assert.Equal(t, original, deckRepo.decks[1])
		})

		t.Run("デッキ内容検証・陣営所持検証・施策整合検証をすべて満たし、カードエントリの合計投入枚数がデッキ上限枚数未満(29枚)のとき、エラーを返さず更新し、返るデッキのis_validはfalseになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			cards := baselineCards()
			cards[0].Count = 2 // 合計を 30 から 29 に減らす
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: cards,
			}

			resp, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.NoError(t, err)
			assert.False(t, resp.IsValid)
		})

		t.Run("デッキ内容検証・陣営所持検証・施策整合検証をすべて満たし、カードエントリの合計投入枚数がデッキ上限枚数ちょうど(30枚)のとき、エラーを返さず更新し、返るデッキのis_validはtrueになる", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			resp, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.NoError(t, err)
			assert.True(t, resp.IsValid)
		})

		t.Run("デッキの永続化処理がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			injected := errors.New("deck repo unavailable")
			deckRepo.updateErr = injected
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ更新後のデッキ構成カード再取得がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			injected := errors.New("deck repo unavailable")
			deckRepo.getDeckCardsErr = injected
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				Cards: baselineCards(),
			}

			_, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ更新が成功したとき、返るデッキのデッキ名・陣営・プロダクトID・ルーチン施策ID・スペシャル施策ID・プレイマット番号・スリーブ番号は、リクエストで指定した値と一致する", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)
			playmatNo := int64(3)
			sleeveNo := int64(8)
			req := apicard.DeckUpdateRequest{
				DeckName: "Updated Deck", Faction: fxFaction,
				ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID,
				PlaymatNo: &playmatNo, SleeveNo: &sleeveNo,
				Cards: baselineCards(),
			}

			resp, err := interactor.UpdateDeck(context.Background(), fxPlayerID, 1, req)

			require.NoError(t, err)
			assert.Equal(t, "Updated Deck", resp.DeckName)
			assert.Equal(t, fxFaction, resp.Faction)
			assert.Equal(t, fxProductID, resp.ProductID)
			assert.Equal(t, fxRoutineID, resp.RoutineID)
			assert.Equal(t, fxSpecialID, resp.SpecialID)
			assert.Equal(t, &playmatNo, resp.PlaymatNo)
			assert.Equal(t, &sleeveNo, resp.SleeveNo)
		})
	})
}

func TestDeleteDeck(t *testing.T) {
	t.Run("[デッキ]デッキ削除", func(t *testing.T) {
		t.Run("指定したデッキの削除処理がエラーを返すとき(例:対象デッキが存在しない)、そのエラーをそのまま返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.deleteErr = port.ErrNotFound

			err := interactor.DeleteDeck(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrNotFound)
		})

		t.Run("削除処理が成功したとき、エラーを返さない", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1}, nil)

			err := interactor.DeleteDeck(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
		})
	})
}

func TestValidateDeckForBattle(t *testing.T) {
	t.Run("[デッキ]バトル可否検証", func(t *testing.T) {
		t.Run("指定したデッキの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			injected := errors.New("deck repo unavailable")
			deckRepo.findByIDErr = injected

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキ構成カードの取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, nil)
			injected := errors.New("deck repo unavailable")
			deckRepo.getDeckCardsErr = injected

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("デッキの構成カード枚数の合計がデッキ上限枚数を超え31枚のとき、エラーになる", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			cards := append(baselineDeckCards(fxPlayerID, 1), domain.DeckCard{PlayerID: fxPlayerID, DeckID: 1, CardID: "TST-0011", ArtNo: 1, Count: 1})
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, cards)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("デッキの構成カード枚数の合計がデッキ上限枚数に満たない29枚のとき、エラーになる", func(t *testing.T) {
			interactor, deckRepo, _, _ := newDeckFixture(t)
			cards := baselineDeckCards(fxPlayerID, 1)
			cards[0].Count = 2
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, cards)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("デッキの構成カード枚数の合計がデッキ上限枚数ちょうど30枚のとき、この条件によるエラーにはならない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
		})

		t.Run("構成カード枚数の合計がデッキ上限枚数ちょうどで、プレイヤーの所持カード取得元がエラーを返すとき、そのエラーを返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction}, baselineDeckCards(fxPlayerID, 1))
			injected := errors.New("player card repo unavailable")
			playerCardRepo.getErr = injected

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, injected)
		})

		t.Run("構成カード枚数の合計がデッキ上限枚数ちょうどで、デッキ内容検証の条件に違反するとき、その検証が返すエラーをそのまま返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			cards := baselineDeckCards(fxPlayerID, 1)
			cards[0].CardID = "TST-LIMITED" // 制限区分 limited (投入上限1枚) のカードを3枚投入し、超過させる
			deckRepo.seed(domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID}, cards)
			owned := baselineOwnedCards()
			owned[0].CardID = "TST-LIMITED"
			playerCardRepo.seed(fxPlayerID, owned...)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrRestrictionExceeded)
		})

		t.Run("構成カード枚数の合計がデッキ上限枚数ちょうどで、施策整合検証の条件に違反するとき、その検証が返すエラーをそのまま返す", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: "IN-0002-R", SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, port.ErrInvalidDeck)
		})

		t.Run("構成カード枚数の合計がデッキ上限枚数ちょうどで、デッキ内容検証・施策整合検証をともに満たすとき、エラーを返さない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, _ := newDeckFixture(t)
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
		})

		t.Run("プレイヤーが宣言陣営を現在所持していなくても、構成カード枚数の合計がデッキ上限枚数ちょうどでデッキ内容検証・施策整合検証をともに満たすなら、エラーを返さない", func(t *testing.T) {
			interactor, deckRepo, playerCardRepo, factionClient := newDeckFixture(t)
			factionClient.factions = nil
			deckRepo.seed(
				domain.Deck{PlayerID: fxPlayerID, DeckID: 1, Faction: fxFaction, ProductID: fxProductID, RoutineID: fxRoutineID, SpecialID: fxSpecialID},
				baselineDeckCards(fxPlayerID, 1),
			)
			playerCardRepo.seed(fxPlayerID, baselineOwnedCards()...)

			err := interactor.ValidateDeckForBattle(context.Background(), fxPlayerID, 1)

			require.NoError(t, err)
		})
	})
}
