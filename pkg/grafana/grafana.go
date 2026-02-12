package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"

	"github.com/lucchmielowski/kyverno-ci-analysis/pkg/types"
)

var (
	owner     string
	repo      string
	outputDir string
	weeks     int
	startDate string
	endDate   string
)

// Grafana JSON format (compatible with SimpleJson datasource)
type GrafanaTimeSeries struct {
	Target     string          `json:"target"`
	Datapoints [][]interface{} `json:"datapoints"`
}

// Prometheus text format metrics
type PrometheusMetric struct {
	Name   string
	Labels map[string]string
	Value  float64
	Time   int64
}

// InfluxDB line protocol format
type InfluxDataPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   int64
}

func logInfo(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

func logDebug(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}

func logError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}

func init() {
	flag.StringVar(&owner, "owner", "kyverno", "GitHub repository owner")
	flag.StringVar(&repo, "repo", "kyverno", "GitHub repository name")
	flag.StringVar(&outputDir, "output", "metrics", "Output directory for metrics")
	flag.IntVar(&weeks, "weeks", 8, "Number of weeks to analyze")
	flag.StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD)")
	flag.StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD)")
}

func main() {
	flag.Parse()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		logError("GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logError("Failed to create output directory: %v", err)
		os.Exit(1)
	}

	// Calculate date range
	var rangeStart, rangeEnd time.Time
	var err error

	if startDate != "" {
		rangeStart, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			logError("Invalid start-date format: %v", err)
			os.Exit(1)
		}
		rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, time.UTC)

		if endDate != "" {
			rangeEnd, err = time.Parse("2006-01-02", endDate)
			if err != nil {
				logError("Invalid end-date format: %v", err)
				os.Exit(1)
			}
		} else {
			rangeEnd = time.Now()
		}
		rangeEnd = time.Date(rangeEnd.Year(), rangeEnd.Month(), rangeEnd.Day(), 23, 59, 59, 0, time.UTC)
	} else {
		if endDate != "" {
			rangeEnd, err = time.Parse("2006-01-02", endDate)
			if err != nil {
				logError("Invalid end-date format: %v", err)
				os.Exit(1)
			}
		} else {
			rangeEnd = time.Now()
		}
		rangeEnd = time.Date(rangeEnd.Year(), rangeEnd.Month(), rangeEnd.Day(), 23, 59, 59, 0, time.UTC)
		rangeStart = rangeEnd.AddDate(0, 0, -(weeks * 7))
		rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, time.UTC)
	}

	logInfo("Generating metrics for %s/%s from %s to %s",
		owner, repo,
		rangeStart.Format("2006-01-02"),
		rangeEnd.Format("2006-01-02"))

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	logInfo("Authenticating with GitHub API...")

	_, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		logError("Failed to authenticate: %v", err)
		os.Exit(1)
	}
	logInfo("Authentication successful (Rate limit: %d/%d)", resp.Rate.Remaining, resp.Rate.Limit)

	startTime := time.Now()
	logInfo("Fetching workflow runs...")

	runs, err := fetchWorkflowRuns(ctx, client, owner, repo, rangeStart)
	if err != nil {
		logError("Failed to fetch runs: %v", err)
		os.Exit(1)
	}

	logInfo("Fetched %d runs in %s", len(runs), time.Since(startTime).Round(time.Second))

	// Filter runs within date range
	var filteredRuns []types.WorkflowRun
	for _, run := range runs {
		if (run.CreatedAt.Equal(rangeStart) || run.CreatedAt.After(rangeStart)) &&
			(run.CreatedAt.Before(rangeEnd) || run.CreatedAt.Equal(rangeEnd)) {
			filteredRuns = append(filteredRuns, run)
		}
	}

	logInfo("Filtered to %d runs in date range", len(filteredRuns))

	if len(filteredRuns) == 0 {
		logError("No runs found in the specified date range")
		os.Exit(1)
	}

	// Group by day
	dailyData := make(map[string][]types.WorkflowRun)
	for _, run := range filteredRuns {
		day := run.CreatedAt.Format("2006-01-02")
		dailyData[day] = append(dailyData[day], run)
	}

	var days []string
	for day := range dailyData {
		days = append(days, day)
	}
	sort.Strings(days)

	// Calculate daily stats
	var dailyStats []types.DailyStats
	for _, day := range days {
		runs := dailyData[day]
		stats := calculateDayStats(runs, day)
		dailyStats = append(dailyStats, stats)
	}

	// Calculate per-workflow stats
	workflowStats := calculateWorkflowStats(filteredRuns)

	logInfo("Generating metrics files...")

	// Generate Grafana JSON format
	if err := generateGrafanaJSON(dailyStats, workflowStats); err != nil {
		logError("Failed to generate Grafana JSON: %v", err)
	} else {
		logInfo("Generated Grafana JSON: %s/grafana.json", outputDir)
	}

	// Generate Prometheus format
	if err := generatePrometheusMetrics(dailyStats, workflowStats); err != nil {
		logError("Failed to generate Prometheus metrics: %v", err)
	} else {
		logInfo("Generated Prometheus metrics: %s/prometheus.txt", outputDir)
	}

	// Generate InfluxDB line protocol
	if err := generateInfluxLineProtocol(dailyStats, workflowStats); err != nil {
		logError("Failed to generate InfluxDB metrics: %v", err)
	} else {
		logInfo("Generated InfluxDB metrics: %s/influxdb.txt", outputDir)
	}

	// Generate raw JSON for custom integrations
	if err := generateRawJSON(dailyStats, workflowStats); err != nil {
		logError("Failed to generate raw JSON: %v", err)
	} else {
		logInfo("Generated raw JSON: %s/raw.json", outputDir)
	}

	logInfo("Metrics generation complete! Total time: %s", time.Since(startTime).Round(time.Second))
	logInfo("All metrics saved to %s/", outputDir)
}

