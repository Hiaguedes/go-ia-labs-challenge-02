package webserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWebServer(t *testing.T) {
	s := NewWebServer(":9000")

	if s.WebServerPort != ":9000" {
		t.Errorf("WebServerPort = %q, want %q", s.WebServerPort, ":9000")
	}
	if s.Router == nil {
		t.Error("Router is nil")
	}
	if s.Handlers == nil || len(s.Handlers) != 0 {
		t.Errorf("Handlers = %v, want empty non-nil map", s.Handlers)
	}
}

func TestAddHandler(t *testing.T) {
	s := NewWebServer(":0")
	s.AddHandler("/ping", func(w http.ResponseWriter, r *http.Request) {})

	if _, ok := s.Handlers["/ping"]; !ok {
		t.Error("handler was not registered under /ping")
	}
}

func TestHandler_RoutesRequestsToRegisteredHandlers(t *testing.T) {
	s := NewWebServer(":0")
	s.AddHandler("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, "pong")
	})

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if rec.Body.String() != "pong" {
		t.Errorf("body = %q, want %q", rec.Body.String(), "pong")
	}
}

func TestHandler_UnknownRouteReturns404(t *testing.T) {
	s := NewWebServer(":0")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestStart_ReturnsErrorForInvalidAddress(t *testing.T) {
	// An address without a port cannot be listened on, so Start fails immediately
	// instead of blocking.
	s := NewWebServer("invalid-address")

	if err := s.Start(); err == nil {
		t.Fatal("Start() expected an error for an invalid address, got nil")
	}
}
