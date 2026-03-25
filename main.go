package main

import (
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/config"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/db"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/handler"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/repository"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/service"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func setupRouter(linksHandler *handler.LinksHandler) *gin.Engine {
	router := gin.Default()
	router.Use(sentrygin.New(sentrygin.Options{}))

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	api := router.Group("/api")
	{
		api.GET("/links", linksHandler.ListLinks)
		api.POST("/links", linksHandler.CreateLink)
		api.GET("/links/:id", linksHandler.GetLink)
		api.PUT("/links/:id", linksHandler.UpdateLink)
		api.DELETE("/links/:id", linksHandler.DeleteLink)
	}

	return router
}

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	if cfg.SentryDSN != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: cfg.SentryDSN,
		})
		if err != nil {
			log.Printf("sentry init failed: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	dbConn, err := repository.OpenDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := dbConn.Close(); err != nil {
			log.Printf("failed to close db connection: %v", err)
		}
	}()

	queries := db.New(dbConn)
	linksService := service.NewLinksService(queries, cfg.AppBaseURL)
	linksHandler := handler.NewLinksHandler(linksService)

	router := setupRouter(linksHandler)

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}
