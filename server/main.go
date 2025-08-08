package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"database/sql"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var lbURL = "http://127.0.0.1:9000"

type Client struct {
	username string
	conn     *websocket.Conn
	server   *Server
	send     chan []byte
}

type Server struct {
	address    string
	clients    map[*Client]bool
	mu         sync.Mutex
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	db         *sql.DB
	logFile    *os.File
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func newServer(address string) *Server {
	db, err := sql.Open("sqlite3", "./chat.db")
	if err != nil {
		log.Fatal("db open:", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal("db create:", err)
	}

	logFile, err := os.OpenFile("chat.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("open chat.log:", err)
	}

	return &Server{
		address:    address,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		db:         db,
		logFile:    logFile,
	}
}

func (s *Server) run() {
	for {
		select {
		case c := <-s.register:
			s.mu.Lock()
			s.clients[c] = true
			s.mu.Unlock()
			s.updateLBLoad()

		case c := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[c]; ok {
				delete(s.clients, c)
				close(c.send)
			}
			s.mu.Unlock()
			s.updateLBLoad()

		case msg := <-s.broadcast:
			s.mu.Lock()
			for c := range s.clients {
				select {
				case c.send <- msg:
				default:
					close(c.send)
					delete(s.clients, c)
				}
			}
			s.mu.Unlock()
		}
	}
}

func (s *Server) updateLBLoad() {
	s.mu.Lock()
	load := len(s.clients)
	addr := s.address
	s.mu.Unlock()

	payload := map[string]interface{}{
		"address": addr,
		"load":    load,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/update", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Println("[Server] updateLBLoad err:", err)
		return
	}
	resp.Body.Close()
}

func (s *Server) registerWithLB() {
	payload := map[string]interface{}{
		"address": s.address,
		"load":    0,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/register", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Println("[Server] registerWithLB err:", err)
		return
	}
	resp.Body.Close()
	log.Println("[Server] registered with LB:", s.address)
}

func (c *Client) readPump() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		_, err = c.server.db.Exec("INSERT INTO messages (username, message) VALUES (?, ?)", c.username, string(msg))
		if err != nil {
			log.Println("db insert:", err)
		}
		_, _ = c.server.logFile.WriteString(fmt.Sprintf("%s: %s\n", c.username, string(msg)))
		c.server.broadcast <- []byte(fmt.Sprintf("%s: %s", c.username, string(msg)))
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWS(s *Server, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	client := &Client{
		username: username,
		conn:     conn,
		server:   s,
		send:     make(chan []byte, 256),
	}
	s.register <- client
	go client.writePump()
	go client.readPump()
}

func main() {
	host := flag.String("host", "127.0.0.1", "host to listen on")
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	address := fmt.Sprintf("ws://%s:%d", *host, *port)
	s := newServer(address)

	s.registerWithLB()

	go s.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(s, w, r)
	})

	log.Printf("[ChatServer] listening on %s (ws endpoint /ws)\n", address)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), nil))
}
