package routertest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EncodeJSON は値を JSON の文字列にします。
func EncodeJSON(t *testing.T, value any) string {
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	return string(encoded)
}

// DecodeObject は応答本文を JSON のオブジェクトとして読みます。
func DecodeObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	var object map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &object), recorder.Body.String())
	return object
}

// DecodeArray は応答本文を JSON の配列として読みます。
func DecodeArray(t *testing.T, recorder *httptest.ResponseRecorder) []any {
	var array []any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &array), recorder.Body.String())
	return array
}

// ToObject は読み込んだ JSON の値をオブジェクトに変換します。
func ToObject(t *testing.T, value any) map[string]any {
	object, isObject := value.(map[string]any)
	require.True(t, isObject, "オブジェクトではない値: %v", value)
	return object
}

// ToArray は読み込んだ JSON の値を配列に変換します。
func ToArray(t *testing.T, value any) []any {
	array, isArray := value.([]any)
	require.True(t, isArray, "配列ではない値: %v", value)
	return array
}

// ExtractSingleObject は応答本文の配列から、唯一の要素のオブジェクトを取り出します。
func ExtractSingleObject(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	array := DecodeArray(t, recorder)
	require.Len(t, array, 1)
	return ToObject(t, array[0])
}

// ExtractSingleElement は配列の値から、唯一の要素のオブジェクトを取り出します。
func ExtractSingleElement(t *testing.T, value any) map[string]any {
	array := ToArray(t, value)
	require.Len(t, array, 1)
	return ToObject(t, array[0])
}

// CollectValues は配列の各オブジェクトから指定の項目の値を集めます。
func CollectValues(t *testing.T, items []any, key string) []any {
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, ToObject(t, item)[key])
	}
	return values
}

// CountValues は値の一覧に指定の値がいくつあるかを数えます。
func CountValues(values []any, want any) int {
	return len(slices.DeleteFunc(slices.Clone(values), func(value any) bool { return value != want }))
}

// FindObject は配列から指定の項目が指定の値のオブジェクトを探します。
func FindObject(t *testing.T, items []any, key string, want any) map[string]any {
	index := slices.IndexFunc(items, func(item any) bool { return ToObject(t, item)[key] == want })
	require.GreaterOrEqual(t, index, 0, "%s が %v の要素が無い: %v", key, want, items)
	return ToObject(t, items[index])
}

// BuildEnvelope はイベントの JSON を base64 にして Pub/Sub の push 本文に包みます。
func BuildEnvelope(eventJSON string) string {
	return `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(eventJSON)) + `"}}`
}

// BuildPlayerOnboardedEventJSON はプレイヤー A のプレイヤー登録完了イベントの JSON を組み立てます。
func BuildPlayerOnboardedEventJSON(factionID string) string {
	return fmt.Sprintf(`{"event_type":"player_onboarded","event_id":"11111111-1111-4111-8111-111111111111","timestamp":"2026-01-02T03:04:05Z","player_id":%q,"initial_faction_id":%q}`, PlayerA, factionID)
}

// BuildCardPackPurchasedEventJSON はプレイヤー A のカードパック購入イベントの JSON を組み立てます。
func BuildCardPackPurchasedEventJSON(packID string) string {
	return fmt.Sprintf(`{"event_type":"card_pack_purchased","event_id":"22222222-2222-4222-8222-222222222222","timestamp":"2026-01-02T03:04:05Z","player_id":%q,"card_pack_id":%q}`, PlayerA, packID)
}

// ErrorLogs は捕捉したログのうち Error レベルのものを返します。
func (h *Harness) ErrorLogs(t *testing.T) []map[string]any {
	records := []map[string]any{}
	for _, line := range strings.FieldsFunc(h.Logs.String(), func(r rune) bool { return r == '\n' }) {
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record), line)
		records = append(records, record)
	}
	return slices.DeleteFunc(records, func(record map[string]any) bool { return record["level"] != "ERROR" })
}

// RequireErrorLogContaining は Error レベルのログの指定の項目が指定の文字列を含むことを確かめます。
func (h *Harness) RequireErrorLogContaining(t *testing.T, key, want string) {
	logs := h.ErrorLogs(t)
	assert.True(t,
		slices.ContainsFunc(logs, func(record map[string]any) bool { return strings.Contains(fmt.Sprint(record[key]), want) }),
		"Errorレベルのログの%s項目が%qを含まない: %v", key, want, logs)
}

// RequireErrorLogEquals は Error レベルのログの指定の項目が指定の値に一致することを確かめます。
func (h *Harness) RequireErrorLogEquals(t *testing.T, key, want string) {
	logs := h.ErrorLogs(t)
	assert.True(t,
		slices.ContainsFunc(logs, func(record map[string]any) bool { return record[key] == want }),
		"Errorレベルのログの%s項目が%qでない: %v", key, want, logs)
}

// InternalErrorCase は失敗の原因ごとの 500 の応答とログの確認の 1 件です。
type InternalErrorCase struct {
	Given   string
	Cause   string
	Arrange func(h *Harness)
}

// RunInternalErrorCases は原因ごとに、500 の応答とログの error 項目に原因を含むことを別々のケースで確かめます。
func RunInternalErrorCases(t *testing.T, newScenario func(t *testing.T) *Harness, send func(h *Harness) *httptest.ResponseRecorder, cases []InternalErrorCase) {
	for _, tc := range cases {
		t.Run(tc.Given, func(t *testing.T) {
			t.Run("内部エラーを示す500の応答になる", func(t *testing.T) {
				h := newScenario(t)
				tc.Arrange(h)

				recorder := send(h)

				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			})
			t.Run("Errorレベルのログのerror項目は"+tc.Cause+"を含む", func(t *testing.T) {
				h := newScenario(t)
				tc.Arrange(h)

				send(h)

				h.RequireErrorLogContaining(t, "error", tc.Cause)
			})
		})
	}
}
