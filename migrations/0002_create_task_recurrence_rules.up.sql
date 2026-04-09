-- Migration: 0002_create_task_recurrence_rules.up.sql
-- Adds the task_recurrence_rules table which stores recurrence configuration
-- for system tasks in a MIS context.
--
-- Design decisions:
--   * Each recurrence type uses a dedicated JSONB column rather than a single
--     polymorphic column. This makes it trivial to index or query individual
--     fields (e.g. daily_config->>'interval_days') without casting.
--   * The UNIQUE constraint on task_id enforces the one-rule-per-task invariant
--     at the database level; the application layer maps the resulting 23505 error
--     to ErrTaskAlreadyHasRule.
--   * TIMESTAMPTZ columns store all timestamps in UTC; the application always
--     writes UTC and reads UTC back.
--   * The foreign key to tasks is ON DELETE CASCADE: removing a task
--     automatically removes its recurrence rule, preventing orphan rows.

CREATE TABLE IF NOT EXISTS task_recurrence_rules (
    id                          BIGSERIAL PRIMARY KEY,
    task_id                     BIGINT      NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    type                        TEXT        NOT NULL,
    active                      BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Strategy columns: exactly one is non-NULL for a given row.
    daily_config                JSONB       NULL,
    monthly_fixed_day_config    JSONB       NULL,
    monthly_custom_dates_config JSONB       NULL,
    monthly_parity_config       JSONB       NULL,

    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One rule per task.
    CONSTRAINT uq_task_recurrence_rules_task_id UNIQUE (task_id),

    -- Exactly one strategy column must be non-NULL.
    CONSTRAINT chk_recurrence_single_strategy CHECK (
        (
            (daily_config IS NOT NULL)::INT +
            (monthly_fixed_day_config IS NOT NULL)::INT +
            (monthly_custom_dates_config IS NOT NULL)::INT +
            (monthly_parity_config IS NOT NULL)::INT
        ) = 1
    ),

    -- The type discriminator must be one of the known values.
    CONSTRAINT chk_recurrence_type CHECK (
        type IN ('daily', 'monthly_fixed_day', 'monthly_custom_dates', 'monthly_parity')
    )
);

-- Index for fast lookup by task_id (covered by the unique constraint, but
-- added explicitly for readability and queryability audits).
CREATE INDEX IF NOT EXISTS idx_task_recurrence_rules_task_id
    ON task_recurrence_rules (task_id);

-- Index to quickly retrieve all active rules (used by a hypothetical scheduler).
CREATE INDEX IF NOT EXISTS idx_task_recurrence_rules_active
    ON task_recurrence_rules (active)
    WHERE active = TRUE;
