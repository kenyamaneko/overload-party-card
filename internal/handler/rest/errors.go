package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

const (
	// internalErrorMessage は 500 応答の本文に返す固定の文言です。原因は応答に含めず、LogInternalErrors が記録します。
	internalErrorMessage = "internal server error"

	invalidDeckIDMessage        = "invalid deck_id"
	malformedRequestBodyMessage = "malformed request body"
	malformedEnvelopeMessage    = "pubsub push: malformed envelope"
	malformedBase64DataMessage  = "pubsub push: malformed base64 data"
)

// errorResponse は失敗したリクエストの応答本文です。
type errorResponse struct {
	Error string `json:"error"`
}

// LogInternalErrors は、500 の応答の原因になったエラーを、リクエストのパスとメソッドとともに Error レベルで記録します。
func LogInternalErrors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, ginError := range c.Errors {
			slog.Error("internal server error",
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
				"error", ginError.Err,
			)
		}
	}
}

func toHTTPStatus(err error) int {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, port.ErrUnowned):
		return http.StatusForbidden
	case errors.Is(err, port.ErrInvalidDeck),
		errors.Is(err, port.ErrRestrictionExceeded):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// respondError は usecase が返したエラーを、エラーの種類に応じたステータスの応答にします。
func respondError(c *gin.Context, err error) {
	status := toHTTPStatus(err)
	if status == http.StatusInternalServerError {
		respondInternalError(c, err)
		return
	}
	c.JSON(status, errorResponse{Error: err.Error()})
}

// respondInternalError は原因を含めない 500 の応答にし、原因を LogInternalErrors が記録できるようにします。
func respondInternalError(c *gin.Context, err error) {
	_ = c.Error(err)
	c.JSON(http.StatusInternalServerError, errorResponse{Error: internalErrorMessage})
}

func respondBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: message})
}
