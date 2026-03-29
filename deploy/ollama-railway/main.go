package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Fatal("API_KEY environment variable is required")
	}
	ollamaHost := os.Getenv("OLLAMA_HOST_URL")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	target, err := url.Parse(ollamaHost)
	if err != nil {
		log.Fatalf("invalid OLLAMA_HOST_URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	limiter := newRateLimiter(30, time.Minute)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","model":"tinyllama"}`))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		token := r.Header.Get("Authorization")
		if token != "Bearer "+apiKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Rate limit by IP
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = fwd
		}
		if !limiter.allow(ip) {
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}

		// Strip auth header before forwarding to Ollama
		r.Header.Del("Authorization")
		proxy.ServeHTTP(w, r)
	})

	log.Printf("proxy listening on :%s -> %s", port, ollamaHost)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// rateLimiter implements a fixed-window per-IP rate limiter.
type rateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	limit   int
	clients map[string]*clientWindow
}

type clientWindow struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		window:  window,
		limit:   limit,
		clients: make(map[string]*clientWindow),
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cw, ok := rl.clients[ip]
	if !ok || now.After(cw.resetAt) {
		rl.clients[ip] = &clientWindow{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if cw.count >= rl.limit {
		return false
	}
	cw.count++
	return true
}
