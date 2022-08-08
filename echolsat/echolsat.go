package echolsat

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/getAlby/echo-lsat/ln"
	"github.com/getAlby/echo-lsat/lsat"
	"github.com/getAlby/echo-lsat/macaroon"
	macaroonutils "github.com/getAlby/echo-lsat/macaroon"
	"github.com/getAlby/echo-lsat/utils"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
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

const (
	FREE_CONTENT_MESSAGE      = "Free Content"
	PROTECTED_CONTENT_MESSAGE = "Protected Content"
	PAYMENT_REQUIRED_MESSAGE  = "Payment Required"
)

type LsatInfo struct {
	Type     string
	Preimage lntypes.Preimage
	Mac      *macaroon.MacaroonIdentifier
	Amount   int64
	Error    error
}

type EchoLsatMiddleware struct {
	AmountFunc func(req *http.Request) (amount int64)
	LNClient   ln.LNClient
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountFunc func(req *http.Request) (amount int64)) (*EchoLsatMiddleware, error) {
	lnClient, err := InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &EchoLsatMiddleware{
		AmountFunc: amountFunc,
		LNClient:   lnClient,
	}
	return middleware, nil
}

func InitLnClient(lnClientConfig *ln.LNClientConfig) (ln.LNClient, error) {
	var lnClient ln.LNClient
	err := godotenv.Load(".env")
	if err != nil {
		return lnClient, errors.New("Failed to load .env file")
	}

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

func (lsatmiddleware *EchoLsatMiddleware) Handler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		acceptLsatField := c.Request().Header["Accept"]
		// Check if client support LSAT

		authField := c.Request().Header["Authorization"]
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
				Value: lsatmiddleware.AmountFunc(c.Echo().AcquireContext().Request()),
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
			}
			macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash)
			if err != nil {
				c.Error(err)
				c.Set("LSAT", &LsatInfo{
					Error: err,
				})
			}
			c.Response().Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%s, invoice=%s", macaroonString, invoice))
			c.JSON(http.StatusPaymentRequired, map[string]interface{}{
				"code":    http.StatusPaymentRequired,
				"message": PAYMENT_REQUIRED_MESSAGE,
			})
			return nil
		} else {
			// Set LSAT type Free if client does not support LSAT
			c.Set("LSAT", &LsatInfo{
				Type: LSAT_TYPE_FREE,
			})
		}
		return next(c)
	}
}
