// Command backend serves the Scribblebox game: a small Scribblenauts-style world
// where you type an object's name and it appears, then use it to solve puzzles.
// Every rule is deterministic and local — no model, no keys, no network calls.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"backend/api"
)

const defaultPort = "8080"

// defaultAllowedOrigins must cover the port the frontend is published on, or the
// browser's own same-origin requests are refused. main_test.go checks this against
// docker-compose.yml so moving a port cannot break the game silently.
const defaultAllowedOrigins = "http://localhost:3000,http://127.0.0.1:3000," +
	"http://localhost:3001,http://127.0.0.1:3001," +
	"http://localhost:3003,http://127.0.0.1:3003"

// newHTTPServer keeps timeouts explicit so a slow or half-open client cannot tie
// up the process.
func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      75 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func address() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}
	return ":" + port
}

func allowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		raw = defaultAllowedOrigins
	}
	origins := []string{}
	for _, origin := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func main() {
	server := newHTTPServer(address(), api.New(api.NewWorldStore(500), allowedOrigins()))

	go func() {
		log.Printf("scribblebox listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server stopped: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Print("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
