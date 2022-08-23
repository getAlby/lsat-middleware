package echolsat

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/getAlby/echo-lsat/echolsat"
	"github.com/getAlby/echo-lsat/ln"
	macaroonutils "github.com/getAlby/echo-lsat/macaroon"
	"github.com/getAlby/echo-lsat/utils"

	"github.com/labstack/echo/v4"

	"github.com/appleboy/gofight/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

const SATS_PER_BTC = 100000000

const MIN_SATS_TO_BE_PAID = 1

const (
	TEST_MACAROON_VALID = "AgEETFNBVALmAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAa/+CAiD/pv/jOjY1/9oC/4z/tHb/qf/2Jf+d/4H/u/+YGHj/+/+O/8D/v/+P/8X/qRL/5v/x/4r/tkIBIA1Y/8j/pR3/0P+b/7cwWP+W/87/sD18GP//Hf/f/9Aj//NcBFs2/9VhNEUF/70AAAAGIDlR1jVm5IfEJgvuSQoJLqLg4FcW4Ib1vW8sbkRHdUWX"
	TEST_PREIMAGE_VALID = "651505fae9ea341c770c6ebef207d8560d546eb3aee26985e584c15d1c987875"

	TEST_MACAROON_INVALID = "AgEETFNBVALhAUr/gQMBARJNYWNhcm9vbklkZW50aWZpZXIB/4IAAQMBB1ZlcnNpb24BBgABC1BheW1lbnRIYXNoAf+EAAEHVG9rZW5JZAH/hgAAABT/gwEBAQRIYXNoAf+EAAEGAUAAABn/hQEBAQlbMzJddWludDgB/4YAAQYBQAAAZv+CAiD/w/+g//R6HUgGQ///DUT/kP+z/7z/wv/9/4AEaX0P/4D/2Gt+/+3/uyEFA0H/8AEg/8f/y0ci/646FkX/vP+YB//kQwH/sUv/wV4KNv+9//Fn/4//7P/WLv/sLP/t/7YxAAAABiB5JhudEoBr8dWuC+BYG4xCl9D90NSwmW8NMw5heQxK+A=="
	TEST_PREIMAGE_INVALID = "fbe9ac25c04e14b10177514e2d57b0e39224e70277ac1a2cd23c28e58cd4ea35"
)

func FiatToBTC(currency string, value float64) *http.Request {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://blockchain.info/tobtc?currency=%s&value=%f", currency, value), nil)
	if err != nil {
		return nil
	}
	return req
}

type FiatRateConfig struct {
	Currency string
	Amount   float64
}

func (fr *FiatRateConfig) FiatToBTCAmountFunc(req *http.Request) (amount int64) {
	if req == nil {
		return MIN_SATS_TO_BE_PAID
	}
	res, err := http.Get(fmt.Sprintf("https://blockchain.info/tobtc?currency=%s&value=%f", fr.Currency, fr.Amount))
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	defer res.Body.Close()

	amountBits, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	amountInBTC, err := strconv.ParseFloat(string(amountBits), 32)
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
	amountInSats := SATS_PER_BTC * amountInBTC
	return int64(amountInSats)
}

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

func TestLsatWithLNURLConfig(t *testing.T) {
	err := godotenv.Load(".env")
	assert.NoError(t, err)

	LNURL_ADDRESS := os.Getenv("LNURL_ADDRESS")

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LNURL",
		LNURLConfig: ln.LNURLoptions{
			Address: LNURL_ADDRESS,
		},
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
			"Accept": "application/vnd.lsat.v1.full+json",
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
			macaroonId, _ := macaroonutils.GetPreimageFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
func TestLsatWithLNDConfig(t *testing.T) {
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
			"Accept": "application/vnd.lsat.v1.full+json",
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
			macaroonId, _ := macaroonutils.GetPreimageFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
