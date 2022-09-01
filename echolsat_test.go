package test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/getAlby/lsat-middleware/echolsat"
	"github.com/getAlby/lsat-middleware/ln"
	macaroonutils "github.com/getAlby/lsat-middleware/macaroon"
	"github.com/getAlby/lsat-middleware/utils"

	"github.com/labstack/echo/v4"

	"github.com/appleboy/gofight/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func echoLsatHandler(lsatmiddleware *echolsat.EchoLsatMiddleware) *echo.Echo {
	router := echo.New()

	router.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"code":    http.StatusAccepted,
			"message": echolsat.FREE_CONTENT_MESSAGE,
		})
	})

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c echo.Context) error {
		lsatInfo := c.Get("LSAT").(*echolsat.LsatInfo)
		if lsatInfo.Type == echolsat.LSAT_TYPE_FREE {
			return c.JSON(http.StatusAccepted, map[string]interface{}{
				"code":    http.StatusAccepted,
				"message": echolsat.FREE_CONTENT_MESSAGE,
			})
		}
		if lsatInfo.Type == echolsat.LSAT_TYPE_PAID {
			return c.JSON(http.StatusAccepted, map[string]interface{}{
				"code":    http.StatusAccepted,
				"message": echolsat.PROTECTED_CONTENT_MESSAGE,
			})
		}
		if lsatInfo.Type == echolsat.LSAT_TYPE_ERROR {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
		return nil
	})

	return router
}

func TestEchoLsatWithLNURLConfig(t *testing.T) {

	err := godotenv.Load(".env")
	assert.NoError(t, err)

	LNURL_ADDRESS := os.Getenv("LNURL_ADDRESS")

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LNURL",
		LNURLConfig: ln.LNURLoptions{
			Address: LNURL_ADDRESS,
		},
		RootKey: []byte(ROOT_KEY),
	}
	fr := &FiatRateConfig{
		Currency: "USD",
		Amount:   0.01,
	}
	lsatmiddleware, err := echolsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	assert.NoError(t, err)

	handler := echoLsatHandler(lsatmiddleware)

	router := gofight.New()

	router.GET("/").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			echolsat.LSAT_HEADER_NAME: echolsat.LSAT_HEADER,
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.PAYMENT_REQUIRED_MESSAGE, message)
			assert.Equal(t, http.StatusPaymentRequired, res.Code)

			assert.True(t, strings.HasPrefix(res.HeaderMap.Get("Www-Authenticate"), "LSAT macaroon="))
			assert.True(t, strings.Contains(res.HeaderMap.Get("Www-Authenticate"), "invoice="))
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_VALID, TEST_PREIMAGE_VALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_INVALID, TEST_PREIMAGE_INVALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(TEST_MACAROON_INVALID)

			macaroonId, _ := macaroonutils.GetMacIdFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}

func TestEchoLsatWithLNDConfig(t *testing.T) {

	err := godotenv.Load(".env")
	assert.NoError(t, err)

	LND_ADDRESS := os.Getenv("LND_ADDRESS")
	MACAROON_HEX := os.Getenv("MACAROON_HEX")

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LND",
		LNDConfig: ln.LNDoptions{
			Address:     LND_ADDRESS,
			MacaroonHex: MACAROON_HEX,
		},
		RootKey: []byte(ROOT_KEY),
	}
	fr := &FiatRateConfig{
		Currency: "USD",
		Amount:   0.01,
	}
	lsatmiddleware, err := echolsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	assert.NoError(t, err)

	handler := echoLsatHandler(lsatmiddleware)

	router := gofight.New()

	router.GET("/").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			echolsat.LSAT_HEADER_NAME: echolsat.LSAT_HEADER,
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.PAYMENT_REQUIRED_MESSAGE, message)
			assert.Equal(t, http.StatusPaymentRequired, res.Code)

			assert.True(t, strings.HasPrefix(res.HeaderMap.Get("Www-Authenticate"), "LSAT macaroon="))
			assert.True(t, strings.Contains(res.HeaderMap.Get("Www-Authenticate"), "invoice="))
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_VALID, TEST_PREIMAGE_VALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, echolsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_INVALID, TEST_PREIMAGE_INVALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(TEST_MACAROON_INVALID)

			macaroonId, _ := macaroonutils.GetMacIdFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
