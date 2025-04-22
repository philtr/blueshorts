package main

import (
	"blueshorts/internal/config"
	"blueshorts/internal/imap"
	"blueshorts/internal/server"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	configPath := getEnv("BLUESHORTS_CONFIG", "./config.toml")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	fetch := imap.NewFetcher(imap.Config{
		Host:     getEnv("BLUESHORTS_IMAP_HOST", cfg.IMAP.Host),
		Port:     getEnvInt("BLUESHORTS_IMAP_PORT", cfg.IMAP.Port),
		Username: getEnv("BLUESHORTS_IMAP_USER", cfg.IMAP.Username),
		Password: getEnv("BLUESHORTS_IMAP_PASS", cfg.IMAP.Password),
	})

	srv := server.New(server.Options{
		APIKey: getEnv("BLUESHORTS_SERVER_API_KEY", cfg.Server.APIKey),
		Feeds:  cfg.Feeds,
		Fetch:  fetch,
		TTL:    5 * time.Minute,
	})

	port := getEnvInt("BLUESHORTS_SERVER_PORT", cfg.Server.Port)
	addr := ":" + strconv.Itoa(port)
	log.Printf("starting blueshorts server, listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if intval, err := strconv.Atoi(val); err == nil {
			return intval
		}
	}

	return fallback
}
