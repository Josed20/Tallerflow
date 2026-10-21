package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/internal/clients"
	platformconsole "github.com/Josed20/Tallerflow/backend/internal/console"
	"github.com/Josed20/Tallerflow/backend/internal/onboarding"
	"github.com/Josed20/Tallerflow/backend/internal/orders"
	"github.com/Josed20/Tallerflow/backend/internal/passwordreset"
	"github.com/Josed20/Tallerflow/backend/internal/team"
	"github.com/Josed20/Tallerflow/backend/internal/workshops"
	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/mail"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type application struct {
	handler http.Handler
	ping    func(context.Context) error
	close   func() error
}

type applicationDependencies struct {
	openDatabase          func(string) (*gorm.DB, error)
	buildOnboardingModule onboardingModuleBuilder
}

type onboardingModule struct {
	service onboarding.UseCases
	ping    func(context.Context) error
	close   func() error
}

type onboardingModuleBuilder func(string, *auth.SessionService) (onboardingModule, error)

type applicationOption func(*applicationDependencies)

func withDatabaseOpener(open func(string) (*gorm.DB, error)) applicationOption {
	return func(dependencies *applicationDependencies) {
		if open != nil {
			dependencies.openDatabase = open
		}
	}
}

func withOnboardingModuleBuilder(build onboardingModuleBuilder) applicationOption {
	return func(dependencies *applicationDependencies) {
		if build != nil {
			dependencies.buildOnboardingModule = build
		}
	}
}

