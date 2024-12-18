package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	dba "go_final_project/dba"
	"go_final_project/handlers"
)

const webDir = "./web"
const defaultPort = "7540"

var db *sql.DB

func main() {

	if err := godotenv.Load(); err != nil { //загружаем переменные окружения
		log.Println("No .env file found")
	}

	dbPath := dba.DatabasePath() //путь до файла базы данных scheduler.db
	db = dba.DbInit(dbPath)      // инициализация и подключение к базе данных scheduler.db
	defer db.Close()

	port := os.Getenv("TODO_PORT") // получаем значение переменной окружения TODO_PORT
	if port == "" {
		port = defaultPort
	}

	http.Handle("/", http.FileServer(http.Dir(webDir))) //регистрация маршрута "/", создание файл-сервера для директории webDir

	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetTaskByIdHandler(w, r, db)
		case http.MethodPost:
			handlers.AddTaskHandler(w, r, db)
		case http.MethodPut:
			handlers.UpdateTaskHandler(w, r, db)
		case http.MethodDelete:
			handlers.DeleteTaskHandler(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/nextdate", handlers.ApiNextDateHandler)
	http.HandleFunc("/api/task/done", func(w http.ResponseWriter, r *http.Request) { handlers.DoneTaskHandler(w, r, db) })
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) { handlers.GetTasksHandler(w, r, db) })

	log.Printf("Starting server on port %s, directory: %s\n", port, webDir)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}

}
