// Package jobs defines Axis's background work abstraction.
//
// Phase 0 only declares the contract. The implementation is a Postgres-backed
// queue (River) introduced in Phase 4 (see TASKS.md); until then there are no
// scheduled jobs. Keeping the interface here lets the rest of the codebase
// depend on the abstraction, not the backend.
package jobs

import "context"

// Job is a unit of background work.
type Job interface {
	// Kind is a stable identifier used for routing and observability.
	Kind() string
}

// Scheduler enqueues and runs background jobs.
type Scheduler interface {
	// Enqueue schedules a job to run as soon as a worker is free.
	Enqueue(ctx context.Context, job Job) error
	// Start begins processing the work queue until ctx is cancelled.
	Start(ctx context.Context) error
}
