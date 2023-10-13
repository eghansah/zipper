package main

import (
	"fmt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/handlers"
	"github.com/spf13/viper"
)

func (s *service) InitRoutes() {

	r := chi.NewRouter()

	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(120 * time.Second))

	r.Route(fmt.Sprintf("%s/oneoffs", s.config.URLPrefix), func(r chi.Router) {
		//TODO: provide api authentication
		// r.Use(s.authGateway.Middlewares.LoginRequired)

		r.Get("/info", s.Info())
		// r.Get("/zip", s.zip())
		// r.Post("/zip", s.zip())

		r.Get("/zip2", s.zipAlt())
		r.Post("/zip2", s.zipAlt())

	})

	s.router = r

	credentials := handlers.AllowCredentials()
	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "x-csrf-token"})
	methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})
	origins := handlers.AllowedOrigins(viper.GetStringSlice("CORS_ORIGIN_WHITELIST"))

	// s.svr.Handler = &r
	s.svr.Handler = handlers.CORS(credentials, headers, methods, origins)(r)
}
