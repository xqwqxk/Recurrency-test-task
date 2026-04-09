package handlers

import (
	"time"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
	recurrenceusecase "example.com/taskservice/internal/usecase/recurrence"
)

// ─── request DTOs ─────────────────────────────────────────────────────────────

// createRecurrenceRuleRequest is the JSON body for POST /api/v1/tasks/{id}/recurrence.
type createRecurrenceRuleRequest struct {
	Type            recurrencedomain.Type                    `json:"type"`
	Active          bool                                     `json:"active"`
	Daily           *recurrencedomain.DailyConfig            `json:"daily,omitempty"`
	MonthlyFixedDay *recurrencedomain.MonthlyFixedDayConfig  `json:"monthly_fixed_day,omitempty"`
	MonthlyCustom   *recurrencedomain.MonthlyCustomConfig    `json:"monthly_custom_dates,omitempty"`
	MonthlyParity   *recurrencedomain.MonthlyParityConfig    `json:"monthly_parity,omitempty"`
}

// updateRecurrenceRuleRequest is the JSON body for PUT /api/v1/recurrence/{id}.
type updateRecurrenceRuleRequest struct {
	Type            recurrencedomain.Type                    `json:"type"`
	Active          bool                                     `json:"active"`
	Daily           *recurrencedomain.DailyConfig            `json:"daily,omitempty"`
	MonthlyFixedDay *recurrencedomain.MonthlyFixedDayConfig  `json:"monthly_fixed_day,omitempty"`
	MonthlyCustom   *recurrencedomain.MonthlyCustomConfig    `json:"monthly_custom_dates,omitempty"`
	MonthlyParity   *recurrencedomain.MonthlyParityConfig    `json:"monthly_parity,omitempty"`
}

// nextOccurrencesRequest is the JSON body for POST /api/v1/tasks/{id}/recurrence/next.
type nextOccurrencesRequest struct {
	// From is the exclusive start timestamp (ISO 8601 / RFC 3339).
	// Defaults to the current UTC time if omitted.
	From *time.Time `json:"from,omitempty"`
	// N is the number of future occurrences to return (1–365).
	N int `json:"n"`
}

// ─── response DTOs ───────────────────────────────────────────────────────────

// recurrenceRuleResponse is the canonical JSON representation returned to clients.
type recurrenceRuleResponse struct {
	ID              int64                                    `json:"id"`
	TaskID          int64                                    `json:"task_id"`
	Type            recurrencedomain.Type                    `json:"type"`
	Active          bool                                     `json:"active"`
	Daily           *recurrencedomain.DailyConfig            `json:"daily,omitempty"`
	MonthlyFixedDay *recurrencedomain.MonthlyFixedDayConfig  `json:"monthly_fixed_day,omitempty"`
	MonthlyCustom   *recurrencedomain.MonthlyCustomConfig    `json:"monthly_custom_dates,omitempty"`
	MonthlyParity   *recurrencedomain.MonthlyParityConfig    `json:"monthly_parity,omitempty"`
	CreatedAt       time.Time                                `json:"created_at"`
	UpdatedAt       time.Time                                `json:"updated_at"`
}

// nextOccurrencesResponse wraps the list of upcoming UTC timestamps.
type nextOccurrencesResponse struct {
	TaskID      int64       `json:"task_id"`
	Occurrences []time.Time `json:"occurrences"`
}

// ─── converters ──────────────────────────────────────────────────────────────

func newRecurrenceRuleResponse(r *recurrencedomain.Rule) recurrenceRuleResponse {
	return recurrenceRuleResponse{
		ID:              r.ID,
		TaskID:          r.TaskID,
		Type:            r.Type,
		Active:          r.Active,
		Daily:           r.Daily,
		MonthlyFixedDay: r.MonthlyFixedDay,
		MonthlyCustom:   r.MonthlyCustom,
		MonthlyParity:   r.MonthlyParity,
		CreatedAt:       r.CreatedAt.UTC(),
		UpdatedAt:       r.UpdatedAt.UTC(),
	}
}

func toCreateRuleInput(taskID int64, req createRecurrenceRuleRequest) recurrenceusecase.CreateRuleInput {
	return recurrenceusecase.CreateRuleInput{
		TaskID:          taskID,
		Type:            req.Type,
		Active:          req.Active,
		Daily:           req.Daily,
		MonthlyFixedDay: req.MonthlyFixedDay,
		MonthlyCustom:   req.MonthlyCustom,
		MonthlyParity:   req.MonthlyParity,
	}
}

func toUpdateRuleInput(req updateRecurrenceRuleRequest) recurrenceusecase.UpdateRuleInput {
	return recurrenceusecase.UpdateRuleInput{
		Type:            req.Type,
		Active:          req.Active,
		Daily:           req.Daily,
		MonthlyFixedDay: req.MonthlyFixedDay,
		MonthlyCustom:   req.MonthlyCustom,
		MonthlyParity:   req.MonthlyParity,
	}
}
