package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"go_final_project/models"
	"net/http"
	"time"
)

const limit = 50

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) DeleteById(id string) (sql.Result, int, error) {
	res, err := r.db.Exec("DELETE FROM scheduler WHERE id=?", id)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("error executing delete: %v", err)
	}
	return res, 0, nil
}

func (r *TaskRepository) GetTaskById(id string) (*models.Task, int, error) {
	var task models.Task
	row := r.db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?", id)
	err := row.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusNotFound, fmt.Errorf("no such row: %v", err)
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("error scanning row: %v", err)
	}
	return &task, 0, nil
}

func (r *TaskRepository) UpdateById(id string, updates map[string]any) (sql.Result, int, error) {
	if len(updates) == 0 {
		return nil, http.StatusInternalServerError, fmt.Errorf("no updates in sql query")
	}

	query := "UPDATE scheduler SET "
	args := make([]any, 0, len(updates)+1)
	for field, value := range updates {
		query += fmt.Sprintf("%s = ?, ", field)
		args = append(args, value)
	}
	query = query[:len(query)-2]

	query += " WHERE id = ?"
	args = append(args, id)

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error updating table: %v", err)
	}
	return res, 0, nil
}

func (r *TaskRepository) GetTasks(search string) ([]models.Task, int, error) {
	var query string
	var args []interface{}

	if search == "" {
		query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?"
		args = append(args, limit)
	} else if isValidDate(search) {
		date, err := time.Parse("02.01.2006", search)
		if err != nil {
			return []models.Task{}, http.StatusBadRequest, fmt.Errorf("invalid date format in search: %v", err)
		}
		formatedDate := date.Format(models.Layout)
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? LIMIT ?"
		args = append(args, formatedDate, limit)
	} else {
		like := "%" + search + "%"
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?"
		args = append(args, like, like, limit)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return []models.Task{}, http.StatusInternalServerError, fmt.Errorf("can't make query: %v", err)
	}
	defer rows.Close()

	var task models.Task
	var tasks []models.Task
	for rows.Next() {
		err := rows.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return []models.Task{}, http.StatusInternalServerError, fmt.Errorf("can't get rows: %v", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return []models.Task{}, http.StatusInternalServerError, fmt.Errorf("loop terminated with error: %v", err)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}
	return tasks, 0, nil
}

func (r *TaskRepository) AddTaskAndGetId(taskDate time.Time, task models.Task) (int64, int, error) {
	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"
	res, err := r.db.Exec(query, taskDate.Format(models.Layout), task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, http.StatusInternalServerError, fmt.Errorf("database error: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, http.StatusInternalServerError, fmt.Errorf("error getting id: %v", err)
	}
	return id, 0, nil
}

func isValidDate(date string) bool {
	_, err := time.Parse("02.01.2006", date)
	return err == nil
}
