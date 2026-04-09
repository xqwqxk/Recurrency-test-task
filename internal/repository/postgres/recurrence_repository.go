package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
)

// RecurrenceRepository implements recurrenceusecase.Repository against PostgreSQL.
// All JSON strategy columns are stored in dedicated JSONB columns for
// queryability; the type discriminator is stored in a separate TEXT column.
type RecurrenceRepository struct {
	pool *pgxpool.Pool
}

// NewRecurrenceRepository constructs a repository backed by pool.
func NewRecurrenceRepository(pool *pgxpool.Pool) *RecurrenceRepository {
	return &RecurrenceRepository{pool: pool}
}

// Create inserts a new recurrence rule. A unique index on task_id in the
// database enforces the one-rule-per-task invariant at the storage level;
// the repository maps that violation to ErrTaskAlreadyHasRule.
func (r *RecurrenceRepository) Create(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error) {
	const query = `
		INSERT INTO task_recurrence_rules (
			task_id, type, active,
			daily_config,
			monthly_fixed_day_config,
			monthly_custom_dates_config,
			monthly_parity_config,
			created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9
		)
		RETURNING
			id, task_id, type, active,
			daily_config,
			monthly_fixed_day_config,
			monthly_custom_dates_config,
			monthly_parity_config,
			created_at, updated_at
	`

	cols := ruleToColumns(rule)
	row := r.pool.QueryRow(ctx, query,
		cols.taskID, cols.ruleType, cols.active,
		cols.daily, cols.fixedDay, cols.custom, cols.parity,
		rule.CreatedAt, rule.UpdatedAt,
	)

	created, err := scanRule(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, recurrencedomain.ErrTaskAlreadyHasRule
		}
		return nil, err
	}
	return created, nil
}

// GetByID retrieves a rule by primary key.
func (r *RecurrenceRepository) GetByID(ctx context.Context, id int64) (*recurrencedomain.Rule, error) {
	const query = `
		SELECT
			id, task_id, type, active,
			daily_config,
			monthly_fixed_day_config,
			monthly_custom_dates_config,
			monthly_parity_config,
			created_at, updated_at
		FROM task_recurrence_rules
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	rule, err := scanRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurrencedomain.ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

// GetByTaskID retrieves the rule associated with a given task.
func (r *RecurrenceRepository) GetByTaskID(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error) {
	const query = `
		SELECT
			id, task_id, type, active,
			daily_config,
			monthly_fixed_day_config,
			monthly_custom_dates_config,
			monthly_parity_config,
			created_at, updated_at
		FROM task_recurrence_rules
		WHERE task_id = $1
	`

	row := r.pool.QueryRow(ctx, query, taskID)
	rule, err := scanRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurrencedomain.ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

// Update overwrites all mutable fields of an existing rule.
func (r *RecurrenceRepository) Update(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error) {
	const query = `
		UPDATE task_recurrence_rules
		SET
			type                        = $1,
			active                      = $2,
			daily_config                = $3,
			monthly_fixed_day_config    = $4,
			monthly_custom_dates_config = $5,
			monthly_parity_config       = $6,
			updated_at                  = $7
		WHERE id = $8
		RETURNING
			id, task_id, type, active,
			daily_config,
			monthly_fixed_day_config,
			monthly_custom_dates_config,
			monthly_parity_config,
			created_at, updated_at
	`

	cols := ruleToColumns(rule)
	row := r.pool.QueryRow(ctx, query,
		cols.ruleType, cols.active,
		cols.daily, cols.fixedDay, cols.custom, cols.parity,
		rule.UpdatedAt, rule.ID,
	)

	updated, err := scanRule(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, recurrencedomain.ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

// Delete removes a recurrence rule by primary key.
func (r *RecurrenceRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_recurrence_rules WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return recurrencedomain.ErrNotFound
	}
	return nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// ruleColumns is a flat, SQL-friendly representation of a Rule's fields.
// JSONB columns are stored as []byte (pgx handles marshalling natively).
type ruleColumns struct {
	taskID   int64
	ruleType string
	active   bool
	daily    interface{}
	fixedDay interface{}
	custom   interface{}
	parity   interface{}
}

// ruleToColumns decomposes a Rule into database column values.
// Strategy fields that don't match the active type are set to nil so that
// the corresponding JSONB column is stored as SQL NULL.
func ruleToColumns(rule *recurrencedomain.Rule) ruleColumns {
	return ruleColumns{
		taskID:   rule.TaskID,
		ruleType: string(rule.Type),
		active:   rule.Active,
		daily:    rule.Daily,
		fixedDay: rule.MonthlyFixedDay,
		custom:   rule.MonthlyCustom,
		parity:   rule.MonthlyParity,
	}
}

// scanRow is the interface satisfied by both pgx.Row and pgx.Rows.
type scanRow interface {
	Scan(dest ...any) error
}

// scanRule reads a full rule row from the database.
// pgx serialises JSONB columns into the target struct via its built-in codec.
func scanRule(row scanRow) (*recurrencedomain.Rule, error) {
	var (
		rule     recurrencedomain.Rule
		ruleType string
	)

	if err := row.Scan(
		&rule.ID,
		&rule.TaskID,
		&ruleType,
		&rule.Active,
		&rule.Daily,
		&rule.MonthlyFixedDay,
		&rule.MonthlyCustom,
		&rule.MonthlyParity,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	); err != nil {
		return nil, err
	}

	rule.Type = recurrencedomain.Type(ruleType)
	rule.CreatedAt = rule.CreatedAt.UTC()
	rule.UpdatedAt = rule.UpdatedAt.UTC()

	return &rule, nil
}

// isUniqueViolation returns true when err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505). Uses pgconn.PgError which is part of pgx/v5.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
