package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bimanyaya/api/internal/analysis"
	"bimanyaya/api/internal/audit"
	"bimanyaya/api/internal/auth"
	"bimanyaya/api/internal/cases"
	"bimanyaya/api/internal/clarifications"
	"bimanyaya/api/internal/config"
	"bimanyaya/api/internal/consent"
	"bimanyaya/api/internal/db"
	"bimanyaya/api/internal/documents"
	"bimanyaya/api/internal/drafts"
	"bimanyaya/api/internal/eligibility"
	"bimanyaya/api/internal/review"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize structured logger (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting BimaNyaya Core API Backend...")

	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize Database connection (Convex client)
	database, err := db.Connect(cfg)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}

	// 3. Initialize Services
	authSvc := auth.NewAuthService(database, cfg)
	eligibilitySvc := eligibility.NewService(database)
	caseSvc := cases.NewService(database)
	consentSvc := consent.NewService(database)
	docSvc := documents.NewService(database)
	analysisSvc := analysis.NewService(database, cfg.AIWorkerURL)
	clarificationSvc := clarifications.NewService(database)
	draftSvc := drafts.NewService(database, cfg.AIWorkerURL)
	reviewSvc := review.NewService(database)
	auditSvc := audit.NewService(database)

	// 4. Setup Router
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware(cfg.CORSAllowedOrigins))
	r.Use(maxBodySizeMiddleware)

	// Base API endpoint
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"online","service":"BimaNyaya API Backend","version":"1.0.0"}`))
	})

	// API Routes Group
	r.Route("/api/v1", func(r chi.Router) {
		// Document Upload simulation endpoint (Public mock for multipart S3 upload emulation)
		r.Post("/documents/upload-endpoint/{documentId}", docSvc.UploadEndpoint)

		// Public Auth Routes (for local E2E simulation, only registered in development)
		if cfg.Environment == "development" {
			r.Post("/auth/request-otp", authSvc.RequestOTP)
			r.Post("/auth/verify-otp", authSvc.VerifyOTP)
		}

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(authSvc.AuthMiddleware)

			r.Get("/auth/me", authSvc.Me)

			// Eligibility
			r.Post("/eligibility/check", eligibilitySvc.CheckEligibility)

			// Cases
			r.Post("/cases", caseSvc.CreateCase)
			r.Get("/cases", caseSvc.GetCases)
			r.Get("/cases/{caseId}", caseSvc.GetCase)
			r.Patch("/cases/{caseId}", caseSvc.PatchCase)
			r.Delete("/cases/{caseId}", caseSvc.DeleteCase)
			r.Get("/cases/{caseId}/timeline", caseSvc.GetTimeline)

			// Consents
			r.Post("/cases/{caseId}/consents", consentSvc.RecordConsent)
			r.Get("/cases/{caseId}/consents", consentSvc.GetConsents)
			r.Post("/cases/{caseId}/consents/withdraw", consentSvc.WithdrawConsent)

			// Documents
			r.Post("/cases/{caseId}/documents/upload-url", docSvc.GetUploadURL)
			r.Post("/cases/{caseId}/documents/complete", docSvc.CompleteUpload)
			r.Get("/cases/{caseId}/documents", docSvc.GetCaseDocuments)
			r.Get("/documents/{documentId}", docSvc.GetDocument)
			r.Delete("/documents/{documentId}", docSvc.DeleteDocument)

			// Analysis and RAG Pipeline
			r.Post("/cases/{caseId}/process", analysisSvc.TriggerCaseProcessing)
			r.Get("/cases/{caseId}/processing-status", analysisSvc.GetProcessingStatus)
			r.Get("/cases/{caseId}/analysis", analysisSvc.GetAnalysis)
			r.Get("/cases/{caseId}/citations", analysisSvc.GetCitations)
			r.Get("/cases/{caseId}/evidence", analysisSvc.GetEvidence)

			// Clarifications
			r.Get("/cases/{caseId}/clarifications", clarificationSvc.GetQuestions)
			r.Post("/cases/{caseId}/clarifications/{questionId}/answer", clarificationSvc.SubmitAnswer)

			// Drafts
			r.Get("/cases/{caseId}/drafts", draftSvc.GetCaseDrafts)
			r.Post("/cases/{caseId}/drafts", draftSvc.CreateDraft)
			r.Patch("/drafts/{draftId}", draftSvc.PatchDraft)
			r.Post("/drafts/{draftId}/translate", draftSvc.TranslateDraft)
			r.Get("/drafts/{draftId}/pdf", draftSvc.ExportPDF)

			// Reviewer Console
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("REVIEWER", "SENIOR_REVIEWER", "ADMIN"))
				r.Get("/reviewer/cases", reviewSvc.GetReviewerCases)
				r.Post("/reviewer/cases/{caseId}/claim", reviewSvc.ClaimCase)
				r.Post("/reviewer/cases/{caseId}/request-information", reviewSvc.RequestInformation)
				r.Post("/reviewer/cases/{caseId}/approve", reviewSvc.ApproveCase)
				r.Post("/reviewer/cases/{caseId}/escalate", reviewSvc.EscalateCase)
				r.Post("/reviewer/cases/{caseId}/reject", reviewSvc.RejectCase)
				r.Post("/reviewer/cases/{caseId}/comments", reviewSvc.AddReviewComment)
				r.Get("/reviewer/cases/{caseId}/comments", reviewSvc.GetReviewComments)
			})

			// Admin Controls
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("ADMIN"))
				r.Get("/admin/audit-logs", auditSvc.GetAuditLogs)
				r.Post("/admin/reviews/sla-checks", reviewSvc.CheckReviewSLAs)
			})
		})
	})

	// 5. Start Server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info(fmt.Sprintf("HTTP Server listening on port %s", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to start HTTP server", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down HTTP server gracefully...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited.")
}

func corsMiddleware(allowedOrigins string) func(http.Handler) http.Handler {
	origins := strings.Split(allowedOrigins, ",")
	allowedMap := make(map[string]bool)
	for _, o := range origins {
		allowedMap[strings.TrimSpace(o)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedMap[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func maxBodySizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // Limit requests to 10MB
		next.ServeHTTP(w, r)
	})
}
