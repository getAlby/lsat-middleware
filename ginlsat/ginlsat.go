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
	"github.com/getAlby/lsat-middleware/utils"

	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
)

type GinLsatMiddleware struct {
	AmountFunc func(req *http.Request) (amount int64)
	LNClient   ln.LNClient
	Caveats    []caveat.Caveat
	RootKey    []byte
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountFunc func(req *http.Request) (amount int64)) (*GinLsatMiddleware, error) {
	lnClient, err := ln.InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &GinLsatMiddleware{
		AmountFunc: amountFunc,
		LNClient:   lnClient,
		Caveats:    lnClientConfig.Caveats,
		RootKey:    lnClientConfig.RootKey,
	}
	return middleware, nil
}

func (lsatmiddleware *GinLsatMiddleware) Handler(c *gin.Context) {
	//First check for presence of authorization header
	authField := c.Request.Header.Get("Authorization")
	mac, preimage, err := utils.ParseLsatHeader(authField)
	if err != nil {
		// No Authorization present, check if client supports LSAT
		acceptLsatField := c.Request.Header.Get(lsat.LSAT_HEADER_NAME)
		if strings.Contains(acceptLsatField, lsat.LSAT_HEADER) {
			lsatmiddleware.SetLSATHeader(c)
			return
		}
		// Set LSAT type Free if client does not support LSAT
		c.Set("LSAT", &lsat.LsatInfo{
			Type: lsat.LSAT_TYPE_FREE,
		})
		return
	}
	//LSAT Header is present, verify it
	err = lsat.VerifyLSAT(mac, lsatmiddleware.Caveats, lsatmiddleware.RootKey, preimage)
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
	c.Set("LSAT", &lsat.LsatInfo{
		Type:        lsat.LSAT_TYPE_PAID,
		Preimage:    preimage,
		PaymentHash: macaroonId.PaymentHash,
	})

}

func (lsatmiddleware *GinLsatMiddleware) SetLSATHeader(c *gin.Context) {
	// Generate invoice and token
	ctx := context.Background()
	lnInvoice := lnrpc.Invoice{
		Value: lsatmiddleware.AmountFunc(c.Request),
		Memo:  "LSAT",
	}
	LNClientConn := &ln.LNClientConn{
		LNClient: lsatmiddleware.LNClient,
	}
	invoice, paymentHash, err := LNClientConn.GenerateInvoice(ctx, lnInvoice, c.Request)
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash, lsatmiddleware.Caveats, lsatmiddleware.RootKey)
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
