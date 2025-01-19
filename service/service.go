package service

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go_final_project/dates"
	"go_final_project/models"
	"go_final_project/repo"
)

type TaskService struct {
	repo *repo.TaskRepository
}

func NewTaskService(repo *repo.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) DeleteTask(id string) (int, error) {
	res, statusCode, err := s.repo.DeleteById(id)
	if err != nil {
		return statusCode, err
	}

	if statusCode, err := checkRowsAffected(res); err != nil {
		return statusCode, err
	}
	return 0, nil
}

func (s *TaskService) DoneTask(id string) (int, error) {

	task, statusCode, err := s.repo.GetTaskById(id)
	if err != nil || task == nil {
		return statusCode, err
	}

	//если поле repeat пустое то запись удаляется, если нет то рассчитывается следующая дата и в таблице обновляется date
	if task.Repeat == "" {
		res, statusCode, err := s.repo.DeleteById(id)
		if err != nil {
			return statusCode, err
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

		updates := map[string]any{
			"date": nextDate,
		}

		res, statusCode, err := s.repo.UpdateById(id, updates)
		if err != nil {
			return statusCode, err
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

	if task.Id == "" {
		return http.StatusBadRequest, fmt.Errorf("empty id")
	} else {
		_, err = strconv.Atoi(task.Id)
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

	updates := map[string]any{
		"date":    taskDate.Format(models.Layout),
		"title":   task.Title,
		"comment": task.Comment,
		"repeat":  task.Repeat,
	}

	result, statusCode, err := s.repo.UpdateById(task.Id, updates)
	if err != nil {
		return statusCode, err
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

func (s *TaskService) GetTaskById(id string) (*models.Task, int, error) {
	task, statusCode, err := s.repo.GetTaskById(id)
	if err != nil || task == nil {
		return &models.Task{}, statusCode, err
	}
	return task, 0, nil
}

func (s *TaskService) GetTasks(search string) (map[string][]models.Task, int, error) {

	tasks, statusCode, err := s.repo.GetTasks(search)
	if err != nil {
		return map[string][]models.Task{}, statusCode, err
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

	id, statusCode, err := s.repo.AddTaskAndGetId(taskDate, task)
	if err != nil {
		return 0, statusCode, err
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
