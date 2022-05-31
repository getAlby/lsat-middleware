package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"proxy/helper"
	"proxy/lnd"

	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
)

const (
	lndAddress  = "rpc.lnd1.regtest.getalby.com:443"
	macaroonHex = "0201036C6E6402F801030A10E2133A1CAC2C5B4D56E44E32DC64C8551201301A160A0761646472657373120472656164120577726974651A130A04696E666F120472656164120577726974651A170A08696E766F69636573120472656164120577726974651A210A086D616361726F6F6E120867656E6572617465120472656164120577726974651A160A076D657373616765120472656164120577726974651A170A086F6666636861696E120472656164120577726974651A160A076F6E636861696E120472656164120577726974651A140A057065657273120472656164120577726974651A180A067369676E6572120867656E657261746512047265616400000620C4F9783E0873FA50A2091806F5EBB919C5DC432E33800B401463ADA6485DF0ED"
)

func getProtectedResource(c *gin.Context) {
	authField, authFieldIsPresent := c.Request.Header["Authorization"]

	// Check invalid tokens (macaroons)
	if !authFieldIsPresent || !helper.IsValidMacaroon(authField[0]) {
		// Generate invoice and token

		lndClient, err := lnd.NewLNDclient(lnd.LNDoptions{
			Address:     lndAddress,
			MacaroonHex: macaroonHex,
		})
		if err != nil {
			log.Fatalf("Failed to create LND client: %v", err)
		}

		ctx := context.Background()
		lnInvoice := lnrpc.Invoice{}

		lndInvoice, err := lndClient.AddInvoice(ctx, &lnInvoice)
		if err != nil {
			log.Fatalf("Failed to generate LND Invoice: %v", err)
		}

		macaroon := "AGIAJEemVQUTEyNCR0exk7ek90Cg=="
		invoice := lndInvoice.PaymentRequest

		c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroon, invoice))
		c.String(http.StatusPaymentRequired, "402 Payment Required")
	}
}

func main() {
	router := gin.Default()
	router.GET("/protected", getProtectedResource)

	router.Run("localhost:8080")
}
