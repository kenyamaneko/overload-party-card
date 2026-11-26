package apicardclient_test

import (
	"errors"
	"net/http"
	"testing"

	apicard "github.com/kenyamaneko/overload-party-card/packages/api-card"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient"
	"github.com/kenyamaneko/overload-party-card/packages/api-card/apicardclient/apicardclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionFailure(t *testing.T) {
	t.Run("[カードAPI接続]カードAPIへの接続失敗", func(t *testing.T) {
		t.Run("異常系", func(t *testing.T) {
			t.Run("いずれかの公開操作を呼ぶと、差し替えたHTTPクライアントが「テスト用通信失敗」のエラーを返したとき、操作が返すエラーは「テスト用通信失敗」のエラーを含む", func(t *testing.T) {
				communicationFailure := errors.New("テスト用通信失敗")
				client, err := apicardclient.New(
					"http://card.test",
					apicardclient.WithHTTPClient(apicardclienttest.FailingHTTPClient{Err: communicationFailure}),
				)
				require.NoError(t, err)

				for _, op := range apicardclienttest.AllOperations {
					assert.ErrorIs(t, op.Call(t.Context(), client), communicationFailure, op.Name)
				}
			})
		})
	})
}

func TestStatusCodeClassification(t *testing.T) {
	t.Run("[カードAPIのエラー応答]ステータスコードの分類", func(t *testing.T) {
		t.Run("デッキを更新する", func(t *testing.T) {
			t.Run("異常系", func(t *testing.T) {
				updateDeck := func(t *testing.T, handler http.HandlerFunc) error {
					t.Helper()
					client := apicardclienttest.NewClientFor(t, handler)
					_, err := client.UpdateDeck(t.Context(), 7, apicard.DeckUpdateRequest{DeckName: "TSTデッキ", Faction: "Tenki"})
					return err
				}

				t.Run("カードAPIがステータス401を返したとき、認証失敗を示すエラーになる", func(t *testing.T) {
					err := updateDeck(t, apicardclienttest.RespondEmpty(http.StatusUnauthorized))

					assert.ErrorIs(t, err, apicardclient.ErrUnauthorized)
				})

				t.Run("カードAPIがステータス403を返したとき、アクセス権限がないことを示すエラーになる", func(t *testing.T) {
					err := updateDeck(t, apicardclienttest.RespondEmpty(http.StatusForbidden))

					assert.ErrorIs(t, err, apicardclient.ErrForbidden)
				})

				t.Run("カードAPIがステータス404を返したとき、対象が存在しないことを示すエラーになる", func(t *testing.T) {
					err := updateDeck(t, apicardclienttest.RespondEmpty(http.StatusNotFound))

					assert.ErrorIs(t, err, apicardclient.ErrNotFound)
				})

				t.Run("カードAPIがステータス500を返したとき、サーバー内部エラーを示すエラーになる", func(t *testing.T) {
					err := updateDeck(t, apicardclienttest.RespondEmpty(http.StatusInternalServerError))

					assert.ErrorIs(t, err, apicardclient.ErrInternalServer)
				})

				t.Run("カードAPIが分類のない応答を返したとき", func(t *testing.T) {
					tests := []struct {
						name    string
						handler http.HandlerFunc
					}{
						{
							name:    "ステータスが499のとき、予期しない応答を示すエラーになる",
							handler: apicardclienttest.RespondEmpty(499),
						},
						{
							name:    "ステータスが200で本文のContent-Typeがtext/plainのとき、予期しない応答を示すエラーになる",
							handler: apicardclienttest.RespondText(http.StatusOK, "ok"),
						},
					}
					for _, tt := range tests {
						t.Run(tt.name, func(t *testing.T) {
							err := updateDeck(t, tt.handler)

							apicardclienttest.AssertUnexpectedResponseError(t, err)
						})
					}
				})
			})
		})
	})
}

func TestCommonOperationErrors(t *testing.T) {
	t.Run("[カードAPIのエラー応答]すべての公開操作に共通のエラー", func(t *testing.T) {
		t.Run("いずれかの公開操作を呼ぶ", func(t *testing.T) {
			t.Run("異常系", func(t *testing.T) {
				t.Run("カードAPIがステータス500を返したとき、サーバー内部エラーを示すエラーになる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusInternalServerError))

					for _, op := range apicardclienttest.AllOperations {
						assert.ErrorIs(t, op.Call(t.Context(), client), apicardclient.ErrInternalServer, op.Name)
					}
				})

				t.Run("カードAPIがステータス429を返したとき、予期しない応答を示すエラーになる", func(t *testing.T) {
					client := apicardclienttest.NewClientFor(t, apicardclienttest.RespondEmpty(http.StatusTooManyRequests))

					for _, op := range apicardclienttest.AllOperations {
						apicardclienttest.AssertUnexpectedResponseError(t, op.Call(t.Context(), client), op.Name)
					}
				})
			})
		})
	})
}
