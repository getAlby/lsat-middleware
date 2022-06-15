package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"proxy/lnd"
	"proxy/macaroon"
	"proxy/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lightningnetwork/lnd/lnrpc"
)

type Service struct {
	lndClient *lnd.LNDWrapper
}

func (svc *Service) getProtectedResource(c *gin.Context) {
	authField, authFieldIsPresent := c.Request.Header["Authorization"]

	// Check invalid tokens (macaroons)
	if !authFieldIsPresent || !utils.IsValidMacaroon(authField[0]) {
		// Generate invoice and token
		ctx := context.Background()
		lnInvoice := lnrpc.Invoice{}

		lndInvoice, err := svc.lndClient.AddInvoice(ctx, &lnInvoice)
		if err != nil {
			c.Error(err)
			return
		}

		invoice := lndInvoice.PaymentRequest

		macaroonString, err := macaroon.GetMacaroonAsString(lnInvoice.RHash)
		if err != nil {
			c.Error(err)
			return
		}

		c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroonString, invoice))
		c.String(http.StatusPaymentRequired, "402 Payment Required")
	}
}

func main() {
	router := gin.Default()
	svc := &Service{}

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     os.Getenv("LND_ADDRESS"),
		MacaroonHex: os.Getenv("MACAROON_HEX"),
	})
	if err != nil {
		log.Fatalf("Failed to create LND client: %v", err)
	}
	svc.lndClient = lndClient

	router.GET("/protected", svc.getProtectedResource)

	router.Run("localhost:8080")
}