func buildApplication(cfg config.Config, options ...applicationOption) (*application, error) {
	dependencies := applicationDependencies{openDatabase: database.Open, buildOnboardingModule: buildPostgresOnboardingModule}
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
	teamRepository, err := team.NewPostgresRepository(db, tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create team repository: %w", err)
	}
	sessions := auth.NewSessionService(repository, []byte(cfg.SessionPepper), time.Now, nil)
	onboardingModule, err := dependencies.buildOnboardingModule(cfg.BootstrapDatabaseURL, sessions)
	if err != nil {
		return nil, fmt.Errorf("create onboarding module: %w", err)
	}
	closeOnboarding := onceClose(onboardingModule.close)
	var closeConsole = func() error { return nil }
	var consoleRegistrar httpx.RouteRegistrar
	var consoleService *platformconsole.Service
	if cfg.ConsoleDatabaseURL != "" {
		consoleDB, openErr := dependencies.openDatabase(cfg.ConsoleDatabaseURL)
		if openErr != nil {
			return nil, fmt.Errorf("open console database: %w", openErr)
		}
		closeConsole = onceClose(func() error { return database.Close(consoleDB) })
		var serviceErr error
		consoleService, serviceErr = platformconsole.NewService(consoleDB, cfg.ConsoleTOTPKey)
		if serviceErr != nil {
			_ = closeConsole()
			return nil, fmt.Errorf("create console service: %w", serviceErr)
		}
		consoleHandler := platformconsole.NewHandler(consoleService, cfg.ConsoleOrigin, cfg.Environment != "development")
		consoleRegistrar = func(routes gin.IRouter) { platformconsole.RegisterRoutes(routes, consoleHandler) }
	}
	closeApplication := onceClose(func() error {
		return errors.Join(closeConsole(), closeOnboarding(), closeDatabase())
	})
	defer func() {
		if failed {
			_ = closeApplication()
		}
	}()
	authService := auth.NewAuthService(repository, repository, sessions, auth.NewPasswordHasher(auth.DefaultPasswordParams()), repository)
	passwordResetRepository, err := passwordreset.NewPostgresRepository(db, time.Now)
	if err != nil {
		return nil, fmt.Errorf("create password reset repository: %w", err)
	}
	passwordResetDelivery, err := passwordreset.NewSMTPDelivery(passwordreset.SMTPDeliveryConfig{
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		FromAddress: cfg.SMTPFromAddress,
		RequireTLS:  cfg.SMTPRequireTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("create password reset delivery: %w", err)
	}
	mailSender, err := mail.NewSMTPSender(mail.SMTPConfig{
		Host:        cfg.SMTPHost,
		Port:        cfg.SMTPPort,
		Username:    cfg.SMTPUsername,
		Password:    cfg.SMTPPassword,
		FromAddress: cfg.SMTPFromAddress,
		RequireTLS:  cfg.SMTPRequireTLS,
	})
	if err != nil {
		return nil, fmt.Errorf("create mail sender: %w", err)
	}
	if consoleService != nil {
		consoleService.SetMailer(mailSender)
	}
	passwordResetService := passwordreset.NewService(
		passwordResetRepository,
		passwordResetDelivery,
		auth.NewPasswordHasher(auth.DefaultPasswordParams()),
		passwordreset.ServiceConfig{BaseURL: cfg.PasswordResetBaseURL},
		time.Now,
	)
	authHandler, err := auth.NewHandler(authService, auth.HandlerConfig{
		AllowedOrigin: cfg.AllowedOrigin,
		MobileOrigins: cfg.MobileOrigins,
		Environment:   cfg.Environment,
		Secure:        cfg.Environment != "development",
	})
	if err != nil {
		return nil, fmt.Errorf("create auth handler: %w", err)
	}
	onboardingHandler, err := onboarding.NewHandler(onboardingModule.service, onboarding.HandlerConfig{
		AllowedOrigin: cfg.AllowedOrigin, Environment: cfg.Environment, Secure: cfg.Environment != "development",
		Limiter: security.NewMemoryRateLimiter([]byte("onboarding-rate-limit:"+cfg.SessionPepper), time.Now, 15*time.Minute, 30, 10000),
	})
	if err != nil {
		return nil, fmt.Errorf("create onboarding handler: %w", err)
	}
	workshopHandler := workshops.NewHandler(workshopService)
	teamService, err := team.NewService(teamRepository, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte(cfg.SessionPepper), cfg.AllowedOrigin, time.Now, nil, mailSender)
	if err != nil {
		return nil, fmt.Errorf("create team service: %w", err)
	}
	teamHandler, err := team.NewHandler(teamService, cfg.AllowedOrigin, []byte("team-invitation-consume-rate-limit:"+cfg.SessionPepper))
	if err != nil {
		return nil, fmt.Errorf("create team handler: %w", err)
	}
	passwordResetHandler := passwordreset.NewHandler(passwordResetService)
	clientsRepo, err := clients.NewPostgresRepository(tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create clients repository: %w", err)
	}
	clientsService := clients.NewService(clientsRepo)
	clientsHandler := clients.NewHandler(clientsService)

	ordersRepo, err := orders.NewPostgresRepository(tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create orders repository: %w", err)
	}
	ordersService := orders.NewService(ordersRepo)
	ordersHandler := orders.NewHandler(ordersService)
	trackingRepo, err := orders.NewPostgresTrackingRepository(db, tenantRunner)
	if err != nil {
		return nil, fmt.Errorf("create tracking repository: %w", err)
	}
	trackingHandler := orders.NewTrackingHandler(orders.NewTrackingService(trackingRepo, nil))

	ping := func(ctx context.Context) error {
		if err := database.Ping(ctx, db); err != nil {
			return err
		}
		return onboardingModule.ping(ctx)
	}
	router := httpx.NewRouter(httpx.Dependencies{
		Ping:       ping,
		Middleware: []gin.HandlerFunc{security.MobileCORS(cfg.MobileOrigins)},
		Routes: []httpx.RouteRegistrar{
			func(routes gin.IRouter) { onboarding.RegisterRoutes(routes, onboardingHandler) },
			func(routes gin.IRouter) { auth.RegisterRoutes(routes, authHandler) },
			func(routes gin.IRouter) { passwordreset.RegisterRoutes(routes, passwordResetHandler) },
			func(routes gin.IRouter) {
				workshops.RegisterRoutes(routes, workshopHandler, authHandler.RequireSession())
			},
			func(routes gin.IRouter) {
				team.RegisterRoutes(routes, teamHandler, authHandler.RequireSession())
			},
			func(routes gin.IRouter) {
				clients.RegisterRoutes(routes, clientsHandler, authHandler.RequireSession())
			},
			func(routes gin.IRouter) {
				orders.RegisterRoutes(routes, ordersHandler, authHandler.RequireSession())
			},
			func(routes gin.IRouter) {
				orders.RegisterTrackingRoutes(routes, trackingHandler, authHandler.RequireSession())
			},
		},
	})
	if consoleRegistrar != nil {
		consoleRegistrar(router)
	}
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("TF_TRUSTED_PROXIES is invalid")
	}

	failed = false
	return &application{handler: router, ping: ping, close: closeApplication}, nil
}

func buildPostgresOnboardingModule(databaseURL string, sessions *auth.SessionService) (onboardingModule, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return onboardingModule{}, errors.New("TF_BOOTSTRAP_DATABASE_URL is invalid")
	}
	bootstrap := auth.NewBootstrapService(auth.NewPostgresBootstrapStore(pool), auth.NewPasswordHasher(auth.DefaultPasswordParams()))
	creator := onboarding.OwnerCreatorFunc(func(ctx context.Context, input auth.BootstrapInput, metadata auth.SessionMetadata) (auth.WebBootstrapResult, error) {
		return bootstrap.CreateWebOwner(ctx, input, metadata, sessions)
	})
	service := onboarding.NewService(onboarding.NewPostgresStatusStore(pool), creator)
	return onboardingModule{service: service, ping: pool.Ping, close: func() error { pool.Close(); return nil }}, nil
}

func onceClose(closeDatabase func() error) func() error {
	var once sync.Once
	var closeErr error
	return func() error {
		once.Do(func() { closeErr = closeDatabase() })
		return closeErr
	}
}
