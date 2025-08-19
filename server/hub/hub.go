package hub

import (
	"context"
	"database/sql"
	"log"
	"sync"

	"github.com/go-redis/redis/v8"

	"lukagolubovic/client"
	"lukagolubovic/loadbalancer"
	"lukagolubovic/models"
)

const (
	redisChannel = "chat-messages"
)

type Hub struct {
	address     string
	clients     map[*client.Client]bool
	mu          sync.Mutex
	register    chan *client.Client
	unregister  chan *client.Client
	redisClient *redis.Client
	db          *sql.DB
	ctx         context.Context
	cancel      context.CancelFunc
	lbClient    *loadbalancer.Client
}

func New(address string, redisClient *redis.Client, db *sql.DB, lbClient *loadbalancer.Client) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		address:     address,
		clients:     make(map[*client.Client]bool),
		register:    make(chan *client.Client),
		unregister:  make(chan *client.Client),
		redisClient: redisClient,
		db:          db,
		ctx:         ctx,
		cancel:      cancel,
		lbClient:    lbClient,
	}
}

func (h *Hub) Run() {
	go h.listenToRedis()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			load := len(h.clients)
			h.mu.Unlock()

			log.Printf("[Server %s] Client '%s' connected. Total clients: %d\n", h.address, client.Username, load)
			h.lbClient.UpdateLoad(load)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.CloseOnce.Do(func() { close(client.Send) })
				load := len(h.clients)
				h.mu.Unlock()

				log.Printf("[Server %s] Client '%s' disconnected. Total clients: %d\n", h.address, client.Username, load)
				h.lbClient.UpdateLoad(load)
			} else {
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) listenToRedis() {
	pubsub := h.redisClient.Subscribe(h.ctx, redisChannel)
	defer pubsub.Close()
	ch := pubsub.Channel()

	for {
		select {
		case <-h.ctx.Done():
			return
		case rawMsg, ok := <-ch:
			if !ok {
				return
			}

			h.mu.Lock()
			var clientsToRemove []*client.Client
			for client := range h.clients {
				select {
				case client.Send <- []byte(rawMsg.Payload):
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
}

func (h *Hub) RegisterClient(c *client.Client) {
	h.register <- c
}

func (h *Hub) UnregisterClient(c *client.Client) {
	h.unregister <- c
}

func (h *Hub) GetAddress() string {
	return h.address
}

func (h *Hub) GetLoad() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *Hub) SaveMessage(msg models.Message) error {
	stmt, err := h.db.Prepare("INSERT INTO messages(username, message, server) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(msg.Username, msg.Content, msg.Server)
	return err
}

func (h *Hub) PublishMessage(msgBytes []byte) error {
	return h.redisClient.Publish(h.ctx, redisChannel, msgBytes).Err()
}