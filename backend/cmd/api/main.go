package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "new-pay/docs" // This is for Swagger
	"new-pay/internal/auth"
	"new-pay/internal/config"
	"new-pay/internal/database"
	"new-pay/internal/email"
	"new-pay/internal/handlers"
	"new-pay/internal/logger"
	"new-pay/internal/middleware"
	"new-pay/internal/repository"
	"new-pay/internal/service"

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
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Setup structured logger
	logger.Setup(logger.Config{
		Level: cfg.Log.Level,
	})

	slog.Info("Starting application",
		"name", cfg.App.Name,
		"version", cfg.App.Version,
		"env", cfg.App.Env,
		"log_level", cfg.Log.Level,
	)

	// Initialize database
	db, err := database.New(&cfg.Database)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	slog.Info("Database connection established")

	// Run database migrations
	migrator := database.NewMigrationExecutor(db.DB)
	if err := migrator.RunMigrations("./migrations"); err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("Database migrations completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	roleRepo := repository.NewRoleRepository(db.DB)
	tokenRepo := repository.NewTokenRepository(db.DB)
	sessionRepo := repository.NewSessionRepository(db.DB)
	auditRepo := repository.NewAuditRepository(db.DB)
	oauthConnRepo := repository.NewOAuthConnectionRepository(db.DB)
	catalogRepo := repository.NewCatalogRepository(db.DB)
	selfAssessmentRepo := repository.NewSelfAssessmentRepository(db.DB)

	// Initialize services
	authService := auth.NewService(&cfg.JWT)
	emailService := email.NewService(&cfg.Email)
	authSvc := service.NewAuthService(userRepo, tokenRepo, roleRepo, sessionRepo, oauthConnRepo, authService, emailService)
	catalogService := service.NewCatalogService(catalogRepo, selfAssessmentRepo, auditRepo)
	selfAssessmentService := service.NewSelfAssessmentService(selfAssessmentRepo, catalogRepo, auditRepo)

	// Initialize middleware
	authMw := middleware.NewAuthMiddleware(authService, sessionRepo, userRepo)
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
	catalogHandler := handlers.NewCatalogHandler(catalogService, auditMw)
	selfAssessmentHandler := handlers.NewSelfAssessmentHandler(selfAssessmentService)

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
	mux.Handle("/api/v1/users/password/change", authMw.Authenticate(http.HandlerFunc(userHandler.ChangePassword)))
	mux.Handle("/api/v1/users/resend-verification", authMw.Authenticate(http.HandlerFunc(userHandler.ResendVerificationEmail)))
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
	mux.Handle("/api/v1/admin/users/send-verification",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.AdminSendVerificationEmail),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/cancel-verification",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.AdminCancelVerification),
			),
		),
	)
	mux.Handle("/api/v1/admin/users/revoke-verification",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.AdminRevokeVerification),
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

	// Catalog routes - Public (users can view catalogs in review phase)
	mux.Handle("GET /api/v1/catalogs", authMw.Authenticate(http.HandlerFunc(catalogHandler.GetAllCatalogs)))
	mux.Handle("GET /api/v1/catalogs/{id}", authMw.Authenticate(http.HandlerFunc(catalogHandler.GetCatalogByID)))

	// Catalog routes - Admin only
	mux.Handle("POST /api/v1/admin/catalogs",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.CreateCatalog),
			),
		),
	)
	mux.Handle("PUT /api/v1/admin/catalogs/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.UpdateCatalog),
			),
		),
	)
	mux.Handle("DELETE /api/v1/admin/catalogs/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.DeleteCatalog),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/transition-to-active",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.TransitionToActive),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/transition-to-archived",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.TransitionToArchived),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/categories",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.CreateCategory),
			),
		),
	)
	mux.Handle("PUT /api/v1/admin/catalogs/{id}/categories/{categoryId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.UpdateCategory),
			),
		),
	)
	mux.Handle("DELETE /api/v1/admin/catalogs/{id}/categories/{categoryId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.DeleteCategory),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/levels",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.CreateLevel),
			),
		),
	)
	mux.Handle("PUT /api/v1/admin/catalogs/{id}/levels/{levelId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.UpdateLevel),
			),
		),
	)
	mux.Handle("DELETE /api/v1/admin/catalogs/{id}/levels/{levelId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.DeleteLevel),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/categories/{categoryId}/paths",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.CreatePath),
			),
		),
	)
	mux.Handle("PUT /api/v1/admin/catalogs/{id}/categories/{categoryId}/paths/{pathId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.UpdatePath),
			),
		),
	)
	mux.Handle("DELETE /api/v1/admin/catalogs/{id}/categories/{categoryId}/paths/{pathId}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.DeletePath),
			),
		),
	)
	mux.Handle("POST /api/v1/admin/catalogs/{id}/descriptions",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.CreateOrUpdateDescription),
			),
		),
	)
	mux.Handle("GET /api/v1/admin/catalogs/{id}/changes",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.GetChanges),
			),
		),
	)

	// Self-Assessment routes
	// Get active catalogs (available to all authenticated users)
	mux.Handle("GET /api/v1/self-assessments/active-catalogs",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.GetActiveCatalogs),
		),
	)
	// Create self-assessment for a catalog
	mux.Handle("POST /api/v1/self-assessments/catalog/{catalogId}",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.CreateSelfAssessment),
		),
	)
	// Get current user's self-assessments
	mux.Handle("GET /api/v1/self-assessments/my",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.GetUserSelfAssessments),
		),
	)
	// Get visible self-assessments (role-based)
	mux.Handle("GET /api/v1/self-assessments",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.GetVisibleSelfAssessments),
		),
	)
	// Get specific self-assessment
	mux.Handle("GET /api/v1/self-assessments/{id}",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.GetSelfAssessment),
		),
	)
	// Update self-assessment status
	mux.Handle("PUT /api/v1/self-assessments/{id}/status",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.UpdateStatus),
		),
	)

	// Admin routes for self-assessments
	mux.Handle("GET /api/v1/admin/self-assessments",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.GetAllSelfAssessmentsAdmin),
		),
	)
	mux.Handle("DELETE /api/v1/admin/self-assessments/{id}",
		authMw.Authenticate(
			http.HandlerFunc(selfAssessmentHandler.DeleteSelfAssessment),
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
	handler := middleware.LoggingMiddleware(
		middleware.SecurityHeaders(
			corsMw.Handler(
				rateLimiter.Limit(mux),
			),
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
		slog.Info("Server starting", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Server shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := getContext(30 * time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
