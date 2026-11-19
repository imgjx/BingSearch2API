package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"bing-search-api/internal/bing"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

var requestTimeout = 25 * time.Second

// Server 是 HTTP 服务封装。
type Server struct {
	client  *bing.Client
	limiter *rateLimiter
}

// NewServer 创建并返回一个带中间件的 http.Handler。
func NewServer(client *bing.Client, ratePerSec int) http.Handler {
	s := &Server{client: client}
	s.limiter = newRateLimiter(ratePerSec, ratePerSec*2+5)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/search/web", s.handleVertical("web"))
	mux.HandleFunc("/search/images", s.handleVertical("images"))
	mux.HandleFunc("/search/videos", s.handleVertical("videos"))
	mux.HandleFunc("/search/news", s.handleVertical("news"))
	mux.HandleFunc("/search/all", s.handleAll)
	mux.HandleFunc("/suggest", s.handleSuggest)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, r, http.StatusOK, map[string]interface{}{
				"service": "bing-search-api",
				"endpoints": []string{
					"GET/POST /search/web", "GET/POST /search/images",
					"GET/POST /search/videos", "GET/POST /search/news",
					"GET/POST /search/all", "GET/POST /suggest", "GET /health",
				},
			})
			return
		}
		writeError(w, r, http.StatusNotFound, "endpoint not found")
	})

	var h http.Handler = mux
	h = s.logging(h)
	h = s.rateLimit(h)
	h = cors(h)
	h = requestIDMiddleware(h)
	h = recoverMiddleware(h)
	return h
}

// SetRequestTimeout 设置单次 API 请求的整体超时。
func SetRequestTimeout(d time.Duration) {
	if d > 0 {
		requestTimeout = d
	}
}

func withTimeout(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), requestTimeout)
}

func requestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// ---------- 中间件 ----------

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[panic] request_id=%s path=%s err=%v", requestIDFrom(r.Context()), r.URL.Path, err)
				writeJSON(w, r, http.StatusInternalServerError, response{
					Success: false, Error: fmt.Sprintf("internal server error: %v", err), RequestID: requestIDFrom(r.Context()),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rw *statusRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *statusRecorder) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		if rw.status == 0 {
			rw.status = http.StatusOK
		}
		log.Printf("[%s] %s %s status=%d duration=%s rid=%s",
			r.Method, r.URL.Path, r.URL.RawQuery, rw.status, time.Since(start), requestIDFrom(r.Context()),
		)
	})
}

// ---------- 简易令牌桶限流（按客户端 IP） ----------

type bucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	rate     float64 // 每秒补充的令牌数
}

func newRateLimiter(perSec int, capacity int) *rateLimiter {
	if perSec <= 0 {
		perSec = 20
	}
	return &rateLimiter{
		buckets:  map[string]*bucket{},
		capacity: float64(capacity),
		rate:     float64(perSec),
	}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	// 清理长时间不活跃的桶，避免无限增长
	if len(l.buckets) > 4096 {
		for k, v := range l.buckets {
			if now.Sub(v.last) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
	}
	return true
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if !s.limiter.allow(host) {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, r, http.StatusTooManyRequests, response{
				Success: false, Error: "rate limit exceeded, please retry later", RequestID: requestIDFrom(r.Context()),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
