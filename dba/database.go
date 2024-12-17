package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const defaultDBFile = "scheduler.db"

func DatabasePath() string {
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

func DbInit(dbPath string) *sql.DB {

	_, err := os.Stat(dbPath)
	install := os.IsNotExist(err)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
	}

	if install {
		log.Println("Database doesn't exist, creating ...")
		createTable(db)
	} else {
		if err = ifTableExists(db); err != nil {
			log.Fatalf("Database initialized but table 'scheduler doesn't exist':%v", err)
		}
	}
	return db
}

func createTable(db *sql.DB) { //создает таблицу scheduler
	query := `
	CREATE TABLE scheduler (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	date TEXT NOT NULL,
	title TEXT NOT NULL,
	comment TEXT NOT NULL,
	repeat TEXT
	);
	CREATE INDEX index_date ON scheduler(date);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	log.Print("Table and index are created")
}

func ifTableExists(db *sql.DB) error { //проверяет существует ли таблица scheduler в базе данных
	query := `
	SELECT name 
	FROM sqlite_master
	WHERE type='table'
	AND name='scheduler' 
	`
	var name string
	err := db.QueryRow(query).Scan(&name)
	if err != nil {
		return fmt.Errorf("can't scan: %v", err)
	}
	if name != "scheduler" {
		return fmt.Errorf("table 'scheduler' doesn't exist:%v", err)
	}
	return nil
}
