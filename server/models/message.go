package models

type Message struct {
	ID        int64  `json:"id,omitempty"`
	Username  string `json:"username"`
	Content   string `json:"content"`
	Server    string `json:"server,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}