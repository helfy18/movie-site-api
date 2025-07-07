package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	c.IndentedJSON(http.StatusNotFound, gin.H{"error": "Route doesn't exist"})
}
