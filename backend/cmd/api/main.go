package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/pwannenmacher/New-Pay/docs" // This is for Swagger
	"github.com/pwannenmacher/New-Pay/internal/auth"
	"github.com/pwannenmacher/New-Pay/internal/config"
	"github.com/pwannenmacher/New-Pay/internal/database"
	"github.com/pwannenmacher/New-Pay/internal/email"
	"github.com/pwannenmacher/New-Pay/internal/handlers"
	"github.com/pwannenmacher/New-Pay/internal/middleware"
	"github.com/pwannenmacher/New-Pay/internal/repository"
	"github.com/pwannenmacher/New-Pay/internal/service"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title New Pay API
// @version 1.0
// @description Backend API for New Pay salary estimation and peer review platform
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@newpay.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Database connection established")

	// Run database migrations
	migrator := database.NewMigrationExecutor(db.DB)
	if err := migrator.RunMigrations("./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	tokenRepo := repository.NewTokenRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	auditRepo := repository.NewAuditRepository(db.DB)
	oauthConnRepo := repository.NewOAuthConnectionRepository(db.DB)

	// Initialize services
	authService := auth.NewService(&cfg.JWT)
	emailService := email.NewService(&cfg.Email)
	authSvc := service.NewAuthService(userRepo, tokenRepo, roleRepo, sessionRepo, oauthConnRepo, authService, emailService)

	// Initialize middleware
	authMw := middleware.NewAuthMiddleware(authService, sessionRepo)
	rbacMw := middleware.NewRBACMiddleware(db.DB)
	corsMw := middleware.NewCORSMiddleware(&cfg.CORS)
	rateLimiter := middleware.NewRateLimiter(&cfg.RateLimit)
	auditMw := middleware.NewAuditMiddleware(db.DB)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authSvc, auditMw, cfg)
	userHandler := handlers.NewUserHandler(userRepo, roleRepo, auditMw, authSvc)
	auditHandler := handlers.NewAuditHandler(auditRepo)
	sessionHandler := handlers.NewSessionHandler(sessionRepo, authSvc, auditMw, db.DB)
	configHandler := handlers.NewConfigHandler(cfg)

	// Setup router
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/v1/auth/verify-email", authHandler.VerifyEmail)
	mux.HandleFunc("/api/v1/auth/password-reset/request", authHandler.RequestPasswordReset)
	mux.HandleFunc("/api/v1/auth/password-reset/confirm", authHandler.ResetPassword)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.RefreshToken)

	// OAuth routes
	mux.HandleFunc("/api/v1/auth/oauth/login", authHandler.OAuthLogin)
	mux.HandleFunc("/api/v1/auth/oauth/callback", authHandler.OAuthCallback)

	// Config routes (public)
	mux.HandleFunc("/api/v1/config/oauth", configHandler.GetOAuthConfig)
	mux.HandleFunc("/api/v1/config/app", configHandler.GetAppConfig)

	// Protected routes
	mux.Handle("/api/v1/users/profile", authMw.Authenticate(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("/api/v1/users/profile/update", authMw.Authenticate(http.HandlerFunc(userHandler.UpdateProfile)))
	mux.Handle("/api/v1/users/sessions", authMw.Authenticate(http.HandlerFunc(sessionHandler.GetMySessions)))
	mux.Handle("/api/v1/users/sessions/delete", authMw.Authenticate(http.HandlerFunc(sessionHandler.DeleteMySession)))
	mux.Handle("/api/v1/users/sessions/delete-all", authMw.Authenticate(http.HandlerFunc(sessionHandler.DeleteAllMySessions)))

	// Admin routes
	mux.Handle("/api/v1/admin/users/get",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.GetUser),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/list",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.ListUsers),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/assign-role",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.AssignRole),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/remove-role",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.RemoveRole),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/update-status",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.UpdateUserActiveStatus),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/update",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.UpdateUser),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/set-password",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.SetUserPassword),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/delete",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.DeleteUser),
			),
		),
	)
	mux.Handle("/api/v1/admin/roles/list",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.ListRoles),
			),
		),
	)
	mux.Handle("/api/v1/admin/audit-logs/list",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(auditHandler.ListAuditLogs),
			),
		),
	)
	mux.Handle("/api/v1/admin/sessions",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(sessionHandler.GetAllSessions),
			),
		),
	)
	mux.Handle("/api/v1/admin/sessions/delete",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(sessionHandler.DeleteUserSession),
			),
		),
	)
	mux.Handle("/api/v1/admin/sessions/delete-all",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(sessionHandler.DeleteAllUserSessions),
			),
		),
	)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","database":"error"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","version":"` + cfg.App.Version + `"}`))
	})

	// Swagger documentation
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Apply global middleware
	handler := middleware.SecurityHeaders(
		corsMw.Handler(
			rateLimiter.Limit(mux),
		),
	)

	// Create server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.TimeoutRead,
		WriteTimeout: cfg.Server.TimeoutWrite,
		IdleTimeout:  cfg.Server.TimeoutIdle,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := getContext(30 * time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
