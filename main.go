package main

import (
	"log"
	"net/http"
	"github.com/sbordeyne/nodegraph-generator/pkg"
)


func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/graph", pkg.GraphHandler)
	mux.HandleFunc("/readyz", pkg.ReadyHandler)
	mux.HandleFunc("/healthz", pkg.HealthHandler)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
