package lb

import (
	"context"
	"encoding/json"
	"fmt"
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

// Response Generic response object for
// API endpoints
type Response struct {
	Message string `json:"message"`
}

// RegisterRequest Request body representation
// for the /register, and /deregister endpoints.
type RegisterRequest struct {
	URL string `json:"url"`
}

// ConfigRequest Request body representation
// for /config endpoint.
type ConfigRequest struct {
	HCFrequency *int64 `json:"hcFrequency"`
	Retries     *int   `json:"retries"`
	RetryDelay  *int   `json:"retryDelay"`
}

// Configuration Stores load balancer
// config and a reference to healthcheck
// ticker for frequency updates on the fly.
type Configuration struct {
	HCTicker   *time.Ticker
	Retries    int
	RetryDelay int
}

// LoadBalancer Contains pool of ServerNodes
// and load balancer configuration.
type LoadBalancer struct {
	pool   *ServerPool
	config *Configuration
}

// Start Creates an empty load balancer
// and spawn healthcheck routine.
func Start() *LoadBalancer {
	lb := &LoadBalancer{
		pool: NewPool(),
		config: &Configuration{
			time.NewTicker(time.Second * 60),
			3,
			1000,
		},
	}

	go lb.pool.HealthChecks(lb.config.HCTicker)
	return lb
}

// HttpHandler main load balancing handler
func (lb *LoadBalancer) HttpHandler(w http.ResponseWriter, r *http.Request) {
	node := lb.pool.GetNextNode()
	if node != nil {
		lb.pool.AddRequestToNode(node, r)
		node.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

// RegisterHandler handles requests for registersing hosts/nodes
// to the load balancer.
func (lb *LoadBalancer) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	lb.RegisterNode(req.URL)
	json.NewEncoder(w).Encode(Response{fmt.Sprintf("Successfully Registered: %v", req.URL)})
}

// DeregisterHandler handles request for removing a host/node from
// the load balancer.
func (lb *LoadBalancer) DeregisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	go lb.DeregisterNode(req.URL)
	json.NewEncoder(w).Encode(Response{fmt.Sprintf("Deregistering: %v", req.URL)})
}

// ConfigHanlder handles requests for effecting the global configuration of the load balancer
func (lb *LoadBalancer) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if req.Retries != nil {
		lb.config.Retries = *req.Retries
	}
	if req.HCFrequency != nil {
		lb.config.HCTicker.Reset(time.Second * time.Duration(*req.HCFrequency))
	}
	if req.RetryDelay != nil {
		lb.config.RetryDelay = *req.RetryDelay
	}

	json.NewEncoder(w).Encode(Response{"Configuration Updated."})
}

// DeregisterNode Finds and removes the node with matching URL from the queue.
func (lb *LoadBalancer) DeregisterNode(nodeURL string) {
	lb.pool.DeregisterNode(nodeURL)
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
		if retries < lb.config.Retries {
			select {
			case <-time.After(time.Duration(lb.config.RetryDelay) * time.Millisecond):
				log.Printf("%s(%s) Retry %d\n", request.RemoteAddr, request.URL.Path, retries+1)
				ctx := context.WithValue(request.Context(), Retry, retries+1)
				node.ReverseProxy.ServeHTTP(writer, request.WithContext(ctx))
			}
			return
		}

		attempts := GetAttemptsFromContext(request)
		if attempts >= lb.config.Retries {
			log.Printf("%s(%s) Max attempts reached, terminating\n", request.RemoteAddr, request.URL.Path)
			http.Error(writer, "Service not available", http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(request.Context(), Attempts, attempts+1)
		log.Printf("%s(%s) Starting Attempt %d\n", request.RemoteAddr, request.URL.Path, attempts+1)
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
