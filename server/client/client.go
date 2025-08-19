package client

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"lukagolubovic/models"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	Hub       HubInterface
	Conn      *websocket.Conn
	Send      chan []byte
	Username  string
	CloseOnce sync.Once
}

type HubInterface interface {
	GetAddress() string
	UnregisterClient(*Client)
	SaveMessage(models.Message) error
	PublishMessage([]byte) error
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.UnregisterClient(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("[Server %s] Client '%s' unexpected close error: %v", c.Hub.GetAddress(), c.Username, err)
			} else {
				log.Printf("[Server %s] Client '%s' disconnected normally", c.Hub.GetAddress(), c.Username)
			}
			break
		}

		var incomingMsg models.Message
		if err := json.Unmarshal(message, &incomingMsg); err != nil {
			log.Printf("Error parsing incoming message JSON: %v", err)
			continue
		}

		msg := models.Message{
			Username: c.Username,
			Content:  incomingMsg.Content,
			Server:   c.Hub.GetAddress(),
		}

		if err := c.Hub.SaveMessage(msg); err != nil {
			log.Printf("Error saving message: %v", err)
			continue
		}

		msgBytes, _ := json.Marshal(msg)
		if err := c.Hub.PublishMessage(msgBytes); err != nil {
			log.Printf("Error publishing to Redis: %v", err)
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[Server %s] Client '%s' write error: %v", c.Hub.GetAddress(), c.Username, err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[Server %s] Client '%s' ping error: %v", c.Hub.GetAddress(), c.Username, err)
				return
			}
		}
	}
}