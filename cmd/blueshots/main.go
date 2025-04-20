package main

import (
	"blueshorts/internal/config"
	"blueshorts/internal/imap"
	"blueshorts/internal/server"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg, err := config.Load("/data/config.toml")
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	fetch := imap.NewFetcher(imap.Config{
		Host:     cfg.IMAP.Host,
		Port:     cfg.IMAP.Port,
		Username: cfg.IMAP.Username,
		Password: cfg.IMAP.Password,
	})

	srv := server.New(server.Options{
		APIKey: cfg.Server.APIKey,
		Feeds:  cfg.Feeds,
		Fetch:  fetch,
		TTL:    5 * time.Minute,
	})

	log.Fatal(http.ListenAndServe(":8080", srv))
}
