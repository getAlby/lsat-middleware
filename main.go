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

	router.Run("localhost:8080")
}
