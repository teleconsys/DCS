package main

import (
	"os"

	"github.com/gin-gonic/gin"

	"github.com/teleconsys/DCS/internal/config"
)

func main() {
	// Load environment variables from .env file
	config.LoadEnv()

	// Initialize router
	router := gin.Default()

	// Get groundcontrol info enbdpoint
	router.GET("/get-groundcontrol-info", func(c *gin.Context) {

		privateKey := os.Getenv("GC_PRIVATE_KEY")
		address := os.Getenv("GC_ADDRESS")
		walletGasId := os.Getenv("GC_WALLET_GAS_ID")

		if privateKey == "" || address == "" || walletGasId == "" {
			c.JSON(400, gin.H{
				"error": "GC_PRIVATE_KEY, GC_ADDRESS, or GC_WALLET_GAS_ID is not set",
			})
			return
		}

		c.JSON(200, gin.H{
			"privateKey":    privateKey,
			"address":        address,
			"walletGasId":  walletGasId,
		})
	})

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	router.Run(":" + port)
};
