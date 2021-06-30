package lb

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const (
	Attempts = iota
	Retry
)

type LoadBalancer struct {
	pool ServerPool
}

// Start Creates an empty load balancer
// and spawn healthcheck routine.
func Start() LoadBalancer {
	lb := LoadBalancer{
		pool: ServerPool{
			nodes: []*ServerNode{},
		},
	}

	go lb.pool.HealthChecks()
	return lb
}

// LBHandler main Load Balancing Function
func (lb *LoadBalancer) HttpHandler(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	peer := lb.pool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// RegisterNode Creates and registers a ServerNode with
// the provided nodeURL
func (lb *LoadBalancer) RegisterNode(nodeURL string) {
	u, _ := url.Parse(nodeURL)
	rp := httputil.NewSingleHostReverseProxy(u)
	rp.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		log.Printf("[%s] %s\n", u.Host, err.Error())
		retries := GetRetryFromContext(request)
		if retries < 3 {
			select {
			case <-time.After(10 * time.Millisecond):
				ctx := context.WithValue(request.Context(), Retry, retries+1)
				rp.ServeHTTP(writer, request.WithContext(ctx))
			}
			return
		}
		lb.pool.MarkBackendStatus(u, false)

		attempts := GetAttemptsFromContext(request)
		log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
		ctx := context.WithValue(request.Context(), Attempts, attempts+1)
		lb.HttpHandler(writer, request.WithContext(ctx))
	}
	lb.pool.RegisterNode(&ServerNode{
		URL:          u,
		Alive:        true,
		ReverseProxy: rp,
	})
}

// GetRetryFromContext returns the number of retries for a request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

// GetAttemptsFromContext returns the number of attempts for a request.
func GetAttemptsFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Attempts).(int); ok {
		return retry
	}
	return 0
}
