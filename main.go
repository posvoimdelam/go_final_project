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
	"go_final_project/service"
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

	service := service.NewTaskService(db)
	handler := handlers.NewTaskHandler(service)

	http.HandleFunc("/api/task", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handler.GetTaskByIdHandler(w, r)
		case http.MethodPost:
			handler.AddTaskHandler(w, r)
		case http.MethodPut:
			handler.UpdateTaskHandler(w, r)
		case http.MethodDelete:
			handler.DeleteTaskHandler(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/api/nextdate", handlers.ApiNextDateHandler)
	http.HandleFunc("/api/task/done", func(w http.ResponseWriter, r *http.Request) { handler.DoneTaskHandler(w, r) })
	http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) { handler.GetTasksHandler(w, r) })

	log.Printf("Starting server on port %s, directory: %s\n", port, webDir)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}

}
