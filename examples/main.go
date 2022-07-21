package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/DhananjayPurohit/gin-lsat/ginlsat"
	"github.com/DhananjayPurohit/gin-lsat/ln"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

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
	lnClient, err := ginlsat.InitLnClient(&ln.LNClientConfig{
		LNClientType: os.Getenv("LN_CLIENT_TYPE"),
		LNDConfig: ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		},
		LNURLConfig: ln.LNURLoptions{
			Address: os.Getenv("LNURL_ADDRESS"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	lsatmiddleware, err := ginlsat.NewLsatMiddleware(&ginlsat.GinLsatMiddleware{
		Amount:   5,
		LNClient: lnClient,
	})
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
