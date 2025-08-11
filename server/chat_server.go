package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

const (
	redisChannel = "chat-messages"
	lbURL        = "http://127.0.0.1:9000"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	username string
}

type Hub struct {
	address     string
	clients     map[*Client]bool
	mu          sync.Mutex
	register    chan *Client
	unregister  chan *Client
	redisClient *redis.Client
	ctx         context.Context
}

type Message struct {
	Username string `json:"username"`
	Content  string `json:"content"`
	Server   string `json:"server"`
}

func newHub(address string, redisClient *redis.Client) *Hub {
	return &Hub{
		address:     address,
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		redisClient: redisClient,
		ctx:         context.Background(),
	}
}

func (h *Hub) run() {
	go h.listenToRedis()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[Server %s] Client '%s' connected. Total clients: %d\n", h.address, client.username, h.getLoad())
			h.updateLBLoad()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[Server %s] Client '%s' disconnected. Total clients: %d\n", h.address, client.username, h.getLoad())
			}
			h.mu.Unlock()
			h.updateLBLoad()
		}
	}
}

func (h *Hub) listenToRedis() {
	pubsub := h.redisClient.Subscribe(h.ctx, redisChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Printf("[Server %s] Subscribed to Redis channel '%s'\n", h.address, redisChannel)

	for msg := range ch {
		h.mu.Lock()
		for client := range h.clients {
			select {
			case client.send <- []byte(msg.Payload):
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		msg := Message{
			Username: c.username,
			Content:  string(message),
			Server:   c.hub.address,
		}
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Error marshalling message: %v", err)
			continue
		}

		if err := c.hub.redisClient.Publish(c.hub.ctx, redisChannel, msgBytes).Err(); err != nil {
			log.Printf("Error publishing to Redis: %v", err)
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (h *Hub) getLoad() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) updateLBLoad() {
	load := h.getLoad()
	payload := map[string]interface{}{
		"address": h.address,
		"load":    load,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/update", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Printf("[Server %s] Failed to update load: %v\n", h.address, err)
		return
	}
	resp.Body.Close()
}

func (h *Hub) registerWithLB() {
	payload := map[string]interface{}{
		"address": h.address,
		"load":    0,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/register", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("[Server %s] Failed to register with LB: %v", h.address, err)
	}
	resp.Body.Close()
	log.Printf("[Server %s] Successfully registered with Load Balancer\n", h.address)
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		username: username,
	}
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func main() {
	host := flag.String("host", "127.0.0.1", "Host to run the server on")
	port := flag.Int("port", 8080, "Port to run the server on")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	flag.Parse()

	address := fmt.Sprintf("ws://%s:%d", *host, *port)

	redisClient := redis.NewClient(&redis.Options{
		Addr: *redisAddr,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis on %s: %v", *redisAddr, err)
	}

	hub := newHub(address, redisClient)

	hub.registerWithLB()

	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	listenAddr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("[ChatServer] starting on %s, connecting to Redis on %s\n", listenAddr, *redisAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}
