package main

import (
	"fmt"
	"log"
	"os"
	"proxy/ln"
	"proxy/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	LND_CLIENT_TYPE   = "LND"
	LNURL_CLIENT_TYPE = "LNURL"
)

func main() {
	router := gin.Default()

	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Failed to load .env file")
	}
	var lnClient ln.LNClient
	switch os.Getenv("LN_CLIENT_TYPE") {
	case LND_CLIENT_TYPE:
		lnClient, err = ln.NewLNDclient(ln.LNDoptions{
			Address:     os.Getenv("LND_ADDRESS"),
			MacaroonHex: os.Getenv("MACAROON_HEX"),
		})
		if err != nil {
			log.Fatalf("Error initializing LN client: %s", err.Error())
		}
	case LNURL_CLIENT_TYPE:
		lnClient = &ln.LNURLWrapper{
			Address: os.Getenv("LNURL_ADDRESS"),
		}
	default:
		log.Fatalf("LN Client type not recognized: %s", os.Getenv("LN_CLIENT_TYPE"))
	}

	svc := &service.Service{
		LnClient: lnClient,
	}

	router.GET("/protected", svc.GetProtectedResource())

	router.Run("localhost:8080")
}
