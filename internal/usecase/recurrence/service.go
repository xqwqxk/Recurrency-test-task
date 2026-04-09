package recurrence

import (
	"context"
	"errors"
	"fmt"
	"time"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
)

// Service implements the Usecase interface.
// It owns validation orchestration, domain-rule construction, and
// delegation to the Repository for all persistence operations.
type Service struct {
	repo Repository
	now  func() time.Time
}

// NewService constructs a production-ready Service.
// The now function is injected to allow deterministic testing.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// CreateRule validates input, builds the domain Rule, and persists it.
func (s *Service) CreateRule(ctx context.Context, input CreateRuleInput) (*recurrencedomain.Rule, error) {
	if input.TaskID <= 0 {
		return nil, fmt.Errorf("%w: task_id must be a positive integer", ErrInvalidInput)
	}

	rule := s.buildRule(input.TaskID, input.Type, input.Active,
		input.Daily, input.MonthlyFixedDay, input.MonthlyCustom, input.MonthlyParity)

	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	now := s.now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	created, err := s.repo.Create(ctx, rule)
	if err != nil {
		if errors.Is(err, recurrencedomain.ErrTaskAlreadyHasRule) {
			return nil, err
		}
		return nil, err
	}
	return created, nil
}

// GetRule returns a rule by primary key.
func (s *Service) GetRule(ctx context.Context, id int64) (*recurrencedomain.Rule, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be a positive integer", ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

// GetRuleByTask returns the rule attached to a task.
func (s *Service) GetRuleByTask(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error) {
	if taskID <= 0 {
		return nil, fmt.Errorf("%w: task_id must be a positive integer", ErrInvalidInput)
	}
	return s.repo.GetByTaskID(ctx, taskID)
}

// UpdateRule validates the new configuration and persists it.
// The rule type may be changed; the old strategy block is replaced entirely.
func (s *Service) UpdateRule(ctx context.Context, id int64, input UpdateRuleInput) (*recurrencedomain.Rule, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be a positive integer", ErrInvalidInput)
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated := s.buildRule(existing.TaskID, input.Type, input.Active,
		input.Daily, input.MonthlyFixedDay, input.MonthlyCustom, input.MonthlyParity)
	updated.ID = id
	updated.CreatedAt = existing.CreatedAt
	updated.UpdatedAt = s.now()

	if err := updated.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	return s.repo.Update(ctx, updated)
}

// DeleteRule removes a recurrence rule permanently.
func (s *Service) DeleteRule(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be a positive integer", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

// NextOccurrences computes the next n scheduled UTC dates for the task's rule.
// from is the reference point (exclusive); all returned dates are strictly
// after from. from is always interpreted as UTC regardless of its Location.
func (s *Service) NextOccurrences(ctx context.Context, taskID int64, from time.Time, n int) ([]time.Time, error) {
	if taskID <= 0 {
		return nil, fmt.Errorf("%w: task_id must be a positive integer", ErrInvalidInput)
	}
	if n <= 0 {
		return nil, fmt.Errorf("%w: n must be a positive integer", ErrInvalidInput)
	}
	if n > 365 {
		return nil, fmt.Errorf("%w: n must not exceed 365 (requested %d)", ErrInvalidInput, n)
	}

	rule, err := s.repo.GetByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if !rule.Active {
		return []time.Time{}, nil
	}

	return rule.NextOccurrences(from.UTC(), n)
}

// ─── internal builder ────────────────────────────────────────────────────────

// buildRule assembles a domain Rule from loose parameters.
// Strategy fields not matching the type are explicitly zeroed to prevent
// ghost data from leaking across update calls.
func (s *Service) buildRule(
	taskID int64,
	rType recurrencedomain.Type,
	active bool,
	daily *recurrencedomain.DailyConfig,
	fixedDay *recurrencedomain.MonthlyFixedDayConfig,
	custom *recurrencedomain.MonthlyCustomConfig,
	parity *recurrencedomain.MonthlyParityConfig,
) *recurrencedomain.Rule {
	rule := &recurrencedomain.Rule{
		TaskID: taskID,
		Type:   rType,
		Active: active,
	}
	switch rType {
	case recurrencedomain.TypeDaily:
		rule.Daily = daily
	case recurrencedomain.TypeMonthlyFixedDay:
		rule.MonthlyFixedDay = fixedDay
	case recurrencedomain.TypeMonthlyCustomDates:
		rule.MonthlyCustom = custom
	case recurrencedomain.TypeMonthlyParity:
		rule.MonthlyParity = parity
	}
	return rule
}