func generateGrafanaJSON(dailyStats []types.DailyStats, workflowStats map[string]*types.WorkflowDayStats) error {
	var series []GrafanaTimeSeries

	// Overall metrics time series
	successRateSeries := GrafanaTimeSeries{Target: "overall.success_rate", Datapoints: [][]interface{}{}}
	flakyRateSeries := GrafanaTimeSeries{Target: "overall.flaky_rate", Datapoints: [][]interface{}{}}
	avgDurationSeries := GrafanaTimeSeries{Target: "overall.avg_duration_minutes", Datapoints: [][]interface{}{}}
	p95DurationSeries := GrafanaTimeSeries{Target: "overall.p95_duration_minutes", Datapoints: [][]interface{}{}}
	totalRunsSeries := GrafanaTimeSeries{Target: "overall.total_runs", Datapoints: [][]interface{}{}}

	for _, stats := range dailyStats {
		t, _ := time.Parse("2006-01-02", stats.Date)
		timestamp := t.Unix() * 1000 // Grafana uses milliseconds

		successRateSeries.Datapoints = append(successRateSeries.Datapoints, []interface{}{stats.SuccessRate, timestamp})
		flakyRateSeries.Datapoints = append(flakyRateSeries.Datapoints, []interface{}{stats.FlakyRate, timestamp})
		avgDurationSeries.Datapoints = append(avgDurationSeries.Datapoints, []interface{}{stats.AvgDuration, timestamp})
		p95DurationSeries.Datapoints = append(p95DurationSeries.Datapoints, []interface{}{stats.P95Duration, timestamp})
		totalRunsSeries.Datapoints = append(totalRunsSeries.Datapoints, []interface{}{float64(stats.TotalRuns), timestamp})
	}

	series = append(series, successRateSeries, flakyRateSeries, avgDurationSeries, p95DurationSeries, totalRunsSeries)

	// Per-workflow metrics (top 10 by run count)
	type wfInfo struct {
		name  string
		stats *types.WorkflowDayStats
	}
	var workflows []wfInfo
	for name, stats := range workflowStats {
		workflows = append(workflows, wfInfo{name, stats})
	}
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].stats.TotalRuns > workflows[j].stats.TotalRuns
	})

	top10 := workflows
	if len(top10) > 10 {
		top10 = top10[:10]
	}

	now := time.Now().Unix() * 1000
	for _, wf := range top10 {
		safeName := strings.ReplaceAll(wf.name, " ", "_")
		safeName = strings.ReplaceAll(safeName, ".", "_")

		successRate := 100.0 - (float64(wf.stats.Failures) / float64(wf.stats.TotalRuns) * 100)
		flakyRate := float64(wf.stats.Flaky) / float64(wf.stats.TotalRuns) * 100

		series = append(series,
			GrafanaTimeSeries{
				Target:     fmt.Sprintf("workflow.%s.success_rate", safeName),
				Datapoints: [][]interface{}{{successRate, now}},
			},
			GrafanaTimeSeries{
				Target:     fmt.Sprintf("workflow.%s.flaky_rate", safeName),
				Datapoints: [][]interface{}{{flakyRate, now}},
			},
			GrafanaTimeSeries{
				Target:     fmt.Sprintf("workflow.%s.avg_duration", safeName),
				Datapoints: [][]interface{}{{wf.stats.AvgDuration, now}},
			},
			GrafanaTimeSeries{
				Target:     fmt.Sprintf("workflow.%s.p95_duration", safeName),
				Datapoints: [][]interface{}{{wf.stats.P95Duration, now}},
			},
			GrafanaTimeSeries{
				Target:     fmt.Sprintf("workflow.%s.total_runs", safeName),
				Datapoints: [][]interface{}{{float64(wf.stats.TotalRuns), now}},
			},
		)
	}

	file, err := os.Create(filepath.Join(outputDir, "grafana.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(series)
}

func generatePrometheusMetrics(dailyStats []types.DailyStats, workflowStats map[string]*types.WorkflowDayStats) error {
	file, err := os.Create(filepath.Join(outputDir, "prometheus.txt"))
	if err != nil {
		return err
	}
	defer file.Close()

	now := time.Now().Unix()

	// Write headers
	fmt.Fprintf(file, "# HELP github_actions_success_rate Success rate percentage\n")
	fmt.Fprintf(file, "# TYPE github_actions_success_rate gauge\n")

	// Overall metrics (latest day)
	if len(dailyStats) > 0 {
		latest := dailyStats[len(dailyStats)-1]
		fmt.Fprintf(file, "github_actions_success_rate{repo=\"%s/%s\"} %.2f %d\n", owner, repo, latest.SuccessRate, now)

		fmt.Fprintf(file, "\n# HELP github_actions_flaky_rate Flaky test rate percentage\n")
		fmt.Fprintf(file, "# TYPE github_actions_flaky_rate gauge\n")
		fmt.Fprintf(file, "github_actions_flaky_rate{repo=\"%s/%s\"} %.2f %d\n", owner, repo, latest.FlakyRate, now)

		fmt.Fprintf(file, "\n# HELP github_actions_avg_duration_minutes Average duration in minutes\n")
		fmt.Fprintf(file, "# TYPE github_actions_avg_duration_minutes gauge\n")
		fmt.Fprintf(file, "github_actions_avg_duration_minutes{repo=\"%s/%s\"} %.2f %d\n", owner, repo, latest.AvgDuration, now)

		fmt.Fprintf(file, "\n# HELP github_actions_p95_duration_minutes P95 duration in minutes\n")
		fmt.Fprintf(file, "# TYPE github_actions_p95_duration_minutes gauge\n")
		fmt.Fprintf(file, "github_actions_p95_duration_minutes{repo=\"%s/%s\"} %.2f %d\n", owner, repo, latest.P95Duration, now)

		fmt.Fprintf(file, "\n# HELP github_actions_total_runs Total number of runs\n")
		fmt.Fprintf(file, "# TYPE github_actions_total_runs counter\n")
		fmt.Fprintf(file, "github_actions_total_runs{repo=\"%s/%s\"} %d %d\n", owner, repo, latest.TotalRuns, now)
	}

	// Per-workflow metrics
	fmt.Fprintf(file, "\n# HELP github_actions_workflow_success_rate Workflow success rate percentage\n")
	fmt.Fprintf(file, "# TYPE github_actions_workflow_success_rate gauge\n")
	for name, stats := range workflowStats {
		successRate := 100.0 - (float64(stats.Failures) / float64(stats.TotalRuns) * 100)
		fmt.Fprintf(file, "github_actions_workflow_success_rate{repo=\"%s/%s\",workflow=\"%s\"} %.2f %d\n",
			owner, repo, name, successRate, now)
	}

	fmt.Fprintf(file, "\n# HELP github_actions_workflow_flaky_rate Workflow flaky rate percentage\n")
	fmt.Fprintf(file, "# TYPE github_actions_workflow_flaky_rate gauge\n")
	for name, stats := range workflowStats {
		flakyRate := float64(stats.Flaky) / float64(stats.TotalRuns) * 100
		fmt.Fprintf(file, "github_actions_workflow_flaky_rate{repo=\"%s/%s\",workflow=\"%s\"} %.2f %d\n",
			owner, repo, name, flakyRate, now)
	}

	fmt.Fprintf(file, "\n# HELP github_actions_workflow_p95_duration_minutes Workflow P95 duration in minutes\n")
	fmt.Fprintf(file, "# TYPE github_actions_workflow_p95_duration_minutes gauge\n")
	for name, stats := range workflowStats {
		fmt.Fprintf(file, "github_actions_workflow_p95_duration_minutes{repo=\"%s/%s\",workflow=\"%s\"} %.2f %d\n",
			owner, repo, name, stats.P95Duration, now)
	}

	return nil
}

func generateInfluxLineProtocol(dailyStats []types.DailyStats, workflowStats map[string]*types.WorkflowDayStats) error {
	file, err := os.Create(filepath.Join(outputDir, "influxdb.txt"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Daily metrics
	for _, stats := range dailyStats {
		t, _ := time.Parse("2006-01-02", stats.Date)
		timestamp := t.UnixNano()

		fmt.Fprintf(file, "github_actions,repo=%s/%s,period=daily success_rate=%.2f,flaky_rate=%.2f,avg_duration=%.2f,p95_duration=%.2f,total_runs=%di %d\n",
			owner, repo,
			stats.SuccessRate, stats.FlakyRate, stats.AvgDuration, stats.P95Duration, stats.TotalRuns,
			timestamp)
	}

	// Per-workflow metrics
	now := time.Now().UnixNano()
	for name, stats := range workflowStats {
		successRate := 100.0 - (float64(stats.Failures) / float64(stats.TotalRuns) * 100)
		flakyRate := float64(stats.Flaky) / float64(stats.TotalRuns) * 100

		escapedName := strings.ReplaceAll(name, " ", "\\ ")
		escapedName = strings.ReplaceAll(escapedName, ",", "\\,")
		escapedName = strings.ReplaceAll(escapedName, "=", "\\=")

		fmt.Fprintf(file, "github_actions_workflow,repo=%s/%s,workflow=%s success_rate=%.2f,flaky_rate=%.2f,avg_duration=%.2f,p95_duration=%.2f,total_runs=%di,failures=%di %d\n",
			owner, repo, escapedName,
			successRate, flakyRate, stats.AvgDuration, stats.P95Duration, stats.TotalRuns, stats.Failures,
			now)
	}

	return nil
}

func generateRawJSON(dailyStats []types.DailyStats, workflowStats map[string]*types.WorkflowDayStats) error {
	data := map[string]interface{}{
		"repository": fmt.Sprintf("%s/%s", owner, repo),
		"generated":  time.Now().Format(time.RFC3339),
		"daily":      dailyStats,
		"workflows":  workflowStats,
	}

	file, err := os.Create(filepath.Join(outputDir, "raw.json"))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func fetchWorkflowRuns(ctx context.Context, client *github.Client, owner, repo string, startDate time.Time) ([]types.WorkflowRun, error) {
	var allRuns []types.WorkflowRun

	opts := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	page := 1
	totalFetched := 0

	for {
		logDebug("Fetching page %d...", page)

		runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("API request failed on page %d: %w", page, err)
		}

		logDebug("Got %d runs from page %d (Rate limit: %d/%d)",
			len(runs.WorkflowRuns), page, resp.Rate.Remaining, resp.Rate.Limit)

		foundOld := false
		for _, run := range runs.WorkflowRuns {
			if run.CreatedAt.Time.Before(startDate) {
				foundOld = true
				break
			}

			conclusion := ""
			if run.Conclusion != nil {
				conclusion = *run.Conclusion
			}

			attempt := 1
			if run.RunAttempt != nil {
				attempt = *run.RunAttempt
			}

			headBranch := ""
			if run.HeadBranch != nil {
				headBranch = *run.HeadBranch
			}

			event := ""
			if run.Event != nil {
				event = *run.Event
			}

			allRuns = append(allRuns, types.WorkflowRun{
				Name:       run.GetName(),
				Conclusion: conclusion,
				CreatedAt:  run.CreatedAt.Time,
				UpdatedAt:  run.UpdatedAt.Time,
				Attempt:    attempt,
				HeadBranch: headBranch,
				Event:      event,
			})
			totalFetched++
		}

		if foundOld {
			logInfo("Stopped at page %d", page)
			break
		}

		if resp.NextPage == 0 {
			logInfo("Reached last page (%d)", page)
			break
		}

		opts.Page = resp.NextPage
		page++

		if page%5 == 0 {
			logInfo("Progress: fetched %d runs", totalFetched)
		}

		if page > 200 {
			logInfo("Reached safety limit of 200 pages")
			break
		}
	}

	return allRuns, nil
}

func calculateDayStats(runs []types.WorkflowRun, date string) types.DailyStats {
	stats := types.DailyStats{
		Date:      date,
		TotalRuns: len(runs),
	}

	if len(runs) == 0 {
		return stats
	}

	var durations []float64
	successCount := 0
	flakyCount := 0

	for _, run := range runs {
		duration := run.UpdatedAt.Sub(run.CreatedAt).Minutes()
		durations = append(durations, duration)

		if run.Conclusion == "success" {
			successCount++
		}
		if run.Attempt > 1 {
			flakyCount++
		}
	}

	stats.SuccessRate = float64(successCount) / float64(len(runs)) * 100
	stats.FlakyRate = float64(flakyCount) / float64(len(runs)) * 100
	stats.AvgDuration = average(durations)
	stats.P95Duration = percentile(durations, 0.95)

	return stats
}

func calculateWorkflowStats(runs []types.WorkflowRun) map[string]*types.WorkflowDayStats {
	workflowMap := make(map[string]*types.WorkflowDayStats)

	for _, run := range runs {
		if _, exists := workflowMap[run.Name]; !exists {
			workflowMap[run.Name] = &types.WorkflowDayStats{
				Durations: []float64{},
			}
		}

		stats := workflowMap[run.Name]
		stats.TotalRuns++

		duration := run.UpdatedAt.Sub(run.CreatedAt).Minutes()
		stats.Durations = append(stats.Durations, duration)

		if run.Conclusion != "success" {
			stats.Failures++
		}
		if run.Attempt > 1 {
			stats.Flaky++
		}
	}

	for _, stats := range workflowMap {
		stats.AvgDuration = average(stats.Durations)
		stats.P95Duration = percentile(stats.Durations, 0.95)
	}

	return workflowMap
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}