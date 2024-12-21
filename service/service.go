package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go_final_project/dates"
	"go_final_project/models"
)

const limit = 50

type TaskService struct {
	db *sql.DB
}

func NewTaskService(db *sql.DB) *TaskService {
	return &TaskService{db: db}
}

func (s *TaskService) DeleteTask(id string) (int, error) {
	res, err := s.db.Exec("DELETE FROM scheduler WHERE id=?", id)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error executing delete: %v", err)
	}
	if statusCode, err := checkRowsAffected(res); err != nil {
		return statusCode, err
	}
	return 0, nil
}

func (s *TaskService) DoneTask(id string) (int, error) {
	//делается запрос в базу данных по id
	row := s.db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?", id)
	//запись данных строки в структуру
	var task models.Task
	err := row.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return http.StatusNotFound, fmt.Errorf("no such row: %v", err)
		} else {
			return http.StatusInternalServerError, fmt.Errorf("error scanning row: %v", err)
		}
	}
	//если поле repeat пустое то запись удаляется, если нет то рассчитывается следующая дата и в таблице обновляется date
	if task.Repeat == "" {
		res, err := s.db.Exec("DELETE FROM scheduler WHERE id=?", id)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error executing delete: %v", err)
		}
		if statusCode, err := checkRowsAffected(res); err != nil {
			return statusCode, err
		}
	} else {
		now := time.Now()
		if task.Date == "" {
			return http.StatusBadRequest, fmt.Errorf("empty date")
		}
		nextDate, err := dates.NextDate(now, task.Date, task.Repeat)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("can't find next date: %v", err)
		}
		res, err := s.db.Exec("UPDATE scheduler SET date=?", nextDate)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error executing update: %v", err)
		}
		if statusCode, err := checkRowsAffected(res); err != nil {
			return statusCode, err
		}
	}
	return 0, nil
}

func (s *TaskService) UpdateTask(task models.Task) (int, error) {

	var err error

	if task.Title == "" {
		return http.StatusBadRequest, fmt.Errorf("empty title")
	}
	var id int
	if task.Id == "" {
		return http.StatusBadRequest, fmt.Errorf("empty id")
	} else {
		id, err = strconv.Atoi(task.Id)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("invalid id format")
		}
	}

	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)

	var taskDate time.Time

	if task.Date == "" {
		taskDate = today
	} else {
		taskDate, err = time.Parse(models.Layout, task.Date)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("invalid date format")
		}
	}

	if task.Repeat == "" {
		taskDate = today
	} else {
		_, err := dates.NextDate(today, taskDate.Format(models.Layout), task.Repeat)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("can't find next date: %v", err)
		}
	}

	result, err := s.db.Exec("UPDATE scheduler SET date=?, title=?, comment=?, repeat=? WHERE id=?", taskDate.Format(models.Layout), task.Title, task.Comment, task.Repeat, id)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error updating table: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error checking rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return http.StatusNotFound, fmt.Errorf("no such task %v", sql.ErrNoRows)
	}
	return 0, nil
}

func (s *TaskService) GetTaskById(id string) (models.Task, int, error) {
	var task models.Task
	err := s.db.QueryRow("SELECT id, date, title, comment, repeat FROM scheduler WHERE id=?", id).Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Task{}, http.StatusNotFound, fmt.Errorf("no such row %v", sql.ErrNoRows)
		} else {
			return models.Task{}, http.StatusNotFound, fmt.Errorf("can't get task: %v", err)
		}
	}
	return task, 0, nil
}

func (s *TaskService) GetTasks(search string) (map[string][]models.Task, int, error) {

	var query string
	var args []interface{}

	if search == "" {
		query = "SELECT id, date, title, comment, repeat FROM scheduler ORDER BY date LIMIT ?"
		args = append(args, limit)
	} else if isValidDate(search) {
		date, err := time.Parse("02.01.2006", search)
		if err != nil {
			return map[string][]models.Task{}, http.StatusBadRequest, fmt.Errorf("invalid date format in search: %v", err)
		}
		formatedDate := date.Format(models.Layout)
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE date = ? LIMIT ?"
		args = append(args, formatedDate, limit)
	} else {
		like := "%" + search + "%"
		query = "SELECT id, date, title, comment, repeat FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?"
		args = append(args, like, like, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return map[string][]models.Task{}, http.StatusInternalServerError, fmt.Errorf("can't make query: %v", err)
	}
	defer rows.Close()

	var task models.Task
	var tasks []models.Task
	for rows.Next() {
		err := rows.Scan(&task.Id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return map[string][]models.Task{}, http.StatusInternalServerError, fmt.Errorf("can't get rows: %v", err)
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return map[string][]models.Task{}, http.StatusInternalServerError, fmt.Errorf("loop terminated with error: %v", err)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	return map[string][]models.Task{"tasks": tasks}, 0, nil
}

func (s *TaskService) AddTask(task models.Task) (int64, int, error) {

	var err error
	if task.Title == "" {
		return 0, http.StatusBadRequest, fmt.Errorf("empty title")
	}

	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)

	var taskDate time.Time

	if task.Date == "" {
		taskDate = today
	} else {
		taskDate, err = time.Parse(models.Layout, task.Date)
		if err != nil {
			return 0, http.StatusBadRequest, fmt.Errorf("invalid date format")
		}
	}

	if task.Repeat != "" {
		_, err = dates.NextDate(today, taskDate.Format(models.Layout), task.Repeat)
		if err != nil {
			return 0, http.StatusBadRequest, fmt.Errorf("can't find next date: %v", err)
		}
	}

	if taskDate.Before(today) {
		if task.Repeat == "" {
			taskDate = today
		} else {
			nextDate, err := dates.NextDate(today, taskDate.Format(models.Layout), task.Repeat)
			if err != nil {
				return 0, http.StatusBadRequest, fmt.Errorf("can't find next date: %v", err)
			}
			taskDate, err = time.Parse(models.Layout, nextDate)
			if err != nil {
				return 0, http.StatusBadRequest, fmt.Errorf("invalid date format")
			}
		}
	}

	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"
	res, err := s.db.Exec(query, taskDate.Format(models.Layout), task.Title, task.Comment, task.Repeat)
	if err != nil {
		return 0, http.StatusInternalServerError, fmt.Errorf("database error: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, http.StatusInternalServerError, fmt.Errorf("error getting id: %v", err)
	}

	return id, 0, nil
}

func ApiNextDate(now string, date string, repeat string) (string, int, string) {

	t, err := time.Parse(models.Layout, now)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Sprintf("Invalid 'now' parameter: %v", err)
	}
	nextDate, err := dates.NextDate(t, date, repeat)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Sprintf("Error calculating next date: %v", err)
	}
	return nextDate, 0, ""
}

func isValidDate(date string) bool {
	_, err := time.Parse("02.01.2006", date)
	return err == nil
}

func checkRowsAffected(res sql.Result) (int, error) {
	rows, err := res.RowsAffected()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error checking rows affected: %v", err)
	}
	if rows == 0 {
		return http.StatusNotFound, fmt.Errorf("no such task %v", sql.ErrNoRows)
	}
	return 0, nil
}
