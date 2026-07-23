// pkg/api/middleware_test.g
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestLoggingMiddleware tests request logging.
func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := LoggingMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

// TestCORSMiddleware tests CORS header handling.
func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware(handler)

	tests := []struct {
		method string
		desc   string
	}{
		{http.MethodGet, "GET request"},
		{http.MethodPost, "POST request"},
		{http.MethodOptions, "OPTIONS request"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/api/test", nil)
		w := httptest.NewRecorder()

		wrapped.ServeHTTP(w, req)

		// Check CORS headers are set
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("%s: Missing CORS origin header", tt.desc)
		}

		if tt.method == http.MethodOptions {
			if w.Code != http.StatusOK {
				t.Errorf("%s: Expected 200 for OPTIONS, got %d", tt.desc, w.Code)
			}
		}
	}
}

// TestCORSHeadersAllowMethods tests that allowed methods are set.
func TestCORSHeadersAllowMethods(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware(handler)

	req := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	for _, method := range expectedMethods {
		if !strings.Contains(methods, method) {
			t.Errorf("Missing method %s in Allow-Methods header", method)
		}
	}
}

// TestRecoveryMiddleware tests panic recovery.
func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	// Should not panic, should return 500
	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Internal server error") {
		t.Errorf("Expected error message in response")
	}
}

// TestRecoveryMiddlewareNoPanic tests normal request when no panic.
func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := RecoveryMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %s", w.Body.String())
	}
}

// TestTimingMiddleware tests request timing.
func TestTimingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := TimingMiddleware(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestMiddlewareChaining tests that middleware are applied in correct order.
func TestMiddlewareChaining(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain multiple middleware
	wrapped := Chain(handler, LoggingMiddleware, CORSMiddleware, TimingMiddleware)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Check CORS headers are still present when chained
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers missing in chained middleware")
	}
}

// TestResponseWriterStatusCode tests that status code is captured.
func TestResponseWriterStatusCode(t *testing.T) {
	w := &responseWriter{
		ResponseWriter: httptest.NewRecorder(),
		statusCode:     http.StatusOK,
	}

	w.WriteHeader(http.StatusCreated)

	if w.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.statusCode)
	}
}

// TestResponseWriterWrite tests that write is delegated.
func TestResponseWriterWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	data := []byte("test data")
	n, err := w.Write(data)

	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if rec.Body.String() != "test data" {
		t.Errorf("Expected body 'test data', got %s", rec.Body.String())
	}
}

// TestResponseWriterFlush tests flusher support.
func TestResponseWriterFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Should not panic even if flusher not supported
	w.Flush()
}

// TestChainEmpty tests chaining with no middleware.
func TestChainEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Chain with no middleware
	wrapped := Chain(handler)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestChainOrder tests that middleware execute in expected order.
func TestChainOrder(t *testing.T) {
	var order []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Chain(handler, m1, m2)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Chain applies right-to-left: m2 wraps handler first, then m1 wraps that
	// So execution order: m1-before, m2-before, handler, m2-after, m1-after
	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}

	if len(order) != len(expected) {
		t.Errorf("Expected %d steps, got %d", len(expected), len(order))
	}

	for i, step := range expected {
		if i >= len(order) || order[i] != step {
			t.Errorf("Step %d: expected %s, got %s", i, step, order[i])
		}
	}
}
