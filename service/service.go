package service

import (
	"context"
	"fmt"
	"net/http"
	"proxy/ln"
	"proxy/lsat"
	macaroonutils "proxy/macaroon"
	"proxy/utils"

	"github.com/gin-gonic/gin"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lntypes"
)

type Service struct {
	LnClient ln.LNClient
}

func (svc *Service) GenerateInvoice(ctx context.Context, lnInvoice lnrpc.Invoice) (string, lntypes.Hash, error) {
	lnClientInvoice, err := svc.LnClient.AddInvoice(ctx, &lnInvoice)
	if err != nil {
		return "", lntypes.Hash{}, err
	}

	invoice := lnClientInvoice.PaymentRequest
	paymentHash, err := lntypes.MakeHash(lnClientInvoice.RHash)
	if err != nil {
		return invoice, lntypes.Hash{}, err
	}
	return invoice, paymentHash, nil
}

func (svc *Service) GetProtectedResource() gin.HandlerFunc {
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

			invoice, paymentHash, err := svc.GenerateInvoice(ctx, lnInvoice)
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
