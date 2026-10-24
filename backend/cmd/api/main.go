package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// ginMode keeps production logs free of Gin's debug banner and route list,
// which it prints in its default mode.
func ginMode(environment string) string {
	if environment == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	gin.SetMode(ginMode(cfg.Environment))
	app, err := buildApplication(cfg)
	if err != nil {
		return fmt.Errorf("build application: %w", err)
	}
	defer func() {
		if err := app.close(); err != nil {
			log.Printf("close application: %v", err)
		}
	}()

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           app.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serveErrors := make(chan error, 1)
	go func() { serveErrors <- server.ListenAndServe() }()
	log.Printf("tallerflow API listening on %s", cfg.HTTPAddress)
	select {
	case err := <-serveErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve API: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown API: %w", err)
	}
	if err := <-serveErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve API: %w", err)
	}
	return nil
}
