package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getProtectedResource(c *gin.Context) {
	authField, authFieldIsPresent := c.Request.Header["Authorization"]

	// Check invalid tokens (macaroons)
	if !authFieldIsPresent || !IsValidMacaroon(authField[0]) {
		// Generate invoice and token
		// Example
		macaroon := "AGIAJEemVQUTEyNCR0exk7ek90Cg=="
		invoice := "lnbc1500n1pw5kjhmpp5fu6xhthlt2vucmzkx6c7w"

		c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroon, invoice))
		c.String(http.StatusPaymentRequired, "402 Payment Required")
	}
}

func main() {
	router := gin.Default()
	router.GET("/protected", getProtectedResource)

	router.Run("localhost:8080")
}
