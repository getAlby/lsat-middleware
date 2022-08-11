# Gin-LSAT

A middleware for [Gin](https://github.com/gin-gonic/gin) framework that uses [LSAT](https://lsat.tech/) (a protocol standard for authentication and paid APIs) and provides handler functions to accept microtransactions before serving ad-free content or any paid APIs.

## Installation

Assuming you've installed Go and Gin 

1. Run this:

```
go get github.com/getAlby/gin-lsat
```

2. Create `.env` file (refer `.env_example`) and configure `LND_ADDRESS` and `MACAROON_HEX` for LND client or `LNURL_ADDRESS` for LNURL client, `LN_CLIENT_TYPE` (out of LND, LNURL) and `ROOT_KEY` (for minting macaroons).  

## Usage

[This example](https://github.com/getAlby/gin-lsat/blob/main/examples/main.go) shows how to use Gin-LSAT for serving simple JSON response:-

```
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
	fr := &FiatRateConfig{
		Currency: "USD",
		Amount:   0.01,
	}
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c *gin.Context) {
		c.Request = FiatToBTC("USD", 0.01)
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
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
	})

	router.Run("localhost:8080")
}
```

[This repo](https://github.com/getAlby/lsat-proxy) demonstrates serving of static files and creating a paywall for paid resources using Gin-LSAT middleware.
## Testing

Run `go test` to run tests.
