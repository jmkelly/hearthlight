package discovery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestHTTPTitleReadsDirectTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html><head><title>Grafana</title></head></html>")
	}))
	defer server.Close()

	ip, port := serverAddr(t, server.URL)
	s, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.httpTitle(context.Background(), ip, port); got != "Grafana" {
		t.Fatalf("httpTitle = %q, want Grafana", got)
	}
}

func TestHTTPTitleDoesNotFollowRedirects(t *testing.T) {
	var internalRequests atomic.Int32
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		internalRequests.Add(1)
		fmt.Fprint(w, "<html><head><title>INTERNAL-SECRET-TITLE</title></head></html>")
	}))
	defer internal.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL+"/internal", http.StatusFound)
	}))
	defer redirect.Close()

	ip, port := serverAddr(t, redirect.URL)
	s, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.httpTitle(context.Background(), ip, port); got != "" {
		t.Fatalf("httpTitle followed a redirect and returned %q", got)
	}
	if n := internalRequests.Load(); n != 0 {
		t.Fatalf("httpTitle sent %d request(s) to the redirect target", n)
	}
}

func serverAddr(t *testing.T, raw string) (string, int) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse server URL %q: %v", raw, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse server port %q: %v", u.Port(), err)
	}
	return u.Hostname(), port
}
