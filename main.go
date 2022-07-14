package main

import (
	"log"
	"net/http"
	"proxy/gin_lsat_middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    http.StatusAccepted,
			"message": "Free content",
		})
	})

	lsatmiddleware, err := gin_lsat_middleware.NewLsatMiddleware(&gin_lsat_middleware.GinLsatMiddleware{})
	if err != nil {
		log.Fatal(err)
	}

	router.Use(lsatmiddleware.GetProtectedResource())

	router.GET("/protected", func(c *gin.Context) {
		lsatStatus := c.GetString("LSAT")
		if lsatStatus == "Free" {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Free content",
			})
		} else if lsatStatus == "Paid" {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusAccepted,
				"message": "Protected content",
			})
		} else {
			c.JSON(http.StatusAccepted, gin.H{
				"code":    http.StatusInternalServerError,
				"message": lsatStatus,
			})
		}
	})

	router.Run("localhost:8080")
}
