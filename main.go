package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getProtectedResource(c *gin.Context) {
	authField := c.Request.Header["Authorization"][0]

	// Check invalid tokens (macaroons)
	if !IsValidMacaroon(authField) {
		// Generate invoice and token
		// Example
		macaroon := "AGIAJEemVQUTEyNCR0exk7ek90Cg=="
		invoice := "lnbc1500n1pw5kjhmpp5fu6xhthlt2vucmzkx6c7w"

		c.IndentedJSON(http.StatusPaymentRequired, gin.H{"WWW-Authenticate": fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroon, invoice)})
	}
}

func main() {
	router := gin.Default()
	router.GET("/protected", getProtectedResource)

	router.Run("localhost:8080")
}
