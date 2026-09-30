package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/internal/passwordreset"
	"github.com/Josed20/Tallerflow/backend/internal/sprint2"
	"github.com/Josed20/Tallerflow/backend/internal/workshops"
	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type application struct {
	handler http.Handler
	ping    func(context.Context) error
	close   func() error
}

type applicationDependencies struct {
	openDatabase func(string) (*gorm.DB, error)
}

type applicationOption func(*applicationDependencies)

func withDatabaseOpener(open func(string) (*gorm.DB, error)) applicationOption {
	return func(dependencies *applicationDependencies) {
		if open != nil {
			dependencies.openDatabase = open
		}
	}
}

func buildApplication(cfg config.Config, options ...applicationOption) (*application, error) {
	dependencies := applicationDependencies{openDatabase: database.Open}
	for _, option := range options {
		if option != nil {
			option(&dependencies)
		}
	}

	db, err := dependencies.openDatabase(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open runtime database: %w", err)
	}
	closeDatabase := onceClose(func() error { return database.Close(db) })
	failed := true
	defer func() {
		if failed {
			_ = closeDatabase()
		}
	}()

	repository, err := auth.NewPostgresRepository(db, time.Now)
	if err != nil {
		return nil, fmt.Errorf("create auth repository: %w", err)
	}
	tenantRunner, err := database.NewTenantRunner(db)
	if err != nil {
		return nil, fmt.Errorf("create tenant runner: %w", err)
	}
	workshopService, err := workshops.NewService(tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create workshop service: %w", err)
	}
	sessions := auth.NewSessionService(repository, []byte(cfg.SessionPepper), time.Now, nil)
	authService := auth.NewAuthService(repository, repository, sessions, auth.NewPasswordHasher(auth.DefaultPasswordParams()), repository)
	authHandler, err := auth.NewHandler(authService, auth.HandlerConfig{
		AllowedOrigin: cfg.AllowedOrigin,
		Environment:   cfg.Environment,
		Secure:        cfg.Environment != "development",
	})
	if err != nil {
		return nil, fmt.Errorf("create auth handler: %w", err)
	}
	workshopHandler := workshops.NewHandler(workshopService)
	var bootstrapDB *gorm.DB
	if cfg.BootstrapDatabaseURL != "" {
		bootstrapDB, err = dependencies.openDatabase(cfg.BootstrapDatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("open onboarding database: %w", err)
		}
	}
	sprintHandler, err := sprint2.New(db, bootstrapDB, tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create sprint 2 handler: %w", err)
	}
	resetDelivery, err := passwordreset.NewSMTPDelivery(passwordreset.SMTPConfig{Address: cfg.SMTPAddress, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, UseTLS: cfg.SMTPTLS})
	if err != nil {
		return nil, fmt.Errorf("create password reset delivery: %w", err)
	}
	resetHandler, err := passwordreset.New(db, resetDelivery, cfg.PasswordResetURL)
	if err != nil {
		return nil, fmt.Errorf("create password reset handler: %w", err)
	}
	ping := func(ctx context.Context) error { return database.Ping(ctx, db) }
	router := httpx.NewRouter(httpx.Dependencies{
		Ping: ping,
		Routes: []httpx.RouteRegistrar{
			func(routes gin.IRouter) { auth.RegisterRoutes(routes, authHandler) },
			func(routes gin.IRouter) { passwordreset.RegisterRoutes(routes, resetHandler) },
			func(routes gin.IRouter) {
				workshops.RegisterRoutes(routes, workshopHandler, authHandler.RequireSession())
			},
			func(routes gin.IRouter) {
				sprint2.RegisterRoutes(routes, sprintHandler, authHandler.RequireSession(), authHandler.RequireMutation())
			},
		},
	})
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("TF_TRUSTED_PROXIES is invalid")
	}

	failed = false
	closeAll := onceClose(func() error {
		if bootstrapDB != nil {
			_ = database.Close(bootstrapDB)
		}
		return closeDatabase()
	})
	return &application{handler: router, ping: ping, close: closeAll}, nil
}

func onceClose(closeDatabase func() error) func() error {
	var once sync.Once
	var closeErr error
	return func() error {
		once.Do(func() { closeErr = closeDatabase() })
		return closeErr
	}
}
