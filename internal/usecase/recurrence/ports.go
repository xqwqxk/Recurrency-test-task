package recurrence

import (
	"context"
	"time"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
)

// Repository defines the persistence contract for recurrence rules.
// Implementations must be safe for concurrent use.
type Repository interface {
	// Create persists a new recurrence rule and returns it with ID and timestamps set.
	Create(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error)

	// GetByID returns the rule with the given primary key.
	// Returns recurrencedomain.ErrNotFound when no row matches.
	GetByID(ctx context.Context, id int64) (*recurrencedomain.Rule, error)

	// GetByTaskID returns the recurrence rule associated with a task.
	// Returns recurrencedomain.ErrNotFound when no rule exists for that task.
	GetByTaskID(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error)

	// Update overwrites the mutable fields of an existing rule.
	// Returns recurrencedomain.ErrNotFound when no row matches.
	Update(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error)

	// Delete removes the rule permanently.
	// Returns recurrencedomain.ErrNotFound when no row matches.
	Delete(ctx context.Context, id int64) error
}

// Usecase defines the application-level operations for recurrence configuration.
type Usecase interface {
	// CreateRule attaches a new recurrence rule to an existing task.
	CreateRule(ctx context.Context, input CreateRuleInput) (*recurrencedomain.Rule, error)

	// GetRule returns a rule by its own primary key.
	GetRule(ctx context.Context, id int64) (*recurrencedomain.Rule, error)

	// GetRuleByTask returns the rule associated with a given task.
	GetRuleByTask(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error)

	// UpdateRule replaces the configuration of an existing rule.
	UpdateRule(ctx context.Context, id int64, input UpdateRuleInput) (*recurrencedomain.Rule, error)

	// DeleteRule removes the recurrence rule for a task.
	DeleteRule(ctx context.Context, id int64) error

	// NextOccurrences computes the next n scheduled UTC dates for a task's rule.
	NextOccurrences(ctx context.Context, taskID int64, from time.Time, n int) ([]time.Time, error)
}

// ─── input DTOs ──────────────────────────────────────────────────────────────

// CreateRuleInput carries the data needed to create a new recurrence rule.
type CreateRuleInput struct {
	TaskID          int64
	Type            recurrencedomain.Type
	Active          bool
	Daily           *recurrencedomain.DailyConfig
	MonthlyFixedDay *recurrencedomain.MonthlyFixedDayConfig
	MonthlyCustom   *recurrencedomain.MonthlyCustomConfig
	MonthlyParity   *recurrencedomain.MonthlyParityConfig
}

// UpdateRuleInput carries the data allowed in a rule update.
// All strategy fields are replaced atomically; the caller must supply the
// complete new configuration, not just the changed fields.
type UpdateRuleInput struct {
	Type            recurrencedomain.Type
	Active          bool
	Daily           *recurrencedomain.DailyConfig
	MonthlyFixedDay *recurrencedomain.MonthlyFixedDayConfig
	MonthlyCustom   *recurrencedomain.MonthlyCustomConfig
	MonthlyParity   *recurrencedomain.MonthlyParityConfig
}
