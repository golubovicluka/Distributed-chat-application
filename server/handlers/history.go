package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"lukagolubovic/models"
)

func GetHistory(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, username, message, server, timestamp FROM messages ORDER BY timestamp DESC LIMIT 50")
		if err != nil {
			http.Error(w, "Failed to retrieve message history", http.StatusInternalServerError)
			log.Printf("DB query error: %v", err)
			return
		}
		defer rows.Close()

		var messages []models.Message
		for rows.Next() {
			var msg models.Message
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
}