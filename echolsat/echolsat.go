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
	"github.com/getAlby/lsat-middleware/middleware"
	"github.com/getAlby/lsat-middleware/utils"
	"github.com/lightningnetwork/lnd/lnrpc"

	"github.com/labstack/echo/v4"
)

type EchoLsat struct {
	Middleware middleware.LsatMiddleware
}

func (lsatmiddleware *EchoLsat) Handler(next echo.HandlerFunc) echo.HandlerFunc {
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
		requestPathCaveat := caveat.NewCaveat(middleware.REQUEST_PATH, requestPath)
		caveats := append(lsatmiddleware.Middleware.Caveats, requestPathCaveat)
		//LSAT Header is present, verify it
		err = lsat.VerifyLSAT(mac, caveats, lsatmiddleware.Middleware.RootKey, preimage)
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

func (lsatmiddleware *EchoLsat) SetLSATHeader(c echo.Context) {
	// Generate invoice and token
	ctx := context.Background()
	lnInvoice := lnrpc.Invoice{
		Value: lsatmiddleware.Middleware.AmountFunc(c.Echo().AcquireContext().Request()),
		Memo:  "LSAT",
	}
	LNClientConn := &ln.LNClientConn{
		LNClient: lsatmiddleware.Middleware.LNClient,
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
	requestPathCaveat := caveat.NewCaveat(middleware.REQUEST_PATH, requestPath)
	caveats := append(lsatmiddleware.Middleware.Caveats, requestPathCaveat)
	macaroonString, err := macaroonutils.GetMacaroonAsString(paymentHash, caveats, lsatmiddleware.Middleware.RootKey)
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
