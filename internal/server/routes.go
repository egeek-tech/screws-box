package server

import (
	"io/fs"
	"net/http"
	"screws-box/internal/session"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds application dependencies for HTTP handlers.
type Server struct {
	store    StoreService
	sessions *session.Manager
	version  string
	// trustedProxyCIDRs lists the reverse-proxy IP ranges in front of the
	// server. When non-empty the client IP is read from X-Forwarded-For
	// (skipping these hops); when empty it is the direct connection address.
	trustedProxyCIDRs []string
}

// Option configures a Server.
type Option func(*Server)

// WithTrustedProxyCIDRs declares the reverse-proxy IP ranges (CIDRs) sitting
// in front of the server. When set, the client IP used for rate limiting and
// logging is extracted from X-Forwarded-For, walking right-to-left and skipping
// these trusted hops so clients cannot spoof their address. When left empty the
// client IP is the direct TCP connection address (safe when no proxy is in
// front). Each CIDR must already be validated by the caller.
func WithTrustedProxyCIDRs(cidrs []string) Option {
	return func(s *Server) { s.trustedProxyCIDRs = cidrs }
}

// NewServer creates a Server with the given dependencies and options.
func NewServer(store StoreService, sessions *session.Manager, version string, opts ...Option) *Server {
	srv := &Server{store: store, sessions: sessions, version: version}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

// Router creates the chi router with all routes.
func (srv *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Healthcheck — outside logging middleware so K8s probes don't flood logs.
	r.Get("/healthz", srv.handleHealthz())

	// All application routes with full middleware stack.
	r.Group(func(r chi.Router) {
		r.Use(srv.clientIPMiddleware())
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(middleware.RequestID)
		r.Use(newRateLimitAPI())

		// Public routes (no auth required)
		r.Get("/login", srv.handleLoginPage())
		r.With(newRateLimitLogin()).Post("/login", srv.handleLoginPost())
		r.Get("/logout", srv.handleLogout())

		// OIDC routes (public -- callback must not be behind authMiddleware)
		r.Get("/auth/oidc", srv.handleOIDCStart())
		r.With(newRateLimitLogin()).Get("/auth/callback", srv.handleOIDCCallback())

		r.Handle("/static/*", http.StripPrefix("/static/",
			http.FileServerFS(mustSubFS(ContentFS, "static"))))

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(srv.authMiddleware())
			r.Use(srv.csrfProtect())

			r.Get("/", srv.handleGrid())
			r.Get("/settings", srv.handleSettings())

			r.Route("/api", func(r chi.Router) {
				r.Route("/items", func(r chi.Router) {
					r.Get("/", srv.handleListItems())
					r.Post("/", srv.handleCreateItem())
					r.Route("/{itemID}", func(r chi.Router) {
						r.Get("/", srv.handleGetItem())
						r.Put("/", srv.handleUpdateItem())
						r.Delete("/", srv.handleDeleteItem())
						r.Post("/tags", srv.handleAddTag())
						r.Delete("/tags/{tagName}", srv.handleRemoveTag())
					})
				})
				r.Get("/tags", srv.handleListTags())
				r.Route("/tags/{tagID}", func(r chi.Router) {
					r.Put("/", srv.handleRenameTag())
					r.Delete("/", srv.handleDeleteTag())
				})
				r.Get("/search", srv.handleSearch())
				r.Get("/containers/{containerID}/items", srv.handleListContainerItems())
				r.Put("/shelf/resize", srv.handleResizeShelf())
				r.Get("/shelf/auth", srv.handleGetAuthSettings())
				r.Put("/shelf/auth", srv.handleUpdateAuthSettings())
				r.Get("/oidc/config", srv.handleGetOIDCConfig())
				r.Put("/oidc/config", srv.handleUpdateOIDCConfig())
				r.Get("/export", srv.handleExport())
				r.Post("/import/validate", srv.handleImportValidate())
				r.Post("/import/confirm", srv.handleImportConfirm())
				r.Get("/duplicates", srv.handleDuplicates())
				r.Get("/sessions", srv.handleListSessions())
				r.Delete("/sessions", srv.handleRevokeAllOthers())
				r.Delete("/sessions/{sessionID}", srv.handleRevokeSession())
			})
		})
	})

	return r
}

func mustSubFS(parent fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(parent, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
