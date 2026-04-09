package recurrence

import "errors"

// ErrNotFound is returned when a requested recurrence rule does not exist.
var ErrNotFound = errors.New("recurrence rule not found")

// ErrTaskAlreadyHasRule is returned when attempting to create a second rule
// for a task that already has an active recurrence configuration.
var ErrTaskAlreadyHasRule = errors.New("task already has a recurrence rule")
