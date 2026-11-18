package httpapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ningpm/ningpm/apps/api/internal/version"
)

func handleHealth(c *gin.Context) {
	OK(c, gin.H{"status": "up"})
}

func handleVersion(c *gin.Context) {
	OK(c, gin.H{"version": version.Number})
}
