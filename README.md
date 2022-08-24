# LSAT-Middleware

A middleware library for [Gin](https://github.com/gin-gonic/gin) and [Echo](https://echo.labstack.com/) framework that uses [LSAT](https://lsat.tech/) (a protocol standard for authentication and paid APIs) and provides handler functions to accept microtransactions before serving ad-free content or any paid APIs.

The middleware:-

1. Checks the preference of the user whether they need paid content or free content.
2. Verify the LSAT before serving paid content.
3. Send macaroon and invoice if the user prefers paid content and fails to present a valid LSAT.

<img src="https://user-images.githubusercontent.com/44242169/186485336-83b0dc68-aae5-4bdc-aee5-33046b9204c7.png" width="700">


## Installation

Assuming you've installed Go and Gin 

1. Run this:

```
go get github.com/getAlby/lsat-middleware
```

2. Create `.env` file (refer `.env_example`) and configure `LND_ADDRESS` and `MACAROON_HEX` for LND client or `LNURL_ADDRESS` for LNURL client, `LN_CLIENT_TYPE` (out of LND, LNURL) and `ROOT_KEY` (for minting macaroons).  

## Usage

[This example](https://github.com/getAlby/lsat-middleware/blob/main/examples/ginlsat/main.go) shows how to use LSAT-Middleware with [Gin](https://github.com/gin-gonic/gin) framework for serving simple JSON response:-

```
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/getAlby/lsat-middleware/ginlsat"
	"github.com/getAlby/lsat-middleware/ln"
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

	err := godotenv.Load("../../.env")
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
		} else if lsatInfo.Type == ginlsat.LSAT_TYPE_ERROR {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    http.StatusInternalServerError,
				"message": fmt.Sprint(lsatInfo.Error),
			})
		}
	})

	router.Run("localhost:8080")
}
```


[This example](https://github.com/getAlby/lsat-middleware/blob/main/examples/echolsat/main.go) shows how to use LSAT-Middleware with [Echo](https://echo.labstack.com/) framework for serving simple JSON response:-

```
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/getAlby/lsat-middleware/echolsat"
	"github.com/getAlby/lsat-middleware/ln"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
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
	router := echo.New()

	router.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusAccepted, map[string]interface{}{
			"code":    http.StatusAccepted,
			"message": "Free content",
		})
	})

	err := godotenv.Load("../../.env")
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
	lsatmiddleware, err := echolsat.NewLsatMiddleware(lnClientConfig, fr.FiatToBTCAmountFunc)
	if err != nil {
		log.Fatal(err)
	}

	router.Use(lsatmiddleware.Handler)

	router.GET("/protected", func(c echo.Context) error {
		lsatInfo := c.Get("LSAT").(*echolsat.LsatInfo)
		if lsatInfo.Type == echolsat.LSAT_TYPE_FREE {
			return c.JSON(http.StatusAccepted, map[string]interface{}{
				"code":    http.StatusAccepted,
				"message": "Free content",
			})
		}
		if lsatInfo.Type == echolsat.LSAT_TYPE_PAID {
			return c.JSON(http.StatusAccepted, map[string]interface{}{
				"code":    http.StatusAccepted,
				"message": "Protected content",
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

	router.Start("localhost:8080")
}
```

[This repo](https://github.com/getAlby/lsat-proxy) demonstrates serving of static files and creating a paywall for paid resources using LSAT-Middleware.

[InSATgram](https://insatgram.getalby.com/), A platform to buy and sell images for sats made using LSAT-Middleware ([link to repo](https://github.com/getAlby/sell-lsat-files)).


## Testing

Run `go test` to run tests.
