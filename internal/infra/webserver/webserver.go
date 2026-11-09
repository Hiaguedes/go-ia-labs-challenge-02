package webserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type WebServer struct {
	Router        chi.Router
	Handlers      map[string]http.HandlerFunc
	WebServerPort string
}

func NewWebServer(serverPort string) *WebServer {
	return &WebServer{
		Router:        chi.NewRouter(),
		Handlers:      make(map[string]http.HandlerFunc),
		WebServerPort: serverPort,
	}
}

func (s *WebServer) AddHandler(path string, handler http.HandlerFunc) {
	s.Handlers[path] = handler
}

// Handler registers the logger and telemetry middlewares and all added handlers
// on the router and returns it, ready to be served.
func (s *WebServer) Handler() http.Handler {
	s.Router.Use(middleware.Logger)
	s.Router.Use(telemetryMiddleware)
	for path, handler := range s.Handlers {
		s.Router.Handle(path, handler)
	}
	return s.Router
}

// Start serves the router on WebServerPort. It blocks until the server fails
// and always returns a non-nil error.
func (s *WebServer) Start() error {
	return http.ListenAndServe(s.WebServerPort, s.Handler())
}
