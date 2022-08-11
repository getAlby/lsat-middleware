package ginlsat

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/getAlby/gin-lsat/ginlsat"
	"github.com/getAlby/gin-lsat/ln"
	macaroonutils "github.com/getAlby/gin-lsat/macaroon"
	"github.com/getAlby/gin-lsat/utils"

	"github.com/appleboy/gofight/v2"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

const SATS_PER_BTC = 100000000

const MIN_SATS_TO_BE_PAID = 1

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

func ginLsatHandler(lsatmiddleware *ginlsat.GinLsatMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    http.StatusAccepted,
			"message": ginlsat.FREE_CONTENT_MESSAGE,
		})
	})

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c *gin.Context) {
		lsatInfo := c.Value("LSAT").(*ginlsat.LsatInfo)
		if lsatInfo.Type == ginlsat.LSAT_TYPE_FREE {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": ginlsat.FREE_CONTENT_MESSAGE,
			})
		} else if lsatInfo.Type == ginlsat.LSAT_TYPE_PAID {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": ginlsat.PROTECTED_CONTENT_MESSAGE,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
	})

	return router
}

func TestLsatWithLNURLConfig(t *testing.T) {
	err := godotenv.Load(".env")
	assert.NoError(t, err)

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LNURL",
		LNURLConfig: ln.LNURLoptions{
			Address: os.Getenv("LNURL_ADDRESS"),
		},
	}
	fr := &FiatRateConfig{
		Currency: "USD",
		Amount:   0.01,
	}
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	assert.NoError(t, err)

	handler := ginLsatHandler(lsatmiddleware)

	router := gofight.New()

	router.GET("/").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Accept": "application/vnd.lsat.v1.full+json",
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PAYMENT_REQUIRED_MESSAGE, message)
			assert.Equal(t, http.StatusPaymentRequired, res.Code)

			assert.True(t, strings.HasPrefix(res.HeaderMap.Get("Www-Authenticate"), "LSAT macaroon="))
			assert.True(t, strings.Contains(res.HeaderMap.Get("Www-Authenticate"), "invoice="))
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON_VALID"), os.Getenv("TEST_PREIMAGE_VALID")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON_INVALID"), os.Getenv("TEST_PREIMAGE_INVALID")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(os.Getenv("TEST_MACAROON_INVALID"))
			macaroonId, _ := macaroonutils.GetPreimageFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", os.Getenv("TEST_PREIMAGE_INVALID"), macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
func TestLsatWithLNDConfig(t *testing.T) {
	err := godotenv.Load(".env")
	assert.NoError(t, err)

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LND",
		LNDConfig: ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		},
	}
	fr := &FiatRateConfig{
		Currency: "USD",
		Amount:   0.01,
	}
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	assert.NoError(t, err)

	handler := ginLsatHandler(lsatmiddleware)

	router := gofight.New()

	router.GET("/").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Accept": "application/vnd.lsat.v1.full+json",
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PAYMENT_REQUIRED_MESSAGE, message)
			assert.Equal(t, http.StatusPaymentRequired, res.Code)

			assert.True(t, strings.HasPrefix(res.HeaderMap.Get("Www-Authenticate"), "LSAT macaroon="))
			assert.True(t, strings.Contains(res.HeaderMap.Get("Www-Authenticate"), "invoice="))
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON_VALID"), os.Getenv("TEST_PREIMAGE_VALID")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON_INVALID"), os.Getenv("TEST_PREIMAGE_INVALID")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(os.Getenv("TEST_MACAROON_INVALID"))
			macaroonId, _ := macaroonutils.GetPreimageFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", os.Getenv("TEST_PREIMAGE_INVALID"), macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
