package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	usecase taskusecase.Usecase
}

func NewTaskHandler(usecase taskusecase.Usecase) *TaskHandler {
	return &TaskHandler{usecase: usecase}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), taskusecase.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, taskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.usecase.List(r.Context())
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	response := make([]taskDTO, 0, len(tasks))
	for i := range tasks {
		response = append(response, newTaskDTO(&tasks[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func getIDFromRequest(r *http.Request) (int64, error) {
	return parsePositiveInt64(mux.Vars(r)["id"], "task id")
}

// parsePositiveInt64 parses rawVal as a positive int64 and returns a
// descriptive error using fieldName when validation fails.
func parsePositiveInt64(rawVal, fieldName string) (int64, error) {
	if rawVal == "" {
		return 0, fmt.Errorf("missing %s", fieldName)
	}

	id, err := strconv.ParseInt(rawVal, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", fieldName)
	}

	if id <= 0 {
		return 0, fmt.Errorf("invalid %s: must be positive", fieldName)
	}

	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
