package database

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	createTableSQL := `CREATE TABLE IF NOT EXISTS messages (
		"id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT,
		"message" TEXT,
		"server" TEXT,
		"timestamp" DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Printf("Failed to create table: %v", err)
		return err
	}

	return nil
}