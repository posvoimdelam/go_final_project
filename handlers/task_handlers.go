package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"go_final_project/dates"
	"go_final_project/models"
)

const limit = 50

func DeleteTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodDelete {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")

	res, err := db.Exec("DELETE FROM scheduler WHERE id=?", id)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("error executing delete: %v", err), http.StatusInternalServerError)
		return
	}
	rows, err := res.RowsAffected()
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("error checking rows affected: %v", err), http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		writeErrorResponse(w, "no rows affected", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(map[string]string{})
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func DoneTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	//проверка метода
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	//получаем id
	id := r.URL.Query().Get("id")
	//делается запрос в базу данных по id
	row := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?", id)
	//запись данных строки в структуру
	var task models.Task
	err := row.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeErrorResponse(w, fmt.Sprintf("no such row: %v", err), http.StatusNotFound)
			return
		} else {
			writeErrorResponse(w, fmt.Sprintf("error scanning row: %v", err), http.StatusInternalServerError)
			return
		}
	}
	//если поле repeat пустое то запись удаляется, если нет рассчитывается следующая дата и в таблице обновляется date
	if task.Repeat == "" {
		res, err := db.Exec("DELETE FROM scheduler WHERE id=?", id)
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("error executing delete: %v", err), http.StatusInternalServerError)
			return
		}
		rows, err := res.RowsAffected()
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("error checking rows affected: %v", err), http.StatusInternalServerError)
			return
		}
		if rows == 0 {
			writeErrorResponse(w, "no rows affected", http.StatusNotFound)
			return
		}
	} else {

		now := time.Now()
		if task.Date == "" {
			writeErrorResponse(w, "empty date", http.StatusBadRequest)
			return
		}

		nextDate, err := dates.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("can't find next date: %v", err), http.StatusBadRequest)
			return
		}
		res, err := db.Exec("UPDATE scheduler SET date=?", nextDate)
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("error updating error: %v", err), http.StatusInternalServerError)
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("error checking rows affected: %v", err), http.StatusInternalServerError)
			return
		}
		if rowsAffected == 0 {
			writeErrorResponse(w, "no such row", http.StatusNotFound)
			return
		}
	}
	//возвращается пустой json либо ошибка
	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(map[string]string{})
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func UpdateTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPut {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("deserialization error: %v", err), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		writeErrorResponse(w, "empty title", http.StatusBadRequest)
		return
	}
	var id int
	if task.Id == "" {
		writeErrorResponse(w, "empty id", http.StatusBadRequest)
		return
	} else {
		id, err = strconv.Atoi(task.Id)
		if err != nil {
			writeErrorResponse(w, "invalid id format", http.StatusBadRequest)
			return
		}
	}

	today := time.Now()

	var taskDate time.Time

	if task.Date == "" {
		taskDate = today
	} else {
		taskDate, err = time.Parse(models.Layout, task.Date)
		if err != nil {
			writeErrorResponse(w, "invalid date format", http.StatusBadRequest)
			return
		}
	}

	if taskDate.Before(today) {
		if task.Repeat == "" {
			taskDate = today
		} else {
			nextDate, err := dates.NextDate(today, taskDate.Format(models.Layout), task.Repeat)
			if err != nil {
				writeErrorResponse(w, fmt.Sprintf("can't find next date: %v", err), http.StatusBadRequest)
				return
			}
			taskDate, err = time.Parse(models.Layout, nextDate)
			if err != nil {
				writeErrorResponse(w, "invalid date format", http.StatusBadRequest)
				return
			}
		}
	}

	result, err := db.Exec("UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?", taskDate.Format(models.Layout), task.Title, task.Comment, task.Repeat, id)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("error updating table: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("error checking rows affected: %v", err), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		writeErrorResponse(w, "no such row", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(map[string]string{})
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func GetTaskByIdHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeErrorResponse(w, "empty id", http.StatusBadRequest)
		return
	}

	var task models.Task
	err := db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?", id).Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeErrorResponse(w, "no such row", http.StatusNotFound)
			return
		} else {
			writeErrorResponse(w, fmt.Sprintf("can't get row: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(task)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func GetTasksHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	search := r.URL.Query().Get("search")

	var query string
	var args []interface{}

	if search == "" {
		query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?"
		args = append(args, limit)
	} else if isValidDate(search) {
		date, err := time.Parse("02.01.2006", search)
		if err != nil {
			writeErrorResponse(w, "invalid date format in search", http.StatusBadRequest)
			return
		}
		formatedDate := date.Format(models.Layout)
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? LIMIT ?"
		args = append(args, formatedDate, limit)
	} else {
		like := "%" + search + "%"
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?"
		args = append(args, like, like, limit)
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("can't make query: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var task models.Task
	var tasks []models.Task
	for rows.Next() {
		err := rows.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("can't get rows: %v", err), http.StatusInternalServerError)
			return
		}

		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		writeErrorResponse(w, fmt.Sprintf("loop terminated with error: %v", err), http.StatusInternalServerError)
		return
	}
	if err = rows.Close(); err != nil {
		log.Println(err)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	resp := map[string][]models.Task{"tasks": tasks}
	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func AddTaskHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost { //проверка метода
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("deserialization error JSON:%v", err), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		writeErrorResponse(w, "empty title", http.StatusBadRequest)
		return
	}

	today := time.Now()

	var taskDate time.Time

	if task.Date == "" {
		taskDate = today
	} else {
		taskDate, err = time.Parse(models.Layout, task.Date)
		if err != nil {
			writeErrorResponse(w, "invalid date format", http.StatusBadRequest)
			return
		}
	}

	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	if taskDate.Before(today) {
		if task.Repeat == "" {
			taskDate = today
		} else {
			nextDate, err := dates.NextDate(today, taskDate.Format(models.Layout), task.Repeat)
			if err != nil {
				writeErrorResponse(w, fmt.Sprintf("can't find next date: %v", err), http.StatusBadRequest)
				return
			}
			taskDate, err = time.Parse(models.Layout, nextDate)
			if err != nil {
				writeErrorResponse(w, "invalid date format", http.StatusBadRequest)
				return
			}
		}
	}
	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"
	res, err := db.Exec(query, taskDate.Format(models.Layout), task.Title, task.Comment, task.Repeat)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("error getting id: %v", err), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Id int64 `json:"id"`
	}{Id: id}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}

func ApiNextDateHandler(w http.ResponseWriter, r *http.Request) {
	now := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	if now == "" || date == "" || repeat == "" {
		http.Error(w, "Missing required query parameters", http.StatusBadRequest)
		return
	}

	t, err := time.Parse(models.Layout, now)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid 'now' parameter: %v", err), http.StatusBadRequest)
		return
	}
	nextDate, err := dates.NextDate(t, date, repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calculating next date: %v", err), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(nextDate))
}

func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func isValidDate(date string) bool {
	_, err := time.Parse("02.01.2006", date)
	return err == nil
}
