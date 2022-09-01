package echolsat

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
	"github.com/lightningnetwork/lnd/lnrpc"

	"github.com/labstack/echo/v4"
)

const REQUEST_PATH = "RequestPath"

type EchoLsatMiddleware struct {
	AmountFunc func(req *http.Request) (amount int64)
	LNClient   ln.LNClient
	Caveats    []caveat.Caveat
	RootKey    []byte
}

func NewLsatMiddleware(lnClientConfig *ln.LNClientConfig,
	amountFunc func(req *http.Request) (amount int64)) (*EchoLsatMiddleware, error) {
	lnClient, err := ln.InitLnClient(lnClientConfig)
	if err != nil {
		return nil, err
	}
	middleware := &EchoLsatMiddleware{
		AmountFunc: amountFunc,
		LNClient:   lnClient,
		Caveats:    lnClientConfig.Caveats,
		RootKey:    lnClientConfig.RootKey,
	}
	return middleware, nil
}

func (lsatmiddleware *EchoLsatMiddleware) Handler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//First check for presence of authorization header
		authField := c.Request().Header.Get("Authorization")
		mac, preimage, err := utils.ParseLsatHeader(authField)
		if err != nil {
			// No Authorization present, check if client supports LSAT

			acceptLsatField := c.Request().Header.Get(lsat.LSAT_HEADER_NAME)

			if strings.Contains(acceptLsatField, lsat.LSAT_HEADER) {
				lsatmiddleware.SetLSATHeader(c)
				return nil
			}
			// Set LSAT type Free if client does not support LSAT
			c.Set("LSAT", &lsat.LsatInfo{
				Type: lsat.LSAT_TYPE_FREE,
			})
			return next(c)
		}
		requestPath := c.Request().URL.Path
		requestPathCaveat := caveat.NewCaveat(REQUEST_PATH, requestPath)
		caveats := append(lsatmiddleware.Caveats, requestPathCaveat)
		//LSAT Header is present, verify it
		err = lsat.VerifyLSAT(mac, caveats, lsatmiddleware.RootKey, preimage)
		if err != nil {
			//not a valid LSAT
			c.Set("LSAT", &lsat.LsatInfo{
				Type:  lsat.LSAT_TYPE_ERROR,
				Error: err,
			})
			return next(c)
		}
		//LSAT verification ok, mark client as having paid
		macaroonId, err := macaroonutils.GetMacIdFromMacaroon(mac)
		c.Set("LSAT", &lsat.LsatInfo{
			Type:        lsat.LSAT_TYPE_PAID,
			Preimage:    preimage,
			PaymentHash: macaroonId.PaymentHash,
		})
		return next(c)
	}
}

func (lsatmiddleware *EchoLsatMiddleware) SetLSATHeader(c echo.Context) {
	// Generate invoice and token
	ctx := context.Background()
	lnInvoice := lnrpc.Invoice{
		Value: lsatmiddleware.AmountFunc(c.Echo().AcquireContext().Request()),
		Memo:  "LSAT",
	}
	LNClientConn := &ln.LNClientConn{
		LNClient: lsatmiddleware.LNClient,
	}
	invoice, paymentHash, err := LNClientConn.GenerateInvoice(ctx, lnInvoice, c.Echo().AcquireContext().Request())
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	requestPath := c.Request().URL.Path
	requestPathCaveat := caveat.NewCaveat(REQUEST_PATH, requestPath)
	caveats := append(lsatmiddleware.Caveats, requestPathCaveat)
	macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash, caveats, lsatmiddleware.RootKey)
	if err != nil {
		c.Set("LSAT", &lsat.LsatInfo{
			Type:  lsat.LSAT_TYPE_ERROR,
			Error: err,
		})
		return
	}
	c.Response().Header().Set("WWW-Authenticate", fmt.Sprintf("LSAT macaroon=%s, invoice=%s", macaroonString, invoice))
	c.JSON(http.StatusPaymentRequired, map[string]interface{}{
		"code":    http.StatusPaymentRequired,
		"message": lsat.PAYMENT_REQUIRED_MESSAGE,
	})
}
