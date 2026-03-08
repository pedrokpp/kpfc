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

package router

import (
	"github.com/go-chi/chi/v5"

	"kpp.dev/kpfc/internal/handler"
	"kpp.dev/kpfc/internal/middleware"
)

// Config holds all dependencies needed to configure the router
type Config struct {
	AuthHandler    *handler.AuthHandler
	UserHandler    *handler.UserHandler
	DeckHandler    *handler.DeckHandler
	CardHandler    *handler.CardHandler
	JWTMiddleware  *middleware.JWTMiddleware
	CORSMiddleware *middleware.CORSMiddleware
	AGPLSourceURL  string
}

// New creates and configures a new Chi router with all routes and middlewares
func New(cfg Config) chi.Router {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recovery)
	r.Use(cfg.CORSMiddleware.Handler)
	r.Use(middleware.AGPLSourceCode(cfg.AGPLSourceURL))

	// Setup all routes
	setupRoutes(r, cfg)

	return r
}
