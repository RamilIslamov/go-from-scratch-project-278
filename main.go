package main

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(sentrygin.New(sentrygin.Options{}))

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	return router
}

func main() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
		})
		if err != nil {
			log.Printf("sentry init failed: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	router := setupRouter()

	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
