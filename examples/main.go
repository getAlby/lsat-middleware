package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/getAlby/gin-lsat/ginlsat"
	"github.com/getAlby/gin-lsat/ln"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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

func AmountFunc(req *http.Request) (amount int64) {
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return MIN_SATS_TO_BE_PAID
	}
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

func main() {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    http.StatusAccepted,
			"message": "Free content",
		})
	})

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Failed to load .env file")
	}
	lnClientConfig := &ln.LNClientConfig{
		LNClientType: os.Getenv("LN_CLIENT_TYPE"),
		LNDConfig: ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		},
		LNURLConfig: ln.LNURLoptions{
			Address: os.Getenv("LNURL_ADDRESS"),
		},
	}
	req := FiatToBTC("USD", 0.01)
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(lnClientConfig, AmountFunc, req)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c *gin.Context) {
		lsatInfo := c.Value("LSAT").(*ginlsat.LsatInfo)
		if lsatInfo.Type == ginlsat.LSAT_TYPE_FREE {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Free content",
			})
		} else if lsatInfo.Type == ginlsat.LSAT_TYPE_PAID {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Protected content",
			})
		} else {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
	})

	router.Run("localhost:8080")
}
