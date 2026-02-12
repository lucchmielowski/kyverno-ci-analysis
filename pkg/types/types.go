package types

import "time"

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	Name       string
	Conclusion string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Attempt    int
	HeadBranch string
	Event      string // Event that triggered the workflow (e.g., "pull_request", "push", "workflow_dispatch")
}

// DailyStats represents statistics for a single day
type DailyStats struct {
	Date          string
	TotalRuns     int
	SuccessRate   float64
	FlakyRate     float64
	AvgDuration   float64
	P95Duration   float64
	WorkflowStats map[string]*WorkflowDayStats
}

// WorkflowDayStats represents statistics for a workflow on a given day
type WorkflowDayStats struct {
	TotalRuns   int
	Failures    int
	Flaky       int
	Durations   []float64
	AvgDuration float64
	P95Duration float64
}
