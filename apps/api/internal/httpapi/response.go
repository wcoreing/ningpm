package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/version"
)

// Envelope is the standard API response shape.
type Envelope struct {
	OK      bool   `json:"ok"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Version string `json:"version"`
}

// OK writes a successful JSON envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		OK:      true,
		Data:    data,
		Version: version.Number,
	})
}

// Fail writes an error JSON envelope.
func Fail(c *gin.Context, status int, msg string) {
	c.JSON(status, Envelope{
		OK:      false,
		Error:   msg,
		Version: version.Number,
	})
}
