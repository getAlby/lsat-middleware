package ginlsat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"proxy/ln"
	"proxy/lsat"
	"proxy/macaroon"
	macaroonutils "proxy/macaroon"
	"proxy/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

const (
	LSAT_TYPE_FREE = "FREE"
	LSAT_TYPE_PAID = "PAID"
)

type LsatInfo struct {
	Type     string
	Preimage lntypes.Preimage
	Mac      *macaroon.MacaroonIdentifier
	Amount   int64
	Error    error
}

type GinLsatMiddleware struct {
	Amount   int64
	Response func(c *gin.Context, code int, message string)
	LNClient ln.LNClient
}

func NewLsatMiddleware(mw *GinLsatMiddleware) (*GinLsatMiddleware, error) {
	mw.Response = func(c *gin.Context, code int, message string) {
		c.JSON(code, gin.H{
			"code":    code,
			"message": message,
		})
	}
	return mw, nil
}

func InitLnClient() (ln.LNClient, error) {
	var lnClient ln.LNClient
	err := godotenv.Load(".env")
	if err != nil {
		return lnClient, errors.New("Failed to load .env file")
	}

	switch os.Getenv("LN_CLIENT_TYPE") {
	case LND_CLIENT_TYPE:
		lnClient, err = ln.NewLNDclient(ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		})
		if err != nil {
			return lnClient, fmt.Errorf("Error initializing LN client: %s", err.Error())
		}
	case LNURL_CLIENT_TYPE:
		lnClient = &ln.LNURLWrapper{
			Address: os.Getenv("LNURL_ADDRESS"),
		}
	default:
		return lnClient, fmt.Errorf("LN Client type not recognized: %s", os.Getenv("LN_CLIENT_TYPE"))
	}

	return lnClient, nil
}

func (lsatmiddleware *GinLsatMiddleware) Handler(c *gin.Context) {

	acceptLsatField := c.Request.Header["Accept"]
	// Check if client support LSAT

	authField := c.Request.Header["Authorization"]
	mac, preimage, err := utils.ParseLsatHeader(authField)

	// If macaroon and preimage are valid
	if err == nil {
		rootKey := utils.GetRootKey()

		// Check valid LSAT and set LSAT type Paid
		err := lsat.VerifyLSAT(mac, rootKey[:], preimage)
		if err != nil {
			c.Set("LSAT", &LsatInfo{
				Type: LSAT_TYPE_PAID,
			})
		}
	} else if len(acceptLsatField) != 0 && acceptLsatField[0] == `application/vnd.lsat.v1.full+json` {
		// Generate invoice and token
		ctx := context.Background()
		lnInvoice := lnrpc.Invoice{
			Value: lsatmiddleware.Amount,
			Memo:  "LSAT",
		}
		LNClientConn := &ln.LNClientConn{
			LNClient: lsatmiddleware.LNClient,
		}
		invoice, paymentHash, err := LNClientConn.GenerateInvoice(ctx, lnInvoice)
		if err != nil {
			c.Error(err)
			c.Set("LSAT", &LsatInfo{
				Error: err,
			})
			return
		}
		macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash)
		if err != nil {
			c.Error(err)
			c.Set("LSAT", &LsatInfo{
				Error: err,
			})
			return
		}
		c.Writer.Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%v, invoice=%v", macaroonString, invoice))
		lsatmiddleware.Response(c, http.StatusPaymentRequired, "402 Payment Required")
		c.Abort()
	} else {
		// Set LSAT type Free if client does not support LSAT
		c.Set("LSAT", &LsatInfo{
			Type: LSAT_TYPE_FREE,
		})
	}

}
