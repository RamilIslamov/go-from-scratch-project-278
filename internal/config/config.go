package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	AppBaseURL  string
	SentryDSN   string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		appBaseURL = "http://localhost:" + port
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		AppBaseURL:  appBaseURL,
		SentryDSN:   os.Getenv("SENTRY_DSN"),
	}
}
