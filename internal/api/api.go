package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/agidelle/merch-shop/internal/domain"
	"github.com/agidelle/merch-shop/internal/service"
	"log"
	"net/http"
	"strconv"
	"time"
)

var errorMap = map[error]int{
	domain.ErrID:             http.StatusBadRequest,
	domain.ErrBadTitle:       http.StatusBadRequest,
	domain.ErrDate:           http.StatusBadRequest,
	domain.ErrInternalServer: http.StatusInternalServerError,
}

const dateForm string = "20060102"

type TaskHandler struct {
	service TaskService
}

type TaskService interface {
	FindAll(ctx context.Context, filter *domain.Filter) ([]*domain.Task, *domain.CustomError)
	Create(ctx context.Context, task *domain.Task) (int64, *domain.CustomError)
	Update(ctx context.Context, task *domain.Task) *domain.CustomError
	Done(ctx context.Context, filter *domain.Filter) *domain.CustomError
	Delete(ctx context.Context, id int) *domain.CustomError
	NextDate(now time.Time, dstart string, repeat string) (string, error)
	CloseDB()
}

func NewHandler(service *service.TaskService) *TaskHandler {
	return &TaskHandler{service: service}
}

func sendJSONError(w http.ResponseWriter, customErr *domain.CustomError) {
	w.WriteHeader(customErr.Code)
	var errorMessage string
	if customErr.ErrStorage != nil {
		errorMessage = fmt.Sprintf("%v: %v", customErr.Err, customErr.ErrStorage)
	} else {
		errorMessage = fmt.Sprintf("%v", customErr.Err)
	}

	err := json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{
		Error: errorMessage,
	})
	if err != nil {
		log.Println(err)
	}
}

func sendJSONTasks(w http.ResponseWriter, tasks []*domain.Task) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(struct {
		Tasks []*domain.Task `json:"tasks"`
	}{
		Tasks: tasks,
	})
	if err != nil {
		log.Println(err)
	}
}

func (h *TaskHandler) AddTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	w.Header().Add("Content-Type", "application/json")
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, errors.New("ошибка десериализации JSON"), nil))
		return
	}

	//Добавление задачи
	id, cErr := h.service.Create(ctx, &task)
	if cErr != nil {
		if code, ok := errorMap[cErr.Err]; ok {
			cErr.Code = code
		} else {
			cErr.Code = http.StatusInternalServerError
		}
		sendJSONError(w, cErr)
		return
	}

	task.ID = strconv.Itoa(int(id))

	w.WriteHeader(http.StatusCreated)
	err := json.NewEncoder(w).Encode(map[string]int64{"id": id})
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var filter domain.Filter
	queryValues := r.URL.Query()
	_, searchParamExists := queryValues["search"]

	filter.SearchTerm = r.URL.Query().Get("search")

	if !searchParamExists {
		res, cErr := h.service.FindAll(ctx, &filter)
		if cErr != nil {
			if code, ok := errorMap[cErr.Err]; ok {
				cErr.Code = code
			} else {
				cErr.Code = http.StatusInternalServerError
			}
			sendJSONError(w, cErr)
			return
		}
		sendJSONTasks(w, res)
	} else {
		res, cErr := h.service.FindAll(ctx, &filter)
		if cErr != nil {
			if code, ok := errorMap[cErr.Err]; ok {
				cErr.Code = code
			} else {
				cErr.Code = http.StatusInternalServerError
			}
			sendJSONError(w, cErr)
			return
		}
		sendJSONTasks(w, res)
	}
}

func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var filter domain.Filter
	searchID := r.URL.Query().Get("id")
	if searchID == "" {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, nil))
		return
	}
	id, err := strconv.Atoi(searchID)
	if err != nil {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, err))
		return
	}
	filter.ID = &id
	task, cErr := h.service.FindAll(ctx, &filter)
	if cErr != nil {
		if code, ok := errorMap[cErr.Err]; ok {
			cErr.Code = code
		} else {
			cErr.Code = http.StatusInternalServerError
		}
		sendJSONError(w, cErr)
		return
	}

	task[0].ID = searchID
	err = json.NewEncoder(w).Encode(&task)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	w.Header().Add("Content-Type", "application/json")
	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, errors.New("ошибка десериализации JSON"), err))
		return
	}
	cErr := h.service.Update(ctx, &task)
	if cErr != nil {
		if code, ok := errorMap[cErr.Err]; ok {
			cErr.Code = code
		} else {
			cErr.Code = http.StatusInternalServerError
		}
		sendJSONError(w, cErr)
		return
	}
	err := json.NewEncoder(w).Encode(domain.Task{})
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *TaskHandler) Done(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var filter domain.Filter
	searchID := r.URL.Query().Get("id")
	if searchID == "" {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, nil))
		return
	}
	id, err := strconv.Atoi(searchID)
	if err != nil {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, err))
		return
	}
	filter.ID = &id
	cErr := h.service.Done(ctx, &filter)
	if cErr != nil {
		if code, ok := errorMap[cErr.Err]; ok {
			cErr.Code = code
		} else {
			cErr.Code = http.StatusInternalServerError
		}
		sendJSONError(w, cErr)
		return
	}

	err = json.NewEncoder(w).Encode(domain.Task{})
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	searchID := r.URL.Query().Get("id")
	if searchID == "" {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, nil))
		return
	}
	id, err := strconv.Atoi(searchID)
	if err != nil {
		sendJSONError(w, domain.NewCustomError(http.StatusBadRequest, domain.ErrID, err))
		return
	}
	cErr := h.service.Delete(ctx, id)
	if cErr != nil {
		if code, ok := errorMap[cErr.Err]; ok {
			cErr.Code = code
		} else {
			cErr.Code = http.StatusInternalServerError
		}
		sendJSONError(w, cErr)
		return
	}

	err = json.NewEncoder(w).Encode(domain.Task{})
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func (h *TaskHandler) NextDateHandler(w http.ResponseWriter, r *http.Request) {
	nowStr := r.URL.Query().Get("now")
	dateStr := r.URL.Query().Get("date")
	repeat := r.URL.Query().Get("repeat")

	var now time.Time
	if nowStr == "" {
		now = time.Now()
	} else {
		var err error
		now, err = time.Parse(dateForm, nowStr)
		if err != nil {
			http.Error(w, domain.ErrDate.Error(), http.StatusBadRequest)
			return
		}
	}
	if dateStr == "" {
		http.Error(w, domain.ErrDate.Error(), http.StatusBadRequest)
		return
	}
	_, err := time.Parse(dateForm, dateStr)
	if err != nil {
		http.Error(w, domain.ErrDate.Error(), http.StatusBadRequest)
		return
	}
	nextDate, err := h.service.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, fmt.Sprintf("ошибка вычисления следующей даты: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if nextDate == "" {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.Write([]byte(nextDate))
	}
}
