package test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/getAlby/lsat-middleware/caveat"
	"github.com/getAlby/lsat-middleware/ginlsat"
	"github.com/getAlby/lsat-middleware/ln"
	"github.com/getAlby/lsat-middleware/lsat"
	macaroonutils "github.com/getAlby/lsat-middleware/macaroon"
	"github.com/getAlby/lsat-middleware/utils"

	"github.com/appleboy/gofight/v2"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func ginLsatHandler(lsatmiddleware *ginlsat.GinLsatMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    http.StatusAccepted,
			"message": lsat.FREE_CONTENT_MESSAGE,
		})
	})

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c *gin.Context) {
		lsatInfo := c.Value("LSAT").(*lsat.LsatInfo)
		if lsatInfo.Type == lsat.LSAT_TYPE_FREE {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": lsat.FREE_CONTENT_MESSAGE,
			})
		} else if lsatInfo.Type == lsat.LSAT_TYPE_PAID {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": lsat.PROTECTED_CONTENT_MESSAGE,
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

func TestGinLsatWithLNURLConfig(t *testing.T) {
	err := godotenv.Load(".env")
	assert.NoError(t, err)

	LNURL_ADDRESS := os.Getenv("LNURL_ADDRESS")

	lnClientConfig := &ln.LNClientConfig{
		LNClientType: "LNURL",
		LNURLConfig: ln.LNURLoptions{
			Address: LNURL_ADDRESS,
		},
		Caveats: []caveat.Caveat{
			{
				Condition: BASE_URL,
				Value:     os.Getenv("BASE_URL"),
			},
		},
		RootKey: []byte(ROOT_KEY),
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

			assert.Equal(t, lsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, lsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			lsat.LSAT_HEADER_NAME: lsat.LSAT_HEADER,
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, lsat.PAYMENT_REQUIRED_MESSAGE, message)
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

			assert.Equal(t, lsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_VALID, TEST_PREIMAGE_INVALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(TEST_MACAROON_VALID)
			macaroonId, _ := macaroonutils.GetMacIdFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_WITHOUT_CAVEATS, TEST_MACAROON_WITHOUT_CAVEATS_PREIMAGE),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, fmt.Sprintf("Caveats does not match"), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}

func TestGinLsatWithLNDConfig(t *testing.T) {
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
		Caveats: []caveat.Caveat{
			{
				Condition: BASE_URL,
				Value:     os.Getenv("BASE_URL"),
			},
		},
		RootKey: []byte(ROOT_KEY),
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

			assert.Equal(t, lsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, lsat.FREE_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			lsat.LSAT_HEADER_NAME: lsat.LSAT_HEADER,
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, lsat.PAYMENT_REQUIRED_MESSAGE, message)
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

			assert.Equal(t, lsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_VALID, TEST_PREIMAGE_INVALID),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			macaroon, _ := utils.GetMacaroonFromString(TEST_MACAROON_VALID)
			macaroonId, _ := macaroonutils.GetMacIdFromMacaroon(macaroon)

			assert.Equal(t, fmt.Sprintf("Invalid Preimage %s for PaymentHash %s", TEST_PREIMAGE_INVALID, macaroonId.PaymentHash), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})

	router.GET("/protected").
		SetHeader(gofight.H{
			"Authorization": fmt.Sprintf("LSAT %s:%s", TEST_MACAROON_WITHOUT_CAVEATS, TEST_MACAROON_WITHOUT_CAVEATS_PREIMAGE),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, fmt.Sprintf("Caveats does not match"), message)
			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
}
