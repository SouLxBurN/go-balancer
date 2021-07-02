package main

import (
	"fmt"
	"go-balance/lb"
	"net/http"
)

func main() {
	loadBalancer := lb.Start()
	startConfigServer(loadBalancer)
	startLoadBalancer(loadBalancer)
}

// startConfigServer Starts the config server
// in a separate goroutine.
func startConfigServer(loadBalancer *lb.LoadBalancer) {
	mux := http.NewServeMux()
	mux.HandleFunc("/register", loadBalancer.RegisterHandler)
	mux.HandleFunc("/deregister", loadBalancer.DeregisterHandler)
	configServer := http.Server{
		Addr:    ":4501",
		Handler: mux,
	}
	go configServer.ListenAndServe()
}

// startLoadBalancer Starts the http server and locks
// the current thread with `server.ListenAndServe()`
func startLoadBalancer(loadBalancer *lb.LoadBalancer) {
	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(loadBalancer.HttpHandler),
	}
	fmt.Printf("Running Load Balancer on %s\n", server.Addr)
	server.ListenAndServe()
}
