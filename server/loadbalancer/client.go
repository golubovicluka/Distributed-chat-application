package loadbalancer

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

const lbURL = "http://127.0.0.1:9000"

type Client struct {
	address string
}

func New(address string) *Client {
	return &Client{
		address: address,
	}
}

func (c *Client) Register() {
	payload := map[string]interface{}{
		"address": c.address,
		"load":    0,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/register", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Fatalf("[Server %s] Failed to register with LB: %v", c.address, err)
	}
	resp.Body.Close()
	log.Printf("[Server %s] Successfully registered with Load Balancer\n", c.address)
}

func (c *Client) UpdateLoad(load int) {
	payload := map[string]interface{}{
		"address": c.address,
		"load":    load,
	}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(lbURL+"/update", "application/json", bytes.NewReader(b))
	if err != nil {
		log.Printf("[Server %s] Failed to update load: %v\n", c.address, err)
		return
	}
	resp.Body.Close()
}