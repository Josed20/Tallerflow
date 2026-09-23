package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpx.NewRouter(httpx.Dependencies{}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("tallerflow API listening on %s", cfg.HTTPAddress)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve API: %v", err)
	}
}
