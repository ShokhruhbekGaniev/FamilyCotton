package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/familycotton/api/internal/handler"
	"github.com/familycotton/api/internal/middleware"
	"github.com/familycotton/api/internal/service"
)

func New(
	authService *service.AuthService,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.CORS)
	r.Use(middleware.Logging)

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required).
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/auth/logout", authHandler.Logout)

		// Protected routes.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authService))

			r.Get("/auth/me", authHandler.Me)

			// Users (owner only).
			r.Route("/users", func(r chi.Router) {
				r.Use(middleware.RequireRole("owner"))
				r.Get("/", userHandler.List)
				r.Post("/", userHandler.Create)
				r.Put("/{id}", userHandler.Update)
				r.Delete("/{id}", userHandler.Delete)
			})
		})
	})

	return r
}
