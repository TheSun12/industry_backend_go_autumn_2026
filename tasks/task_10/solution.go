package main

import (
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
	"strings"
	"strconv"
	"sort"
	"encoding/json"
)

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	UpdatedAt time.Time `json:"updatedAt"`
}
type TaskRepo interface {
	Create(title string) (Task, error)
	Get(id string) (Task, bool)
	List(done bool) []Task
	SetDone(id string, done bool) (Task, error)
}
type Clock interface{ Now() time.Time }

var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("invalid title")
)

type inMemoryTaskRepo struct {
	mu    sync.RWMutex
	clock Clock
	seq   uint64
	tasks map[string]Task
}

func NewInMemoryTaskRepo(clock Clock) TaskRepo {
	return &inMemoryTaskRepo{clock: clock, tasks: make(map[string]Task)}
}
func (r *inMemoryTaskRepo) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	id := strconv.FormatUint(r.seq, 10)
	task := Task{ID: id, Title: title, Done: false, UpdatedAt: r.clock.Now()}
	r.tasks[id] = task

	return task, nil
}
func (r *inMemoryTaskRepo) Get(id string) (Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, status := r.tasks[id]
	return task, status
}
func (r *inMemoryTaskRepo) List(done bool) []Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Task, 0)

	for _, task := range r.tasks {
		if task.Done == done {
			list = append(list, task)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].UpdatedAt.Equal(list[j].UpdatedAt) {
			return list[i].ID < list[j].ID
		}
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list
}
func (r *inMemoryTaskRepo) SetDone(id string, done bool) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, status := r.tasks[id]
	if !status {
		return Task{}, ErrNotFound
	}

	if task.Done == done {
		return task, nil
	}

	task.Done = done
	task.UpdatedAt = r.clock.Now()
	r.tasks[id] = task

	return task, nil
}


type httpHandler struct{ repo TaskRepo }

func NewHTTPHandler(repo TaskRepo) http.Handler {
	return &httpHandler{repo: repo}
}
func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path

	if !strings.HasPrefix(url, "/tasks") {
		http.NotFound(w, r)
		return
	}

	rest := strings.TrimPrefix(url, "/tasks")

	if rest == "" {
		switch r.Method {
		case http.MethodPost:
			h.handleCreate(w, r)
		case http.MethodGet:
			h.handleList(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if !strings.HasPrefix(rest, "/") {
		http.NotFound(w, r)
		return
	}

	id := strings.TrimPrefix(rest, "/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPatch:
		h.handlePatch(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
func (h *httpHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title *string `json:"title"`
	}

	err := decodeStrictJSON(r.Body, &body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Title == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, status := h.repo.Create(*body.Title)
	if status != nil {
		if errors.Is(status, ErrInvalidTitle) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}
func (h *httpHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	task, status := h.repo.Get(id)
	if !status {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
func (h *httpHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	doneParams, ok := query["done"]
	done := false

	if ok {
		if len(doneParams) != 1 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		
		switch doneParams[0] {
		case "true":
			done = true
		case "false":
			done = false
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	tasks := h.repo.List(done)
	writeJSON(w, http.StatusOK, tasks)
}
func (h *httpHandler) handlePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Done *bool `json:"done"`
	}
	
	err1 := decodeStrictJSON(r.Body, &body)
	if err1 != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	
	if body.Done == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	task, err2 := h.repo.SetDone(id, *body.Done)
	if err2 != nil {
		if errors.Is(err2, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, task)
}
func decodeStrictJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	
	err1 := dec.Decode(v)
	if err1 != nil {
		return err1
	}

	err2 := dec.Decode(&struct{}{})
	if err2 != io.EOF {
		return errors.New("trailing data after JSON object")
	}
	
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
