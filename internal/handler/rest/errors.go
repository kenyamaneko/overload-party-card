package rest

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

func toHTTPStatus(err error) int {
	switch {
	case errors.Is(err, port.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, port.ErrUnowned):
		return http.StatusForbidden
	case errors.Is(err, port.ErrInvalidDeck),
		errors.Is(err, port.ErrRestrictionExceeded),
		errors.Is(err, port.ErrInvalidArgument):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func respondError(c *gin.Context, err error) {
	c.JSON(toHTTPStatus(err), gin.H{"error": err.Error()})
}
