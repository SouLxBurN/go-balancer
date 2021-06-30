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
	pool *ServerPool
}

// Start Creates an empty load balancer
// and spawn healthcheck routine.
func Start() *LoadBalancer {
	lb := &LoadBalancer{
		pool: &ServerPool{
			queue: []*ServerNode{},
		},
	}

	go lb.pool.HealthChecks()
	return lb
}

// HttpHandler main load balancing handler
func (lb *LoadBalancer) HttpHandler(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}
	peer := lb.pool.GetNextNode()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// ConfigHandler configuration handler for registering
// and deregistering nodes among other things.
func (lb *LoadBalancer) ConfigHandler(w http.ResponseWriter, r *http.Request) {

}

// RegisterNode Creates and registers a ServerNode with
// the provided nodeURL
func (lb *LoadBalancer) RegisterNode(nodeURL string) {
	node := &ServerNode{Alive: true}
	node.URL, _ = url.Parse(nodeURL)
	node.ReverseProxy = httputil.NewSingleHostReverseProxy(node.URL)
	node.ReverseProxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, err error) {
		log.Printf("[%s] %s\n", node.URL.Host, err.Error())
		retries := GetRetryFromContext(request)
		if retries < 3 {
			select {
			case <-time.After(10 * time.Millisecond):
				ctx := context.WithValue(request.Context(), Retry, retries+1)
				node.ReverseProxy.ServeHTTP(writer, request.WithContext(ctx))
			}
			return
		}
		node.SetAlive(false)

		attempts := GetAttemptsFromContext(request)
		log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
		ctx := context.WithValue(request.Context(), Attempts, attempts+1)
		lb.HttpHandler(writer, request.WithContext(ctx))
	}
	lb.pool.RegisterNode(node)
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
