// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"kpp.dev/kpfc/internal/config"
	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/middleware"
	"kpp.dev/kpfc/internal/repository/dynamo"
	"kpp.dev/kpfc/internal/repository/s3"
	"kpp.dev/kpfc/internal/router"
	cardUseCase "kpp.dev/kpfc/internal/usecase/card"
	deckUseCase "kpp.dev/kpfc/internal/usecase/deck"
	userUseCase "kpp.dev/kpfc/internal/usecase/user"
)

func main() {
	log.Println("kpfc - Flashcards API")
	log.Println("Copyright 2026 kpp.dev")
	log.Println("Licensed under GNU Affero General Public License v3.0")
	log.Println("Source code: https://github.com/pedrokpp/kpfc")
	log.Println()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	dynamoClient, err := dynamo.NewClient(
		ctx,
		cfg.AWS.Region,
		cfg.AWS.Endpoint,
		cfg.AWS.AccessKeyID,
		cfg.AWS.SecretAccessKey,
	)
	if err != nil {
		log.Fatalf("Failed to create DynamoDB client: %v", err)
	}

	s3Storage, err := s3.NewStorageRepository(
		ctx,
		cfg.S3.BucketAvatars,
		cfg.AWS.Region,
		cfg.AWS.Endpoint,
		cfg.AWS.AccessKeyID,
		cfg.AWS.SecretAccessKey,
	)
	if err != nil {
		log.Fatalf("Failed to create S3 storage: %v", err)
	}

	userRepo := dynamo.NewUserRepository(dynamoClient, cfg.DynamoDB.TableUsers)
	cardRepo := dynamo.NewCardRepository(dynamoClient, cfg.DynamoDB.TableCards)
	deckRepo := dynamo.NewDeckRepository(dynamoClient, cfg.DynamoDB.TableDecks, cardRepo)

	userUC := userUseCase.NewUseCase(userRepo, s3Storage)
	deckUC := deckUseCase.NewUseCase(deckRepo)
	cardUC := cardUseCase.NewUseCase(cardRepo, deckRepo)

	jwtExpiration, err := time.ParseDuration(cfg.JWT.Expiration)
	if err != nil {
		log.Fatalf("Invalid JWT expiration: %v", err)
	}
	jwtMid := middleware.NewJWTMiddleware(cfg.JWT.Secret, jwtExpiration)
	corsMid := middleware.NewCORSMiddleware(cfg.CORS.AllowedOrigins)

	authHandler := handler.NewAuthHandler(userUC, jwtMid)
	userHandler := handler.NewUserHandler(userUC)
	deckHandler := handler.NewDeckHandler(deckUC)
	cardHandler := handler.NewCardHandler(cardUC)

	r := router.New(router.Config{
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		DeckHandler:    deckHandler,
		CardHandler:    cardHandler,
		JWTMiddleware:  jwtMid,
		CORSMiddleware: corsMid,
		AGPLSourceURL:  "https://github.com/pedrokpp/kpfc",
	})

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
