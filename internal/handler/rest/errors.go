package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

// internalErrorMessage は 500 応答本文に返す固定文言。呼び出し側が対処できない
// 障害のため原因を応答へ載せず、構造化ログにのみ記録する。
const internalErrorMessage = "internal server error"

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

func respondError(c *gin.Context, err error) {
	status := toHTTPStatus(err)
	if status != http.StatusInternalServerError {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	slog.Error("internal server error",
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"error", err,
	)
	c.JSON(status, gin.H{"error": internalErrorMessage})
}
