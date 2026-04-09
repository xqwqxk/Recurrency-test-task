package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
	recurrenceusecase "example.com/taskservice/internal/usecase/recurrence"
)

// RecurrenceHandler handles all HTTP endpoints for task recurrence rules.
type RecurrenceHandler struct {
	usecase recurrenceusecase.Usecase
}

// NewRecurrenceHandler constructs a handler backed by the given usecase.
func NewRecurrenceHandler(usecase recurrenceusecase.Usecase) *RecurrenceHandler {
	return &RecurrenceHandler{usecase: usecase}
}

// CreateRule handles POST /api/v1/tasks/{id}/recurrence.
// Creates a new recurrence rule attached to the task identified by path parameter {id}.
func (h *RecurrenceHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req createRecurrenceRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := h.usecase.CreateRule(r.Context(), toCreateRuleInput(taskID, req))
	if err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newRecurrenceRuleResponse(rule))
}

// GetRuleByTask handles GET /api/v1/tasks/{id}/recurrence.
// Returns the recurrence rule for the task identified by path parameter {id}.
func (h *RecurrenceHandler) GetRuleByTask(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := h.usecase.GetRuleByTask(r.Context(), taskID)
	if err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurrenceRuleResponse(rule))
}

// GetRule handles GET /api/v1/recurrence/{id}.
// Returns a recurrence rule by its own primary key.
func (h *RecurrenceHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id, err := getRecurrenceIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := h.usecase.GetRule(r.Context(), id)
	if err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurrenceRuleResponse(rule))
}

// UpdateRule handles PUT /api/v1/recurrence/{id}.
// Replaces the full configuration of the rule identified by {id}.
func (h *RecurrenceHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id, err := getRecurrenceIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req updateRecurrenceRuleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := h.usecase.UpdateRule(r.Context(), id, toUpdateRuleInput(req))
	if err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newRecurrenceRuleResponse(rule))
}

// DeleteRule handles DELETE /api/v1/recurrence/{id}.
func (h *RecurrenceHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id, err := getRecurrenceIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.DeleteRule(r.Context(), id); err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NextOccurrences handles POST /api/v1/tasks/{id}/recurrence/next.
// Computes the next N scheduled UTC dates for the task's recurrence rule.
//
// Request body:
//
//	{ "from": "2026-04-01T00:00:00Z", "n": 5 }
//
// The "from" field is optional; it defaults to now (UTC).
func (h *RecurrenceHandler) NextOccurrences(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req nextOccurrencesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	from := time.Now().UTC()
	if req.From != nil {
		from = req.From.UTC()
	}

	occurrences, err := h.usecase.NextOccurrences(r.Context(), taskID, from, req.N)
	if err != nil {
		writeRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, nextOccurrencesResponse{
		TaskID:      taskID,
		Occurrences: occurrences,
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// getRecurrenceIDFromRequest reads {recurrence_id} from the path variables.
func getRecurrenceIDFromRequest(r *http.Request) (int64, error) {
	return parsePositiveInt64(mux.Vars(r)["recurrence_id"], "recurrence_id")
}

// writeRecurrenceUsecaseError maps domain and usecase errors to HTTP statuses.
func writeRecurrenceUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, recurrencedomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, recurrencedomain.ErrTaskAlreadyHasRule):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, recurrenceusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
