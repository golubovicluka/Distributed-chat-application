package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const (
	redisChannel = "chat-messages"
	lbURL        = "http://127.0.0.1:9000"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
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
	db          *sql.DB
	ctx         context.Context
}

type Message struct {
	ID        int64  `json:"id,omitempty"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	Server    string `json:"server,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

func newHub(address string, redisClient *redis.Client, db *sql.DB) *Hub {
	return &Hub{
		address:     address,
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		redisClient: redisClient,
		db:          db,
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
				select {
				case <-client.send:
				default:
					close(client.send)
				}
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

	for rawMsg := range ch {
		h.mu.Lock()
		var clientsToRemove []*Client
		for client := range h.clients {
			select {
			case client.send <- []byte(rawMsg.Payload):
			default:
				clientsToRemove = append(clientsToRemove, client)
			}
		}
		h.mu.Unlock()

		for _, client := range clientsToRemove {
			h.unregister <- client
		}
	}
}

func (h *Hub) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, username, message, server, timestamp FROM messages ORDER BY timestamp DESC LIMIT 50")
	if err != nil {
		http.Error(w, "Failed to retrieve message history", http.StatusInternalServerError)
		log.Printf("DB query error: %v", err)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Username, &msg.Content, &msg.Server, &msg.Timestamp); err != nil {
			http.Error(w, "Failed to scan message row", http.StatusInternalServerError)
			log.Printf("DB scan error: %v", err)
			return
		}
		messages = append(messages, msg)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetPongHandler(func(string) error { return nil })

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("[Server %s] Client '%s' unexpected close error: %v", c.hub.address, c.username, err)
			} else {
				log.Printf("[Server %s] Client '%s' disconnected normally", c.hub.address, c.username)
			}
			break
		}

		var incomingMsg Message
		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			log.Printf("Error parsing incoming message JSON: %v", err)
			continue
		}

		msg := Message{
			Username: c.username,
			Content:  incomingMsg.Content,
			Server:   c.hub.address,
		}

		stmt, err := c.hub.db.Prepare("INSERT INTO messages(username, message, server) VALUES(?, ?, ?)")
		if err != nil {
			log.Printf("Error preparing db statement: %v", err)
			continue
		}
		_, err = stmt.Exec(msg.Username, msg.Content, msg.Server)
		if err != nil {
			log.Printf("Error executing db statement: %v", err)
			stmt.Close()
			continue
		}
		stmt.Close()

		msgBytes, _ := json.Marshal(msg)
		if err := c.hub.redisClient.Publish(c.hub.ctx, redisChannel, msgBytes).Err(); err != nil {
			log.Printf("Error publishing to Redis: %v", err)
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Server %s] Client '%s' write error: %v", c.hub.address, c.username, err)
				return
			}
		}
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

	db, err := sql.Open("sqlite3", "./chat.db?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS messages (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT,
		"message" TEXT,
		"server" TEXT,
		"timestamp" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: *redisAddr,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("Could not connect to Redis on %s: %v", *redisAddr, err)
	}

	hub := newHub(address, redisClient, db)
	hub.registerWithLB()
	go hub.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/history", hub.handleGetHistory)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	handler := corsMiddleware(mux)

	listenAddr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("[ChatServer] starting on %s, serving /ws and /history\n", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, handler))
}
