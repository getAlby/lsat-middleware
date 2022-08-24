package ginlsat

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getAlby/gin-lsat/ln"
	"github.com/getAlby/gin-lsat/lsat"
	"github.com/getAlby/gin-lsat/macaroon"
	macaroonutils "github.com/getAlby/gin-lsat/macaroon"
	"github.com/getAlby/gin-lsat/utils"

	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

const (
	LSAT_TYPE_FREE   = "FREE"
	LSAT_TYPE_PAID   = "PAID"
	LSAT_TYPE_ERROR  = "ERROR"
	LSAT_HEADER      = "LSAT"
	LSAT_HEADER_NAME = "Accept-Authenticate"
)

const (
	FREE_CONTENT_MESSAGE      = "Free Content"
	PROTECTED_CONTENT_MESSAGE = "Protected Content"
	PAYMENT_REQUIRED_MESSAGE  = "Payment Required"
)

type LsatInfo struct {
	Type        string
	Preimage    lntypes.Preimage
	Mac         *macaroon.MacaroonIdentifier
	PaymentHash lntypes.Hash
	Amount      int64
	Error       error
}

type GinLsatMiddleware struct {
	AmountFunc func(req *http.Request) (amount int64)
	LNClient   ln.LNClient
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountFunc func(req *http.Request) (amount int64)) (*GinLsatMiddleware, error) {
	lnClient, err := InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &GinLsatMiddleware{
		AmountFunc: amountFunc,
		LNClient:   lnClient,
	}
	return middleware, nil
}

func InitLnClient(lnClientConfig *ln.LNClientConfig) (lnClient ln.LNClient, err error) {
	switch lnClientConfig.LNClientType {
	case LND_CLIENT_TYPE:
		lnClient, err = ln.NewLNDclient(lnClientConfig.LNDConfig)
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	case LNURL_CLIENT_TYPE:
		lnClient, err = ln.NewLNURLClient(lnClientConfig.LNURLConfig)
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	default:
		return lnClient, fmt.Errorf("LN Client type not recognized: %s", lnClientConfig.LNClientType)
	}

	return lnClient, nil
}

func (lsatmiddleware *GinLsatMiddleware) Handler(c *gin.Context) {
	//First check for presence of authorization header
	authField := c.Request.Header.Get("Authorization")
	mac, preimage, err := utils.ParseLsatHeader(authField)
	if err != nil {
		// No Authorization present, check if client supports LSAT
		acceptLsatField := c.Request.Header.Get(LSAT_HEADER_NAME)
		if strings.Contains(acceptLsatField, LSAT_HEADER) {
			lsatmiddleware.SetLSATHeader(c)
			return
		}
		// Set LSAT type Free if client does not support LSAT
		c.Set("LSAT", &LsatInfo{
			Type: LSAT_TYPE_FREE,
		})
		return
	}
	//LSAT Header is present, verify it
	err = lsat.VerifyLSAT(mac, utils.GetRootKey(), preimage)
	if err != nil {
		//not a valid LSAT
		c.Set("LSAT", &LsatInfo{
			Type:  LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	//LSAT verification ok, mark client as having paid
	macaroonId, err := macaroonutils.GetMacIdFromMacaroon(mac)
	c.Set("LSAT", &LsatInfo{
		Type:        LSAT_TYPE_PAID,
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
		c.Set("LSAT", &LsatInfo{
			Type:  LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash)
	if err != nil {
		c.Set("LSAT", &LsatInfo{
			Type:  LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%s, invoice=%s", macaroonString, invoice))
	c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
		"code":    http.StatusPaymentRequired,
		"message": PAYMENT_REQUIRED_MESSAGE,
	})
}
