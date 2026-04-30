//go:build !windows

package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
)

// HTTPServer implements Server over HTTP for non-Windows platforms.
type HTTPServer struct {
	port int
}

// NewHTTPServer creates an HTTPServer listening on the given port.
func NewHTTPServer(port int) *HTTPServer {
	return &HTTPServer{port: port}
}

// Serve starts the HTTP server and blocks until ctx is cancelled.
func (s *HTTPServer) Serve(ctx context.Context, handler HandlerFunc) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp := Response{OK: false, Error: fmt.Sprintf("bad request: %v", err)}
			writeJSON(w, resp)
			return
		}
		resp := handler(req)
		writeJSON(w, resp)
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		resp := handler(Request{Command: CmdStatus})
		writeJSON(w, resp)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("http server: listen %s: %w", addr, err)
	}

	srv := &http.Server{Handler: mux}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	log.Printf("http server: listening on %s", addr)
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		select {
		case <-ctx.Done():
			return nil
		default:
			return fmt.Errorf("http server: serve: %w", err)
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("http server: encode response: %v", err)
	}
}
