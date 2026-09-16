package main

import (
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAddressHonoursPort(t *testing.T) {
	t.Setenv("PORT", "")
	if got := address(); got != ":8080" {
		t.Fatalf("address() = %q, want the default", got)
	}
	t.Setenv("PORT", "9123")
	if got := address(); got != ":9123" {
		t.Fatalf("address() = %q", got)
	}
}

func TestAllowedOriginsFromEnvironment(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", " http://a.example , ,http://b.example ")
	origins := allowedOrigins()
	if len(origins) != 2 || origins[0] != "http://a.example" || origins[1] != "http://b.example" {
		t.Fatalf("allowedOrigins() = %v", origins)
	}
	t.Setenv("ALLOWED_ORIGINS", "")
	if len(allowedOrigins()) == 0 {
		t.Fatal("the defaults should never be empty")
	}
}

func TestServerTimeoutsAreSet(t *testing.T) {
	server := newHTTPServer(":0", http.NewServeMux())
	if server.ReadHeaderTimeout == 0 || server.ReadTimeout == 0 || server.WriteTimeout == 0 || server.IdleTimeout == 0 {
		t.Fatalf("every timeout must be set: %+v", server)
	}
	// An AI call may legitimately take up to the client's 60s ceiling, so the
	// write timeout has to leave room for that and the response.
	if server.WriteTimeout <= 60*time.Second {
		t.Fatalf("WriteTimeout = %v, which is too short for an AI call", server.WriteTimeout)
	}
}

// TestDefaultsAllowThePublishedFrontendPort is the guard for a real bug: the
// frontend was moved to another host port while the backend's CORS allowlist kept
// the old one, so the browser's same-origin requests were refused with 403 and the
// game never loaded. The allowlist is now checked against docker-compose.yml.
func TestDefaultsAllowThePublishedFrontendPort(t *testing.T) {
	compose, err := os.ReadFile("../docker-compose.yml")
	if err != nil {
		t.Skipf("docker-compose.yml is not readable from here: %v", err)
	}
	frontendPort := publishedFrontendPort(t, string(compose))
	if frontendPort == "" {
		t.Fatal("no published frontend port found in docker-compose.yml")
	}
	origins := map[string]bool{}
	for _, origin := range allowedOrigins() {
		origins[origin] = true
	}
	for _, host := range []string{"localhost", "127.0.0.1"} {
		want := "http://" + host + ":" + frontendPort
		if !origins[want] {
			t.Fatalf("the frontend is published on %s but %s is not in the allowed origins: %v", frontendPort, want, allowedOrigins())
		}
	}
}

// publishedFrontendPort finds the frontend service's published host port. A
// service starts with a two-space indented "name:" line; everything indented
// deeper belongs to it.
func publishedFrontendPort(t *testing.T, compose string) string {
	t.Helper()
	serviceStart := regexp.MustCompile(`^ {2}([A-Za-z0-9_-]+):\s*$`)
	published := regexp.MustCompile(`127\.0\.0\.1:(\d+):\d+`)

	lines := strings.Split(compose, "\n")
	inFrontend := false
	for _, line := range lines {
		if matches := serviceStart.FindStringSubmatch(line); matches != nil {
			inFrontend = matches[1] == "frontend"
			continue
		}
		if !inFrontend {
			continue
		}
		if matches := published.FindStringSubmatch(line); matches != nil {
			if _, err := strconv.Atoi(matches[1]); err == nil {
				return matches[1]
			}
		}
	}
	return ""
}
