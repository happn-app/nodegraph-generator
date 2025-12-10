package main

import (
	"log"
	"net/http"

	"github.com/sbordeyne/nodegraph-generator/pkg"
)


func main() {
	config := pkg.LoadConfig()
	mux := http.NewServeMux()
	mux.HandleFunc("/graph", pkg.GraphHandler)
	mux.HandleFunc("/readyz", pkg.ReadyHandler)
	mux.HandleFunc("/healthz", pkg.HealthHandler)

	log.Printf("listening on %s", config.Host)
	if err := http.ListenAndServe(config.Host, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
