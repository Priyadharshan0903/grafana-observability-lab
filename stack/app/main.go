// checkout is a fake service that emits the same kind of HTTP metrics a real
// service would, so SLOs, burn rates and dashboards have something to chew on.
//
// It generates its own traffic and lets you inject failure at runtime:
//
//	curl 'localhost:8091/admin/chaos?error_rate=0.05&slow_rate=0.2'
//	curl 'localhost:8091/admin/chaos'   # show current settings
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests by route and status code.",
	}, []string{"route", "code"})

	// Buckets include 0.3 so "requests faster than 300ms" is an exact bucket,
	// which is what a latency SLI needs.
	duration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: []float64{0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2.5},
	}, []string{"route"})
)

// chaos holds the failure knobs; both are probabilities in [0,1].
type chaos struct {
	mu        sync.RWMutex
	errorRate float64 // chance a request returns 500
	slowRate  float64 // chance a request takes 400ms–1.5s instead of ~40ms
}

func (c *chaos) get() (float64, float64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errorRate, c.slowRate
}

func main() {
	c := &chaos{
		errorRate: envFloat("ERROR_RATE", 0.001),
		slowRate:  envFloat("SLOW_RATE", 0.005),
	}
	rps := envFloat("RPS", 20)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/checkout", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		errorRate, slowRate := c.get()

		// Normal requests take 20–60ms; slow ones 400ms–1.5s.
		d := time.Duration(20+rand.IntN(40)) * time.Millisecond
		if rand.Float64() < slowRate {
			d = time.Duration(400+rand.IntN(1100)) * time.Millisecond
		}
		time.Sleep(d)

		code := http.StatusOK
		if rand.Float64() < errorRate {
			code = http.StatusInternalServerError
		}
		w.WriteHeader(code)

		requests.WithLabelValues("/api/checkout", strconv.Itoa(code)).Inc()
		duration.WithLabelValues("/api/checkout").Observe(time.Since(start).Seconds())
	})

	mux.HandleFunc("/admin/chaos", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		c.mu.Lock()
		if v, err := strconv.ParseFloat(q.Get("error_rate"), 64); err == nil {
			c.errorRate = v
		}
		if v, err := strconv.ParseFloat(q.Get("slow_rate"), 64); err == nil {
			c.slowRate = v
		}
		fmt.Fprintf(w, "error_rate=%g slow_rate=%g\n", c.errorRate, c.slowRate)
		c.mu.Unlock()
	})

	mux.Handle("/metrics", promhttp.Handler())

	go generateLoad(rps)

	log.Printf("checkout listening on :8080 (rps=%g)", rps)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// generateLoad calls our own checkout endpoint at roughly rps requests/sec.
func generateLoad(rps float64) {
	client := &http.Client{Timeout: 5 * time.Second}
	tick := time.NewTicker(time.Duration(float64(time.Second) / rps))
	for range tick.C {
		go func() {
			resp, err := client.Get("http://localhost:8080/api/checkout")
			if err == nil {
				resp.Body.Close()
			}
		}()
	}
}

func envFloat(key string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(key), 64); err == nil {
		return v
	}
	return def
}
