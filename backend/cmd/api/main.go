package main

import (
	"context"
	"errors"
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
	"new-pay/internal/keymanager"
	"new-pay/internal/logger"
	"new-pay/internal/middleware"
	"new-pay/internal/repository"
	"new-pay/internal/scheduler"
	"new-pay/internal/securestore"
	"new-pay/internal/service"
	"new-pay/internal/vault"

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
	defer func(db *database.Database) {
		err := db.Close()
		if err != nil {
			slog.Error("Failed to close database connection", "error", err)
		}
	}(db)

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
	assessmentResponseRepo := repository.NewAssessmentResponseRepository(db.DB)
	reviewerResponseRepo := repository.NewReviewerResponseRepository(db.DB)
	consolidationOverrideRepo := repository.NewConsolidationOverrideRepository(db.DB)
	consolidationOverrideApprovalRepo := repository.NewConsolidationOverrideApprovalRepository(db.DB)
	consolidationAveragedApprovalRepo := repository.NewConsolidationAveragedApprovalRepository(db.DB)
	finalConsolidationRepo := repository.NewFinalConsolidationRepository(db.DB)
	finalConsolidationApprovalRepo := repository.NewFinalConsolidationApprovalRepository(db.DB)
	discussionRepo := repository.NewDiscussionRepository(db.DB)

	// Initialize services
	authService := auth.NewService(&cfg.JWT)
	emailService := email.NewService(&cfg.Email)
	authSvc := service.NewAuthService(userRepo, tokenRepo, roleRepo, sessionRepo, oauthConnRepo, authService, emailService)
	catalogService := service.NewCatalogService(catalogRepo, selfAssessmentRepo, auditRepo, emailService)

	// Initialize encryption services (if Vault is enabled)
	var encryptedResponseSvc *service.EncryptedResponseService
	var reviewerService *service.ReviewerService
	var consolidationService *service.ConsolidationService
	var discussionService *service.DiscussionService
	if cfg.Vault.Enabled {
		slog.Info("Vault is enabled - initializing encryption services")
		vaultClient, err := vault.NewClient(&vault.Config{
			Address:      cfg.Vault.Address,
			Token:        cfg.Vault.Token,
			TransitMount: cfg.Vault.TransitMount,
		})
		if err != nil {
			slog.Error("Failed to initialize Vault client", "error", err)
			os.Exit(1)
		}

		keyManager, err := keymanager.NewKeyManager(db.DB, vaultClient)
		if err != nil {
			slog.Error("Failed to initialize KeyManager", "error", err)
			os.Exit(1)
		}

		secureStore := securestore.NewSecureStore(db.DB, keyManager)
		encryptedResponseSvc = service.NewEncryptedResponseService(db.DB, assessmentResponseRepo, keyManager, secureStore)
		reviewerService = service.NewReviewerService(db.DB, reviewerResponseRepo, selfAssessmentRepo, assessmentResponseRepo, keyManager, secureStore)
		consolidationService = service.NewConsolidationService(db.DB, consolidationOverrideRepo, consolidationOverrideApprovalRepo, consolidationAveragedApprovalRepo, finalConsolidationRepo, finalConsolidationApprovalRepo, selfAssessmentRepo, assessmentResponseRepo, reviewerResponseRepo, catalogRepo, encryptedResponseSvc, keyManager, secureStore, emailService)
		discussionService = service.NewDiscussionService(discussionRepo, selfAssessmentRepo, reviewerResponseRepo, assessmentResponseRepo, consolidationOverrideRepo, finalConsolidationRepo, catalogRepo, userRepo, secureStore)

		slog.Info("Encryption services initialized", "vault_addr", cfg.Vault.Address)
	} else {
		slog.Warn("Vault is disabled - encrypted responses will not work")
	}

	selfAssessmentService := service.NewSelfAssessmentService(selfAssessmentRepo, catalogRepo, auditRepo, assessmentResponseRepo, encryptedResponseSvc, reviewerResponseRepo)

	// Initialize scheduler
	schedulerService := scheduler.NewScheduler(selfAssessmentRepo, userRepo, roleRepo, emailService, &cfg.Scheduler)
	schedulerService.Start()
	defer schedulerService.Stop()

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
	reviewerHandler := handlers.NewReviewerHandler(reviewerService, selfAssessmentRepo, discussionService)
	consolidationHandler := handlers.NewConsolidationHandler(consolidationService)
	discussionHandler := handlers.NewDiscussionHandler(discussionService)

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
	mux.Handle("/api/v1/admin/users/create",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(userHandler.CreateUser),
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

	// Catalog routes - Accessible to users and reviewers (read-only access to catalog structure)
	mux.Handle("GET /api/v1/catalogs",
		authMw.Authenticate(
			rbacMw.RequireAnyRole("user", "reviewer")(
				http.HandlerFunc(catalogHandler.GetAllCatalogs),
			),
		),
	)
	mux.Handle("GET /api/v1/catalogs/{id}",
		authMw.Authenticate(
			rbacMw.RequireAnyRole("user", "reviewer")(
				http.HandlerFunc(catalogHandler.GetCatalogByID),
			),
		),
	)

	// Catalog routes - Admin only
	// Admin can list all catalogs without filtering
	mux.Handle("GET /api/v1/admin/catalogs",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.GetAllCatalogs),
			),
		),
	)
	mux.Handle("GET /api/v1/admin/catalogs/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.GetCatalogByID),
			),
		),
	)
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
	mux.Handle("PUT /api/v1/admin/catalogs/{id}/valid-until",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(catalogHandler.UpdateCatalogValidUntil),
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

	// Self-Assessment routes - Require user role only
	// Get active catalogs (available only to users with user role)
	mux.Handle("GET /api/v1/self-assessments/active-catalogs",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetActiveCatalogs),
			),
		),
	)
	// Get current user's self-assessments
	mux.Handle("GET /api/v1/self-assessments/my",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetUserSelfAssessments),
			),
		),
	)

	// Create self-assessment for a catalog
	mux.Handle("POST /api/v1/catalogs/{catalogId}/self-assessments",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.CreateSelfAssessment),
			),
		),
	)

	// Routes with {id}/subpath patterns
	// Get responses for an assessment
	mux.Handle("GET /api/v1/self-assessments/{id}/responses",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetResponses),
			),
		),
	)
	// Save or update a response
	mux.Handle("POST /api/v1/self-assessments/{id}/responses",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.SaveResponse),
			),
		),
	)
	// Delete a response
	mux.Handle("DELETE /api/v1/self-assessments/{id}/responses/{categoryId}",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.DeleteResponse),
			),
		),
	)
	// Get completeness status
	mux.Handle("GET /api/v1/self-assessments/{id}/completeness",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetCompleteness),
			),
		),
	)
	// Get weighted score
	mux.Handle("GET /api/v1/self-assessments/{id}/weighted-score",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetWeightedScore),
			),
		),
	)
	// Submit assessment for review
	mux.Handle("PUT /api/v1/self-assessments/{id}/submit",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.SubmitAssessment),
			),
		),
	)

	// Generic self-assessment routes
	// Get specific self-assessment
	mux.Handle("GET /api/v1/self-assessments/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(selfAssessmentHandler.GetSelfAssessment),
			),
		),
	)
	// Update self-assessment status (user can submit, admin can close)
	mux.Handle("PUT /api/v1/self-assessments/{id}/status",
		authMw.Authenticate(
			rbacMw.RequireAnyRole("admin", "user")(
				http.HandlerFunc(selfAssessmentHandler.UpdateStatus),
			),
		),
	)
	// Admin routes for self-assessments
	mux.Handle("GET /api/v1/admin/self-assessments",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(selfAssessmentHandler.GetAllSelfAssessmentsAdmin),
			),
		),
	)
	mux.Handle("DELETE /api/v1/admin/self-assessments/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("admin")(
				http.HandlerFunc(selfAssessmentHandler.DeleteSelfAssessment),
			),
		),
	)

	// Reviewer routes for self-assessments
	mux.Handle("GET /api/v1/review/open-assessments",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(selfAssessmentHandler.GetOpenAssessmentsForReview),
			),
		),
	)

	// Reviewer response routes (only reviewers, NOT admins - strict role separation)
	mux.Handle("GET /api/v1/review/assessment/{id}/responses",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(reviewerHandler.GetResponses),
			),
		),
	)
	mux.Handle("POST /api/v1/review/assessment/{id}/responses",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(reviewerHandler.CreateOrUpdateResponse),
			),
		),
	)
	mux.Handle("DELETE /api/v1/review/assessment/{id}/responses/{categoryId}",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(reviewerHandler.DeleteResponse),
			),
		),
	)
	mux.Handle("POST /api/v1/review/assessment/{id}/complete",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(reviewerHandler.CompleteReview),
			),
		),
	)
	mux.Handle("GET /api/v1/review/assessment/{id}/completion-status",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(reviewerHandler.GetCompletionStatus),
			),
		),
	)

	// Consolidation routes (reviewer/admin only)
	mux.Handle("GET /api/v1/review/consolidation/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.GetConsolidationData),
			),
		),
	)
	mux.Handle("POST /api/v1/review/consolidation/{id}/override",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.CreateOrUpdateOverride),
			),
		),
	)
	mux.Handle("POST /api/v1/review/consolidation/{id}/override/{categoryId}/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.ApproveOverride),
			),
		),
	)
	mux.Handle("DELETE /api/v1/review/consolidation/{id}/override/{categoryId}/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.RevokeOverrideApproval),
			),
		),
	)
	mux.Handle("DELETE /api/v1/review/consolidation/{id}/override/{categoryId}",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.DeleteOverride),
			),
		),
	)
	mux.Handle("POST /api/v1/review/consolidation/{id}/averaged/{categoryId}/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.ApproveAveragedResponse),
			),
		),
	)
	mux.Handle("DELETE /api/v1/review/consolidation/{id}/averaged/{categoryId}/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.RevokeAveragedApproval),
			),
		),
	)
	mux.Handle("POST /api/v1/review/consolidation/{id}/final",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.SaveFinalConsolidation),
			),
		),
	)
	mux.Handle("POST /api/v1/review/consolidation/{id}/final/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.ApproveFinalConsolidation),
			),
		),
	)
	mux.Handle("DELETE /api/v1/review/consolidation/{id}/final/approve",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(consolidationHandler.RevokeFinalApproval),
			),
		),
	)

	// Discussion endpoints
	mux.Handle("GET /api/v1/discussion/{id}",
		authMw.Authenticate(
			rbacMw.RequireRole("reviewer")(
				http.HandlerFunc(discussionHandler.GetDiscussionResult),
			),
		),
	)

	mux.Handle("PUT /api/v1/discussion/{id}/note",
		authMw.Authenticate(
			rbacMw.RequireRole("user")(
				http.HandlerFunc(discussionHandler.UpdateDiscussionNote),
			),
		),
	)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.HealthCheck(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, err := w.Write([]byte(`{"status":"unhealthy","database":"error"}`))
			if err != nil {
				slog.Error("Failed to write health check response", "error", err)
				return
			}
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"healthy","version":"` + cfg.App.Version + `"}`))
		if err != nil {
			slog.Error("Failed to write health check response", "error", err)
			return
		}
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
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
