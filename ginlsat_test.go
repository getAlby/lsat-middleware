package ginlsat

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/getAlby/gin-lsat/ginlsat"
	"github.com/getAlby/gin-lsat/ln"

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
			c.JSON(http.StatusAccepted, gin.H{
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

	lnClient, err := ginlsat.InitLnClient(&ln.LNClientConfig{
		LNClientType: "LNURL",
		LNURLConfig: ln.LNURLoptions{
			Address: os.Getenv("LNURL_ADDRESS"),
		},
	})
	assert.NoError(t, err)

	lsatmiddleware, err := ginlsat.NewLsatMiddleware(&ginlsat.GinLsatMiddleware{
		Amount:   5,
		LNClient: lnClient,
	})

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
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON"), os.Getenv("TEST_PREIMAGE")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})
}
func TestLsatWithLNDConfig(t *testing.T) {
	err := godotenv.Load(".env")
	assert.NoError(t, err)

	lnClient, err := ginlsat.InitLnClient(&ln.LNClientConfig{
		LNClientType: "LND",
		LNDConfig: ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		},
	})
	assert.NoError(t, err)

	lsatmiddleware, err := ginlsat.NewLsatMiddleware(&ginlsat.GinLsatMiddleware{
		Amount:   5,
		LNClient: lnClient,
	})

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
			"Authorization": fmt.Sprintf("LSAT %s:%s", os.Getenv("TEST_MACAROON"), os.Getenv("TEST_PREIMAGE")),
		}).
		Run(handler, func(res gofight.HTTPResponse, req gofight.HTTPRequest) {
			message := fmt.Sprint(gjson.Get(res.Body.String(), "message"))

			assert.Equal(t, ginlsat.PROTECTED_CONTENT_MESSAGE, message)
			assert.Equal(t, http.StatusAccepted, res.Code)
		})
}
