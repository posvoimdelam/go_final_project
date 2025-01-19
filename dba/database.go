package db

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
)

const defaultDBFile = "scheduler.db"

func DatabasePath() string { //возвращает путь до файла базы данных
	dbPath := os.Getenv("TODO_DBFILE")
	if dbPath == "" {
		appPath, err := os.Executable()
		if err != nil {
			log.Fatalf("Can't determine app path: %v", err)
		}
		dbPath = filepath.Join(filepath.Dir(appPath), defaultDBFile)
	}

	return dbPath
}

func DbInit(dbPath string) *sql.DB { //инициализирует и подключается к базе данных

	_, err := os.Stat(dbPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Println("Database doesn't exist, creating ...")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
	}

	createTable(db)

	return db
}

func createTable(db *sql.DB) { //создает таблицу scheduler
	query := `
	CREATE TABLE IF NOT EXISTS scheduler (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date VARCHAR(8) NOT NULL,
	title TEXT NOT NULL,
	comment TEXT NOT NULL,
	repeat VARCHAR(128)
	);
	CREATE INDEX IF NOT EXISTS index_date ON scheduler(date);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
}
