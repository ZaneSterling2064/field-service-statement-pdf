package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	statement "github.com/example/field-service-statement"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		slog.Error("INFRAI_API_KEY is required")
		os.Exit(2)
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = statement.DefaultBaseURL
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	client := statement.NewInfraiClient(baseURL, apiKey, &http.Client{Timeout: 20 * time.Second})
	server := &http.Server{
		Addr:              addr,
		Handler:           statement.NewService(client, logger).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	logger.Info("statement service listening", "addr", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
