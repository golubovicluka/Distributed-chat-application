package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"

	"lukagolubovic/database"
	"lukagolubovic/handlers"
	"lukagolubovic/hub"
	"lukagolubovic/loadbalancer"
	"lukagolubovic/middleware"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Host to run the server on")
	port := flag.Int("port", 8080, "Port to run the server on")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	address := fmt.Sprintf("ws://%s:%d", *host, *port)

	db, err := database.InitDB("./chat.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: *redisAddr,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis on %s: %v", *redisAddr, err)
	}

	lbClient := loadbalancer.New(address)
	lbClient.Register()

	hub := hub.New(address, redisClient, db, lbClient)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/history", handlers.GetHistory(db))
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeWS(hub, w, r)
	})

	handler := middleware.CORS(mux)

	listenAddr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("[ChatServer] starting on %s, serving /ws and /history\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, handler))
}