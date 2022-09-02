package ginlsat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getAlby/lsat-middleware/caveat"
	"github.com/getAlby/lsat-middleware/ln"
	"github.com/getAlby/lsat-middleware/lsat"
	macaroonutils "github.com/getAlby/lsat-middleware/macaroon"
	"github.com/getAlby/lsat-middleware/middleware"
	"github.com/getAlby/lsat-middleware/utils"
	"github.com/lightningnetwork/lnd/lnrpc"

	"github.com/gin-gonic/gin"
)

type GinLsat struct {
	Middleware middleware.LsatMiddleware
}

func (lsatmiddleware *GinLsat) Handler(c *gin.Context) {
	//First check for presence of authorization header
	authField := c.Request.Header.Get("Authorization")
	mac, preimage, err := utils.ParseLsatHeader(authField)
	caveats := []caveat.Caveat{}
	if lsatmiddleware.Middleware.CaveatFunc != nil {
		caveats = lsatmiddleware.Middleware.CaveatFunc(c.Request)
	}
	if err != nil {
		// No Authorization present, check if client supports LSAT
		acceptLsatField := c.Request.Header.Get(lsat.LSAT_HEADER_NAME)
		if strings.Contains(acceptLsatField, lsat.LSAT_HEADER) {
			lsatmiddleware.SetLSATHeader(c, caveats)
			return
		}
		// Set LSAT type Free if client does not support LSAT
		c.Set("LSAT", &lsat.LsatInfo{
			Type: lsat.LSAT_TYPE_FREE,
		})
		return
	}
	//LSAT Header is present, verify it
	err = lsat.VerifyLSAT(mac, caveats, lsatmiddleware.Middleware.RootKey, preimage)
	if err != nil {
		//not a valid LSAT
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	//LSAT verification ok, mark client as having paid
	macaroonId, err := macaroonutils.GetMacIdFromMacaroon(mac)
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	c.Set("LSAT", &lsat.LsatInfo{
		Type:        lsat.LSAT_TYPE_PAID,
		Preimage:    preimage,
		PaymentHash: macaroonId.PaymentHash,
	})

}

func (lsatmiddleware *GinLsat) SetLSATHeader(c *gin.Context, caveats []caveat.Caveat) {
	// Generate invoice and token
	ctx := context.Background()
	lnInvoice := lnrpc.Invoice{
		Value: lsatmiddleware.Middleware.AmountFunc(c.Request),
		Memo:  "LSAT",
	}
	LNClientConn := &ln.LNClientConn{
		LNClient: lsatmiddleware.Middleware.LNClient,
	}
	invoice, paymentHash, err := LNClientConn.GenerateInvoice(ctx, lnInvoice, c.Request)
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash, caveats, lsatmiddleware.Middleware.RootKey)
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%s, invoice=%s", macaroonString, invoice))
	c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
		"code":    http.StatusPaymentRequired,
		"message": lsat.PAYMENT_REQUIRED_MESSAGE,
	})
}
