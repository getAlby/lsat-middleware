package gin_lsat_middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"proxy/ln"
	"proxy/lsat"
	macaroonutils "proxy/macaroon"
	"proxy/service"
	"proxy/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/lightningnetwork/lnd/lnrpc"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

type GinLsatMiddleware struct {
	Response func(c *gin.Context, code int, message string)
	Service  *service.Service
}

func NewLsatMiddleware(mw *GinLsatMiddleware) (*GinLsatMiddleware, error) {
	lnClient, err := InitLnClient()
	if err != nil {
		return nil, err
	}
	svc := &service.Service{
		LnClient: lnClient,
	}
	mw.Service = svc
	if mw.Response == nil {
		mw.Response = func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		}
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

func (lsatmiddleware *GinLsatMiddleware) GetProtectedResource() gin.HandlerFunc {
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
				lsatmiddleware.Response(c, http.StatusAccepted, "Protected content")
				return
			}
		} else if len(acceptLsatField) != 0 && acceptLsatField[0] == `application/vnd.lsat.v1.full+json` {
			// Generate invoice and token
			ctx := context.Background()
			lnInvoice := lnrpc.Invoice{
				Value: 5,
				Memo:  "LSAT",
			}

			invoice, paymentHash, err := lsatmiddleware.Service.GenerateInvoice(ctx, lnInvoice)
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
			lsatmiddleware.Response(c, http.StatusPaymentRequired, "402 Payment Required")
			return
		} else {
			// Return Free content if client does not support LSAT
			lsatmiddleware.Response(c, http.StatusAccepted, "Free content")
			return
		}
	}
}
