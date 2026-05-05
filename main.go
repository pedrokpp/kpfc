package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kpp.dev/kpfc/internal/config"
	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/logger"
	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/model"
	"kpp.dev/kpfc/internal/repository"
	"kpp.dev/kpfc/internal/service"
	"kpp.dev/kpfc/internal/storage"
)

// version is injected at build time via:
//
//	go build -ldflags "-X main.version=$(cat VERSION)" -o kpfc .
var version string = "local"

func main() {
	debug := flag.Bool("debug", false, "enable debug-level log output")
	verbose := flag.Bool("verbose", false, "enable verbose output")
	logFormat := flag.String("log-format", "human", `log output format: "human" or "json"`)
	flag.Parse()

	log := logger.New(*logFormat, *debug)

	if *verbose {
		log.Info("verbose output enabled")
	}

	if version == "" {
		version = "dev"
	}
	log.Info("starting kpfc", "version", version)

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	if cfg.JWTSecret == "" {
		log.Error("JWT_SECRET is not set — authentication will not work correctly")
		os.Exit(-1)
	}

	log.Debug("config loaded", "port", cfg.Port, "db_path", cfg.DBPath)

	if cfg.DBPath == "" {
		log.Error("DB_PATH is not set")
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Error("failed to create database directory", "path", filepath.Dir(cfg.DBPath), "err", err)
		os.Exit(1)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	log.Info("database connected", "driver", "sqlite", "path", cfg.DBPath)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Deck{},
		&model.Card{},
		&model.UserDeckUpvote{},
		&model.Media{},
	); err != nil {
		log.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}
	log.Debug("migrations applied")

	// Na prática:
	// - dono: `7` = leitura + escrita + execução
	// - grupo: `5` = leitura + execução
	// - outros: `5` = leitura + execução
	// 0o755 = rwxr-xr-x
	if err := os.MkdirAll(cfg.MediaRoot, 0o755); err != nil {
		log.Error("failed to create media root directory", "err", err)
		os.Exit(1)
	}

	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)
	mediaRepo := repository.NewGORMMediaRepository(db)
	localStore := storage.NewLocalStorage(cfg.MediaRoot)
	accessSvc := service.NewAccessService(deckRepo, cardRepo, mediaRepo)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	userSvc := service.NewUserService(userRepo)
	deckSvc := service.NewDeckService(deckRepo, accessSvc)
	cardSvc := service.NewCardService(cardRepo, accessSvc)
	studySvc := service.NewStudyService(cardRepo, userRepo, accessSvc)
	mediaSvc := service.NewMediaService(mediaRepo, localStore, newUUID, accessSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)
	cardHandler := handler.NewCardHandler(cardSvc)
	studyHandler := handler.NewStudyHandler(studySvc)
	publicHandler := handler.NewPublicHandler(deckSvc)
	mediaHandler := handler.NewMediaHandler(mediaSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.CORS(cfg.CORSAllowOrigins))
	if *verbose {
		r.Use(chimiddleware.Logger)
	}

	r.Get("/api/v1/health", healthHandler(version))
	r.Get("/api/v1/version", versionHandler(version))

	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)

	r.Get("/api/v1/public/decks", publicHandler.ListDecks)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(authSvc))
		r.Get("/api/v1/users/me", userHandler.GetMe)
		r.Put("/api/v1/users/me", userHandler.UpdateMe)

		r.Get("/api/v1/decks", deckHandler.List)
		r.Post("/api/v1/decks", deckHandler.Create)
		r.Get("/api/v1/decks/{id}", deckHandler.Get)
		r.Put("/api/v1/decks/{id}", deckHandler.Update)
		r.Delete("/api/v1/decks/{id}", deckHandler.Delete)

		r.Get("/api/v1/decks/{id}/cards", cardHandler.ListByDeck)
		r.Post("/api/v1/decks/{id}/cards", cardHandler.Create)
		r.Get("/api/v1/cards/{id}", cardHandler.Get)
		r.Put("/api/v1/cards/{id}", cardHandler.Update)
		r.Delete("/api/v1/cards/{id}", cardHandler.Delete)

		r.Post("/api/v1/decks/{id}/study", studyHandler.StartSession)
		r.Post("/api/v1/cards/{id}/review", studyHandler.SubmitReview)

		r.Post("/api/v1/decks/{id}/upvote", deckHandler.Upvote)

		r.Post("/api/v1/media", mediaHandler.Upload)
		r.Delete("/api/v1/media/{id}", mediaHandler.Delete)
	})

	r.Get("/api/v1/media/{id}", mediaHandler.Serve)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "err", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}

// newUUID generates a random 16-byte hex string suitable for storage paths.
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func healthHandler(v string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": v,
		})
	}
}

func versionHandler(v string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"version": v,
		})
	}
}
