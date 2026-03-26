package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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
)

// version is injected at build time via:
//
//	go build -ldflags "-X main.version=$(cat VERSION)" -o kpfc .
var version string

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
		log.Warn("JWT_SECRET is not set — authentication will not work correctly")
	}

	log.Debug("config loaded", "port", cfg.Port, "db_path", cfg.DBPath)

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	log.Info("database connected", "path", cfg.DBPath)

	if err := db.AutoMigrate(
		&model.User{},
		&model.Deck{},
		&model.Card{},
		&model.UserDeckUpvote{},
	); err != nil {
		log.Error("failed to run migrations", "err", err)
		os.Exit(1)
	}
	log.Debug("migrations applied")

	userRepo := repository.NewGORMUserRepository(db)
	deckRepo := repository.NewGORMDeckRepository(db)
	cardRepo := repository.NewGORMCardRepository(db)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	userSvc := service.NewUserService(userRepo)
	deckSvc := service.NewDeckService(deckRepo)
	cardSvc := service.NewCardService(cardRepo, deckRepo)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(userSvc)
	deckHandler := handler.NewDeckHandler(deckSvc)
	cardHandler := handler.NewCardHandler(cardSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	if *verbose {
		r.Use(chimiddleware.Logger)
	}

	r.Get("/api/v1/health", healthHandler(version))
	r.Get("/api/v1/version", versionHandler(version))

	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)

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
	})

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
