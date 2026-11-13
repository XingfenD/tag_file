package ai

// Task statuses stored in ai_tasks.status.
const (
	TaskQueued  = "queued"
	TaskRunning = "running"
	TaskDone    = "done"
	TaskFailed  = "failed"
)

// Review statuses stored in ai_suggestions.review_status. Nothing reaches the
// tag library before a human picks one of the two terminal states.
const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

// File level AI tagging states derived from the two tables above (see Task 10).
// They are display labels only: File.TagDone stays manual and untouched.
const (
	AIStatusRunning  = "running"
	AIStatusFailed   = "failed"
	AIStatusPending  = "pending"
	AIStatusApproved = "approved"
)
