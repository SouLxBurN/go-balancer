package main

import (
	"context"
	"go-balancer/lb"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	loadBalancer := lb.Start()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	mux := http.NewServeMux()
	mux.HandleFunc("/config", loadBalancer.ConfigHandler)
	mux.HandleFunc("/register", loadBalancer.RegisterHandler)
	mux.HandleFunc("/deregister", loadBalancer.DeregisterHandler)
	configServer := &http.Server{
		Addr:    ":4501",
		Handler: mux,
	}
	httpsServer := &http.Server{
		Addr:    ":443",
		Handler: http.HandlerFunc(loadBalancer.HttpHandler),
	}
	redirectTLS := &http.Server{
		Addr: ":80",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
		}),
	}

	go startServer(redirectTLS, "Redirect TLS Server")
	go startTLSServer(configServer, "Config Server")
	go startTLSServer(httpsServer, "Load Balancer")

	<-terminate

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := configServer.Shutdown(ctx); err != nil {
		log.Fatalf("Configuration Server Shutdown Failed: %+v", err)
	}
	if err := httpsServer.Shutdown(ctx); err != nil {
		log.Fatalf("Load Balancer Server Shutdown Failed: %+v", err)
	}
	if err := redirectTLS.Shutdown(ctx); err != nil {
		log.Fatalf("Redirect TLS Server Shutdown Failed: %+v", err)
	}

	log.Print("Server Shutdown Completed Successfully")
}

// startServer Starts a unsecured named *http.Server with error handling.
func startServer(server *http.Server, serverName string) {
	log.Printf("Running %s on port %s\n", serverName, server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("%s Server Closed Unexpectedly.\n%s\n", serverName, err)
	}
}

// startTLSServer Starts a secured named *http.Server with error handling.
func startTLSServer(server *http.Server, serverName string) {
	log.Printf("Running %s on port %s\n", serverName, server.Addr)
	if err := server.ListenAndServeTLS("/root/certs/MyCertificate.crt", "/root/certs/MyKey.key"); err != nil && err != http.ErrServerClosed {
		log.Fatalf("%s Server Closed Unexpectedly.\n%s\n", serverName, err)
	}
}
