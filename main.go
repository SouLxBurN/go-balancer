package main

import (
	"fmt"
	"go-balance/lb"
	"net/http"
)

func main() {
	loadBalancer := lb.Start()
	loadBalancer.RegisterNode("http://localhost:3000")

	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(loadBalancer.HttpHandler),
	}

	fmt.Printf("Running Load Balancer on %s\n", server.Addr)
	server.ListenAndServe()
}