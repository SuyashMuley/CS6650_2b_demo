package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Product mirrors the Product schema in api.yaml.
// Pointer fields let us detect missing required fields (nil) vs. zero values.
type Product struct {
	ProductID    *int32  `json:"product_id"`
	SKU          *string `json:"sku"`
	Manufacturer *string `json:"manufacturer"`
	CategoryID   *int32  `json:"category_id"`
	Weight       *int32  `json:"weight"`
	SomeOtherID  *int32  `json:"some_other_id"`
}

// ErrorResponse mirrors the Error schema in api.yaml.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// store is an in-memory product store. The RWMutex allows many concurrent
// readers (GET) while serializing writers (POST).
type store struct {
	mu       sync.RWMutex
	products map[int32]Product
}

func newStore() *store {
	s := &store{products: make(map[int32]Product)}
	s.seed()
	return s
}

// seed preloads products so GET/POST can succeed: POST only updates existing
// products (404 otherwise), so something must exist first.
func (s *store) seed() {
	items := []struct {
		id    int32
		sku   string
		manu  string
		cat   int32
		w     int32
		other int32
	}{
		{1, "ABC-123-XYZ", "Acme Corporation", 456, 1250, 789},
		{2, "WID-200-BLU", "Widgets Inc", 12, 300, 55},
		{3, "GAD-300-RED", "Gadget Co", 7, 4500, 91},
	}
	for _, it := range items {
		id, sku, manu, cat, w, other := it.id, it.sku, it.manu, it.cat, it.w, it.other
		s.products[id] = Product{&id, &sku, &manu, &cat, &w, &other}
	}
}

func (s *store) get(id int32) (Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

// update replaces an existing product; returns false if it doesn't exist.
func (s *store) update(id int32, p Product) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.products[id]; !ok {
		return false
	}
	s.products[id] = p
	return true
}

func errJSON(c *gin.Context, status int, code, msg, details string) {
	c.JSON(status, ErrorResponse{Error: code, Message: msg, Details: details})
}

// parseProductID parses the path parameter as an int32 >= 1.
func parseProductID(raw string) (int32, bool) {
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 1 {
		return 0, false
	}
	return int32(n), true
}

// validate checks a Product against the constraints in api.yaml.
// Returns an empty string when valid, otherwise a description of the problem.
func validate(p Product) string {
	switch {
	case p.ProductID == nil || *p.ProductID < 1:
		return "product_id is required and must be an integer >= 1"
	case p.SKU == nil || len(*p.SKU) < 1 || len(*p.SKU) > 100:
		return "sku is required and must be 1-100 characters"
	case p.Manufacturer == nil || len(*p.Manufacturer) < 1 || len(*p.Manufacturer) > 200:
		return "manufacturer is required and must be 1-200 characters"
	case p.CategoryID == nil || *p.CategoryID < 1:
		return "category_id is required and must be an integer >= 1"
	case p.Weight == nil || *p.Weight < 0:
		return "weight is required and must be an integer >= 0"
	case p.SomeOtherID == nil || *p.SomeOtherID < 1:
		return "some_other_id is required and must be an integer >= 1"
	}
	return ""
}

// simulateFailure lets us exercise the 500 path on demand:
// send the header "X-Simulate-Error: true".
func simulateFailure(c *gin.Context) bool {
	return c.GetHeader("X-Simulate-Error") == "true"
}

// RED metrics (Rate, Errors, Duration), labelled by method, route pattern and status.
// Labelling by the route pattern (/products/:productId) instead of the raw URL
// keeps the number of label values small.
var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests (rate).",
	}, []string{"method", "endpoint", "status"})

	errorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "HTTP requests that returned a non-2xx status (errors).",
	}, []string{"method", "endpoint", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds (duration).",
		Buckets: []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
	}, []string{"method", "endpoint"})
)

// logLine is the one-JSON-object-per-request structure written to stdout.
type logLine struct {
	Time       string  `json:"time"`
	Method     string  `json:"method"`
	Endpoint   string  `json:"endpoint"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMs float64 `json:"duration_ms"`
	ClientIP   string  `json:"client_ip"`
}

// observe records the RED metrics and emits a structured JSON log line for every request.
func observe() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = "unmatched"
		}
		status := c.Writer.Status()
		code := strconv.Itoa(status)

		requestsTotal.WithLabelValues(c.Request.Method, endpoint, code).Inc()
		if status < 200 || status > 299 {
			errorsTotal.WithLabelValues(c.Request.Method, endpoint, code).Inc()
		}
		requestDuration.WithLabelValues(c.Request.Method, endpoint).Observe(elapsed.Seconds())

		line, err := json.Marshal(logLine{
			Time:       start.UTC().Format(time.RFC3339Nano),
			Method:     c.Request.Method,
			Endpoint:   endpoint,
			Path:       c.Request.URL.Path,
			Status:     status,
			DurationMs: float64(elapsed.Microseconds()) / 1000,
			ClientIP:   c.ClientIP(),
		})
		if err == nil {
			os.Stdout.Write(append(line, '\n'))
		}
	}
}

func setupRouter(s *store) *gin.Engine {
	r := gin.New()
	// observe is outermost so it still sees the 500 that Recovery writes after a panic.
	r.Use(observe(), gin.Recovery())

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	r.GET("/products/:productId", func(c *gin.Context) {
		if simulateFailure(c) {
			errJSON(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", "simulated failure")
			return
		}
		id, ok := parseProductID(c.Param("productId"))
		if !ok {
			// api.yaml lists only 200/404/500 for GET, so an invalid ID
			// can never match a product: 404.
			errJSON(c, http.StatusNotFound, "NOT_FOUND", "Product not found", "Product ID must be a positive integer")
			return
		}
		p, found := s.get(id)
		if !found {
			errJSON(c, http.StatusNotFound, "NOT_FOUND", "Product not found", "No product with the given ID")
			return
		}
		c.JSON(http.StatusOK, p)
	})

	r.POST("/products/:productId/details", func(c *gin.Context) {
		if simulateFailure(c) {
			errJSON(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", "simulated failure")
			return
		}
		id, ok := parseProductID(c.Param("productId"))
		if !ok {
			errJSON(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid", "Product ID must be a positive integer")
			return
		}
		var p Product
		if err := c.ShouldBindJSON(&p); err != nil {
			errJSON(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid", "Malformed JSON or wrong field types")
			return
		}
		if msg := validate(p); msg != "" {
			errJSON(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid", msg)
			return
		}
		if *p.ProductID != id {
			errJSON(c, http.StatusBadRequest, "INVALID_INPUT", "The provided input data is invalid", "product_id in body must match productId in path")
			return
		}
		if !s.update(id, p) {
			errJSON(c, http.StatusNotFound, "NOT_FOUND", "Product not found", "No product with the given ID")
			return
		}
		c.Status(http.StatusNoContent)
	})

	return r
}

func main() {
	setupRouter(newStore()).Run(":8080")
}
