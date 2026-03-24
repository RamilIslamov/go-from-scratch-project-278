package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	return router
}

func main() {
	router := setupRouter()

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
