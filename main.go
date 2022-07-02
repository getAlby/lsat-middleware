package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"proxy/invoice"
	"proxy/lnd"
	"proxy/lnurl"
	"proxy/lsat"
	macaroonutils "proxy/macaroon"
	"proxy/service"
	"proxy/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lightningnetwork/lnd/lnrpc"
)

func getProtectedResource(svc *service.Service) gin.HandlerFunc {
	return func(c *gin.Context) {

		acceptLsatField := c.Request.Header["Accept"]
		// Check if client support LSAT

		authField := c.Request.Header["Authorization"]
		mac, preimage, err := utils.ParseLsatHeader(authField)

		// If macaroon and preimage are valid
		if err == nil {
			rootKey := utils.GetRootKey()

			// Check valid LSAT and return protected content
			err := lsat.VerifyLSAT(mac, rootKey[:], preimage)
			if err != nil {
				c.String(http.StatusAccepted, "Protected content")
				return
			}
		} else if len(acceptLsatField) != 0 && acceptLsatField[0] == `application/vnd.lsat.v1.full+json` {
			// Generate invoice and token
			ctx := context.Background()
			lnInvoice := lnrpc.Invoice{
				Value: 5,
				Memo:  "LSAT",
			}

			invoice, paymentHash, err := invoice.GenerateInvoice(svc, ctx, lnInvoice)
			if err != nil {
				c.Error(err)
				return
			}
			macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash)
			if err != nil {
				c.Error(err)
				return
			}

			c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroonString, invoice))
			c.String(http.StatusPaymentRequired, "402 Payment Required")
			return
		} else {
			// Return Free content if client does not support LSAT
			c.String(http.StatusAccepted, "Free content")
			return
		}
	}
}

func main() {
	router := gin.Default()

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
		Address:     os.Getenv("LND_ADDRESS"),
		MacaroonHex: os.Getenv("MACAROON_HEX"),
	})
	if err != nil {
		fmt.Printf("Failed to create LND client: %v", err)
	}
	lnurlClient := &lnurl.LNURLWrapper{
		Address: os.Getenv("LNURL_ADDRESS"),
	}

	svc := &service.Service{
		LndClient:   lndClient,
		LnurlClient: lnurlClient,
	}

	router.GET("/protected", getProtectedResource(svc))

	router.Run("localhost:8080")
}
