package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type ChatServerInfo struct {
	Address string `json:"Address"`
	Load    int    `json:"load"`
}

type LoadBalancer struct {
	mu      sync.Mutex
	servers map[string]*ChatServerInfo
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		servers: make(map[string]*ChatServerInfo),
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (lb *LoadBalancer) registerServer(w http.ResponseWriter, r *http.Request) {
	var s ChatServerInfo
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	lb.mu.Lock()
	lb.servers[s.Address] = &ChatServerInfo{Address: s.Address, Load: s.Load}
	lb.mu.Unlock()
	log.Printf("[LB] Registered server %s with initial load %d\n", s.Address, s.Load)
	w.WriteHeader(http.StatusOK)
}

func (lb *LoadBalancer) updateServer(w http.ResponseWriter, r *http.Request) {
	var s ChatServerInfo
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	lb.mu.Lock()
	if existing, ok := lb.servers[s.Address]; ok {
		existing.Load = s.Load
	} else {
		lb.servers[s.Address] = &ChatServerInfo{Address: s.Address, Load: s.Load}
	}
	lb.mu.Unlock()
	log.Printf("[LB] Updated server %s load to %d\n", s.Address, s.Load)
	w.WriteHeader(http.StatusOK)
}

func (lb *LoadBalancer) getServer(w http.ResponseWriter, r *http.Request) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.servers) == 0 {
		http.Error(w, "no available servers", http.StatusServiceUnavailable)
		return
	}

	var bestServer *ChatServerInfo
	for _, s := range lb.servers {
		if bestServer == nil || s.Load < bestServer.Load {
			bestServer = s
		}
	}

	if bestServer == nil {
		http.Error(w, "could not determine best server", http.StatusInternalServerError)
		return
	}

	log.Printf("[LB] Directing client to server %s (load=%d)\n", bestServer.Address, bestServer.Load)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bestServer); err != nil {
		http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	lb := NewLoadBalancer()

	mux := http.NewServeMux()
	mux.HandleFunc("/register", lb.registerServer)
	mux.HandleFunc("/update", lb.updateServer)
	mux.HandleFunc("/get", lb.getServer)

	handler := corsMiddleware(mux)

	log.Println("[LB] Load Balancer is running on :9000")
	if err := http.ListenAndServe(":9000", handler); err != nil {
		log.Fatalf("Failed to start load balancer: %v", err)
	}
}
