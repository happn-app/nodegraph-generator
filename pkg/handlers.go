package pkg

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type GraphResponse struct {
	Nodes []*NodeResult `json:"nodes"`
	Edges []*EdgeResult `json:"edges"`
}

func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	// For now, if the process is running and VM client was constructed at startup,
	// we consider the service "ready".
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready\n"))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Simple liveness probe; you can add deeper checks if you want.
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("not found\n"))
}


func GraphHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	env := q.Get("env")
	if env == "" {
		env = "preprod"
	}
	startService := strings.TrimSpace(q.Get("service"))

	maxDepth := 0
	if mdStr := q.Get("maxDepth"); mdStr != "" {
		if md, err := strconv.Atoi(mdStr); err == nil && md > 0 {
			maxDepth = md
		} else {
			log.Printf("[graph] invalid maxDepth=%q: %v", mdStr, err)
		}
	}
	if startService != "" && maxDepth == 0 {
		maxDepth = 2
	}

	log.Printf("[graph] request env=%q service=%q maxDepth=%d", env, startService, maxDepth)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client, err := New(os.Getenv("VM_HOST"), ctx)
	if err != nil {
		log.Fatalf("Could not create Prometheus client: %s", err)
		http.Error(w, "failed to create VM client", http.StatusInternalServerError)
		return
	}

	nodes, edges, err := client.GetFullGraph(env)
	if err != nil {
		log.Printf("[graph] GetFullGraph error: %v", err)
		http.Error(w, "failed to get full graph", http.StatusInternalServerError)
		return
	}
	err = client.EnrichGraph(nodes, edges, env)
	if err != nil {
		log.Printf("[graph] EnrichGraph error: %v", err)
		http.Error(w, "failed to enrich graph", http.StatusInternalServerError)
		return
	}


	if len(edges) == 0 {
		resp := GraphResponse{Nodes: []*NodeResult{}, Edges: []*EdgeResult{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("[graph] response nodes=%d edges=%d", len(nodes), len(edges))
	nodeSlice := slices.Collect(maps.Values(nodes))
	edgeSlice := slices.Collect(maps.Values(edges))

	resp := GraphResponse{Nodes: nodeSlice, Edges: edgeSlice}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("[graph] encode response error: %v", err)
	}
}
