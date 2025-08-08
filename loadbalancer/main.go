package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type ChatServerInfo struct {
	Address string `json: "address"`
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

func (lb *LoadBalancer) registerServer(w http.ResponseWriter, r *http.Request) {
	var s ChatServerInfo
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	lb.mu.Lock()
	lb.servers[s.Address] = &ChatServerInfo{Address: s.Address, Load: s.Load}
	lb.mu.Unlock()
	log.Printf("[LB] registered %s (load=%d)\n", s.Address, s.Load)
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
		log.Printf("[LB] update %s -> load=%d\n", s.Address, s.Load)
	} else {
		lb.servers[s.Address] = &ChatServerInfo{Address: s.Address, Load: s.Load}
		log.Printf("[LB] (auto-register) %s -> load=%d\n", s.Address, s.Load)
	}
	lb.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (lb *LoadBalancer) getServer(w http.ResponseWriter, r *http.Request) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	if len(lb.servers) == 0 {
		http.Error(w, "no servers", http.StatusServiceUnavailable)
		return
	}
	var best *ChatServerInfo
	for _, s := range lb.servers {
		if best == nil || s.Load < best.Load {
			best = s
		}
	}
	if best == nil {
		http.Error(w, "no servers", http.StatusServiceUnavailable)
		return
	}
	if err := json.NewEncoder(w).Encode(best); err != nil {
		http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	lb := NewLoadBalancer()
	http.HandleFunc("/register", lb.registerServer)
	http.HandleFunc("/update", lb.updateServer)
	http.HandleFunc("/get", lb.getServer)

	log.Println("[LB] running on :9000")
	log.Fatal(http.ListenAndServe(":9000", nil))
}
