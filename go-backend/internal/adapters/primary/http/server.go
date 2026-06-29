package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/omniroute/go-backend/internal/adapters/primary/http/handlers"
	"github.com/omniroute/go-backend/internal/adapters/primary/http/middleware"
	"github.com/omniroute/go-backend/internal/core/ports/primary"
	"go.uber.org/zap"
)

type Server struct {
	httpServer *http.Server
	log        *zap.Logger
}

type ServerDeps struct {
	ChatUseCase     primary.ChatUseCase
	ProviderUseCase primary.ProviderUseCase
	ModelUseCase    primary.ModelUseCase
	ComboUseCase    primary.ComboUseCase
	AuthUseCase     primary.AuthUseCase
	UsageUseCase    primary.UsageUseCase
	Log             *zap.Logger
	Port            string
	RequireAuth     bool
}

func NewServer(deps *ServerDeps) *Server {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestLogger(deps.Log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Handlers.
	chatH := handlers.NewChatHandler(deps.ChatUseCase, deps.Log)
	embH := handlers.NewEmbeddingsHandler(deps.ChatUseCase, deps.Log)
	modH := handlers.NewModelsHandler(deps.ModelUseCase, deps.Log)
	provH := handlers.NewProvidersHandler(deps.ProviderUseCase, deps.Log)
	comboH := handlers.NewCombosHandler(deps.ComboUseCase, deps.Log)
	keyH := handlers.NewAPIKeysHandler(deps.AuthUseCase, deps.Log)

	// Auth middleware.
	authMiddleware := middleware.OptionalAuth(deps.AuthUseCase.(middleware.APIKeyValidator))

	// Health (no auth).
	r.Get("/health", handlers.Health)
	r.Get("/", handlers.Health)

	// OpenAI-compatible v1 API.
	r.Route("/v1", func(r chi.Router) {
		r.Use(authMiddleware)

		// Chat completions.
		r.Post("/chat/completions", chatH.ChatCompletions)

		// Embeddings.
		r.Post("/embeddings", embH.Embeddings)

		// Models.
		r.Get("/models", modH.ListModels)
		r.Get("/models/{model}", modH.GetModel)
	})

	// Management API.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)

		// Providers.
		r.Route("/providers", func(r chi.Router) {
			r.Get("/", provH.List)
			r.Post("/", provH.Create)
			r.Get("/health", provH.Health)
			r.Get("/{id}", provH.Get)
			r.Put("/{id}", provH.Update)
			r.Delete("/{id}", provH.Delete)
		})

		// Combos.
		r.Route("/combos", func(r chi.Router) {
			r.Get("/", comboH.List)
			r.Post("/", comboH.Create)
			r.Get("/{id}", comboH.Get)
			r.Put("/{id}", comboH.Update)
			r.Delete("/{id}", comboH.Delete)
		})

		// API keys.
		r.Route("/api-keys", func(r chi.Router) {
			r.Get("/", keyH.List)
			r.Post("/", keyH.Create)
			r.Delete("/{id}", keyH.Revoke)
		})
	})

	// Auth endpoints (no auth required).
	r.Post("/api/v1/auth/login", keyH.Login)

	srv := &http.Server{
		Addr:         ":" + deps.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second, // Long for streaming.
		IdleTimeout:  120 * time.Second,
	}

	return &Server{httpServer: srv, log: deps.Log}
}

func (s *Server) Start() error {
	s.log.Info("starting HTTP server", zap.String("addr", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
