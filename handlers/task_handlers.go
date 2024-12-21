package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"go_final_project/models"
	"go_final_project/service"
)

type TaskHandler struct {
	service *service.TaskService
}

func NewTaskHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func (h *TaskHandler) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	if id == "" {
		writeErrorResponse(w, "missing id", http.StatusInternalServerError)
		return
	}

	statusCode, err := h.service.DeleteTask(id)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}

	writeResponse(w, map[string]string{})
}

func (h *TaskHandler) DoneTaskHandler(w http.ResponseWriter, r *http.Request) {
	//проверка метода
	if r.Method != http.MethodPost {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}
	//получаем и проверяем id
	id := r.URL.Query().Get("id")
	if id == "" {
		writeErrorResponse(w, "missing id", http.StatusBadRequest)
		return
	}

	statusCode, err := h.service.DoneTask(id)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}
	//возвращается пустой json либо ошибка
	writeResponse(w, map[string]string{})
}

func (h *TaskHandler) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("deserialization error: %v", err), http.StatusBadRequest)
		return
	}

	statusCode, err := h.service.UpdateTask(task)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}

	writeResponse(w, map[string]string{})
}

func (h *TaskHandler) GetTaskByIdHandler(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	if id == "" {
		writeErrorResponse(w, "empty id", http.StatusBadRequest)
		return
	}

	task, statusCode, err := h.service.GetTaskById(id)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}

	writeResponse(w, task)
}

func (h *TaskHandler) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, "wrong method", http.StatusMethodNotAllowed)
		return
	}

	search := r.URL.Query().Get("search")

	resp, statusCode, err := h.service.GetTasks(search)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}

	writeResponse(w, resp)
}

func (h *TaskHandler) AddTaskHandler(w http.ResponseWriter, r *http.Request) {

	var task models.Task
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("deserialization error JSON:%v", err), http.StatusBadRequest)
		return
	}

	id, statusCode, err := h.service.AddTask(task)
	if err != nil {
		writeErrorResponse(w, err.Error(), statusCode)
		return
	}

	resp := struct {
		Id int64 `json:"id"`
	}{Id: id}

	writeResponse(w, resp)
}

func ApiNextDateHandler(w http.ResponseWriter, r *http.Request) {
	now := r.FormValue("now")
	date := r.FormValue("date")
	repeat := r.FormValue("repeat")

	if now == "" || date == "" || repeat == "" {
		http.Error(w, "Missing required query parameters", http.StatusBadRequest)
		return
	}

	nextDate, statusCode, errStr := service.ApiNextDate(now, date, repeat)
	if errStr != "" {
		http.Error(w, errStr, statusCode)
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

func writeResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("serialization error: %v", err), http.StatusInternalServerError)
		return
	}
}
