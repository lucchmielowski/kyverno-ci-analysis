package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"

	"github.com/lucchmielowski/kyverno-ci-analysis/pkg/types"
)

var (
	owner     string
	repo      string
	outputDir string
	weeks     int
	lastWeek  bool
	startDate string
	endDate   string
)

type WeekInfo struct {
	Year       int
	WeekNumber int
	StartDate  time.Time
	EndDate    time.Time
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
	flag.StringVar(&outputDir, "output", "reports", "Output directory for reports")
	flag.IntVar(&weeks, "weeks", 8, "Number of weeks to analyze (from now or end-date)")
	flag.BoolVar(&lastWeek, "last-week", false, "Generate report only for last week (Mon-Sun)")
	flag.StringVar(&startDate, "start-date", "", "Start date (YYYY-MM-DD) - overrides weeks")
	flag.StringVar(&endDate, "end-date", "", "End date (YYYY-MM-DD) - defaults to today")
}

func main() {
	flag.Parse()

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		logError("GITHUB_TOKEN environment variable not set")
		os.Exit(1)
	}

	// Create output directory (defaults to "reports")
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		logError("Failed to resolve output directory path: %v", err)
		os.Exit(1)
	}
	if err = os.MkdirAll(absOutputDir, 0755); err != nil {
		logError("Failed to create output directory: %v", err)
		os.Exit(1)
	}
	outputDir = absOutputDir
	logInfo("Reports will be saved to: %s", outputDir)

	// Calculate date range
	var rangeStart, rangeEnd time.Time

	if lastWeek {
		// Last week = previous Mon-Sun
		now := time.Now()
		daysToMonday := int(now.Weekday() - time.Monday)
		if daysToMonday < 0 {
			daysToMonday += 7
		}
		thisMonday := now.AddDate(0, 0, -daysToMonday)
		lastMonday := thisMonday.AddDate(0, 0, -7)
		lastSunday := lastMonday.AddDate(0, 0, 6)

		rangeStart = time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, time.UTC)
		rangeEnd = time.Date(lastSunday.Year(), lastSunday.Month(), lastSunday.Day(), 23, 59, 59, 0, time.UTC)

		logInfo("Last week mode: %s to %s", rangeStart.Format("2006-01-02"), rangeEnd.Format("2006-01-02"))
	} else if startDate != "" {
		rangeStart, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			logError("Invalid start-date format (use YYYY-MM-DD): %v", err)
			os.Exit(1)
		}
		rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, time.UTC)

		if endDate != "" {
			rangeEnd, err = time.Parse("2006-01-02", endDate)
			if err != nil {
				logError("Invalid end-date format (use YYYY-MM-DD): %v", err)
				os.Exit(1)
			}
		} else {
			rangeEnd = time.Now()
		}
		rangeEnd = time.Date(rangeEnd.Year(), rangeEnd.Month(), rangeEnd.Day(), 23, 59, 59, 0, time.UTC)

		logInfo("Custom range: %s to %s", rangeStart.Format("2006-01-02"), rangeEnd.Format("2006-01-02"))
	} else {
		// Default: last N weeks
		if endDate != "" {
			rangeEnd, err = time.Parse("2006-01-02", endDate)
			if err != nil {
				logError("Invalid end-date format (use YYYY-MM-DD): %v", err)
				os.Exit(1)
			}
		} else {
			rangeEnd = time.Now()
		}
		rangeEnd = time.Date(rangeEnd.Year(), rangeEnd.Month(), rangeEnd.Day(), 23, 59, 59, 0, time.UTC)
		rangeStart = rangeEnd.AddDate(0, 0, -(weeks * 7))
		rangeStart = time.Date(rangeStart.Year(), rangeStart.Month(), rangeStart.Day(), 0, 0, 0, 0, time.UTC)

		logInfo("Analyzing last %d weeks: %s to %s", weeks, rangeStart.Format("2006-01-02"), rangeEnd.Format("2006-01-02"))
	}

	logInfo("Starting analysis for %s/%s", owner, repo)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	logInfo("Authenticating with GitHub API...")

	// Test API access
	_, resp, err := client.Users.Get(ctx, "")
	if err != nil {
		logError("Failed to authenticate: %v", err)
		os.Exit(1)
	}
	logInfo("Authentication successful (Rate limit: %d/%d)", resp.Rate.Remaining, resp.Rate.Limit)

	logInfo("Fetching workflow runs (this may take a while)...")
	startTime := time.Now()

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

	if len(filteredRuns) > 0 {
		logInfo("Date range: %s to %s",
			filteredRuns[len(filteredRuns)-1].CreatedAt.Format("2006-01-02"),
			filteredRuns[0].CreatedAt.Format("2006-01-02"))
	}

	if len(filteredRuns) == 0 {
		logError("No runs found in the specified date range")
		os.Exit(1)
	}

	// Group runs by week
	weeklyRuns := groupByWeek(filteredRuns)

	// Generate weeks for the entire date range, even if they have no data
	allWeeks := generateWeeksForDateRange(rangeStart, rangeEnd)
	
	// Merge weeks with data into the complete week list
	weekMap := make(map[string]WeekInfo)
	for _, week := range allWeeks {
		key := fmt.Sprintf("%d-%02d", week.Year, week.WeekNumber)
		weekMap[key] = week
	}
	// Update with weeks that have data
	for _, week := range weeklyRuns {
		key := fmt.Sprintf("%d-%02d", week.Year, week.WeekNumber)
		weekMap[key] = week
	}
	
	// Convert back to sorted slice
	var finalWeeks []WeekInfo
	for _, week := range weekMap {
		finalWeeks = append(finalWeeks, week)
	}
	sort.Slice(finalWeeks, func(i, j int) bool {
		if finalWeeks[i].Year != finalWeeks[j].Year {
			return finalWeeks[i].Year < finalWeeks[j].Year
		}
		return finalWeeks[i].WeekNumber < finalWeeks[j].WeekNumber
	})

	logInfo("Found %d weeks in date range (%d with data)", len(finalWeeks), len(weeklyRuns))

	// Generate report for each week
	for _, week := range finalWeeks {
		logInfo("Generating report for week %d of %d (%s to %s)...",
			week.WeekNumber, week.Year,
			week.StartDate.Format("2006-01-02"),
			week.EndDate.Format("2006-01-02"))

		filename := fmt.Sprintf("week-%d-%02d.md", week.Year, week.WeekNumber)
		outputFile := filepath.Join(outputDir, filename)

		if err := generateWeekReport(outputFile, week, filteredRuns); err != nil {
			logError("Failed to generate report for week %d: %v", week.WeekNumber, err)
			continue
		}

		logInfo("Report written to %s", outputFile)
	}

	// Generate overview/index file
	indexFile := filepath.Join(outputDir, "README.md")
	if err := generateIndexFile(indexFile, weeklyRuns); err != nil {
		logError("Failed to generate index file: %v", err)
	} else {
		logInfo("Index written to %s", indexFile)
	}

	logInfo("Analysis complete! Total time: %s", time.Since(startTime).Round(time.Second))
	logInfo("All reports saved to %s/", outputDir)
}

func groupByWeek(runs []types.WorkflowRun) []WeekInfo {
	weekMap := make(map[string]WeekInfo)

	for _, run := range runs {
		year, week := run.CreatedAt.ISOWeek()
		key := fmt.Sprintf("%d-%02d", year, week)

		if _, exists := weekMap[key]; !exists {
			// Calculate week start (Monday) and end (Sunday)
			startOfWeek := startOfISOWeek(year, week)
			endOfWeek := startOfWeek.AddDate(0, 0, 6)

			weekMap[key] = WeekInfo{
				Year:       year,
				WeekNumber: week,
				StartDate:  startOfWeek,
				EndDate:    endOfWeek,
			}
		}
	}

	// Convert to sorted slice
	var weeks []WeekInfo
	for _, week := range weekMap {
		weeks = append(weeks, week)
	}

	sort.Slice(weeks, func(i, j int) bool {
		if weeks[i].Year != weeks[j].Year {
			return weeks[i].Year < weeks[j].Year
		}
		return weeks[i].WeekNumber < weeks[j].WeekNumber
	})

	return weeks
}

func generateWeeksForDateRange(startDate, endDate time.Time) []WeekInfo {
	var weeks []WeekInfo
	seenWeeks := make(map[string]bool)
	
	// Find the Monday of the week containing startDate
	year, weekNum := startDate.ISOWeek()
	currentWeekStart := startOfISOWeek(year, weekNum)
	
	// If startDate is before the Monday of its week, we need to include the previous week
	if startDate.Before(currentWeekStart) {
		prevYear, prevWeek := startDate.AddDate(0, 0, -7).ISOWeek()
		currentWeekStart = startOfISOWeek(prevYear, prevWeek)
		year, weekNum = prevYear, prevWeek
	}
	
	// Iterate week by week (Monday to Monday) until we've covered endDate
	current := currentWeekStart
	for !current.After(endDate) {
		year, weekNum = current.ISOWeek()
		key := fmt.Sprintf("%d-%02d", year, weekNum)
		
		// Skip if we've already processed this week
		if seenWeeks[key] {
			current = current.AddDate(0, 0, 7)
			continue
		}
		seenWeeks[key] = true
		
		// Calculate week start (Monday) and end (Sunday)
		startOfWeek := startOfISOWeek(year, weekNum)
		endOfWeek := startOfWeek.AddDate(0, 0, 6)
		
		// Only add if this week overlaps with our date range
		if !endOfWeek.Before(startDate) && !startOfWeek.After(endDate) {
			weeks = append(weeks, WeekInfo{
				Year:       year,
				WeekNumber: weekNum,
				StartDate:  startOfWeek,
				EndDate:    endOfWeek,
			})
		}
		
		// Move to next week (Monday)
		current = startOfWeek.AddDate(0, 0, 7)
	}
	
	return weeks
}

func startOfISOWeek(year, week int) time.Time {
	// January 4th is always in week 1
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)

	// Find the Monday of week 1
	daysToMonday := int(time.Monday - jan4.Weekday())
	if daysToMonday > 0 {
		daysToMonday -= 7
	}

	week1Monday := jan4.AddDate(0, 0, daysToMonday)

	// Add weeks
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

func filterRunsForWeek(runs []types.WorkflowRun, week WeekInfo) []types.WorkflowRun {
	var weekRuns []types.WorkflowRun

	for _, run := range runs {
		if (run.CreatedAt.Equal(week.StartDate) || run.CreatedAt.After(week.StartDate)) &&
			(run.CreatedAt.Before(week.EndDate.Add(24 * time.Hour))) {
			weekRuns = append(weekRuns, run)
		}
	}

	return weekRuns
}

func generateWeekReport(outputFile string, week WeekInfo, allRuns []types.WorkflowRun) error {
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Filter runs for this week
	weekRuns := filterRunsForWeek(allRuns, week)

	if len(weekRuns) == 0 {
		fmt.Fprintf(f, "# Week %d of %d - No Data\n\n", week.WeekNumber, week.Year)
		fmt.Fprintf(f, "**Period:** %s to %s\n\n",
			week.StartDate.Format("2006-01-02"),
			week.EndDate.Format("2006-01-02"))
		fmt.Fprintf(f, "No workflow runs found for this week.\n")
		return nil
	}

	fmt.Fprintf(f, "# Week %d of %d\n\n", week.WeekNumber, week.Year)
	fmt.Fprintf(f, "**Period:** %s to %s\n",
		week.StartDate.Format("2006-01-02"),
		week.EndDate.Format("2006-01-02"))
	fmt.Fprintf(f, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Calculate workflow stats
	workflowStats := calculateWorkflowStats(weekRuns)

	// Get top 10 workflows by run count
	type workflowInfo struct {
		name  string
		stats *types.WorkflowDayStats
	}
	var allWorkflows []workflowInfo
	for name, stats := range workflowStats {
		allWorkflows = append(allWorkflows, workflowInfo{name, stats})
	}
	sort.Slice(allWorkflows, func(i, j int) bool {
		return allWorkflows[i].stats.TotalRuns > allWorkflows[j].stats.TotalRuns
	})

	top10 := allWorkflows
	if len(top10) > 10 {
		top10 = top10[:10]
	}

	// Summary
	fmt.Fprintf(f, "## Summary\n\n")
	allStats := calculateDayStats(weekRuns)
	fmt.Fprintf(f, "- **Total runs**: %d\n", allStats.TotalRuns)
	fmt.Fprintf(f, "- **Success rate**: %.1f%%\n", allStats.SuccessRate)
	fmt.Fprintf(f, "- **Flaky rate**: %.1f%%\n", allStats.FlakyRate)
	fmt.Fprintf(f, "- **Average duration**: %.1f min\n", allStats.AvgDuration)
	fmt.Fprintf(f, "- **P95 duration**: %.1f min\n\n", allStats.P95Duration)

	// End-user experience (PR-triggered runs)
	prRuns := filterPRRuns(weekRuns)
	if len(prRuns) > 0 {
		fmt.Fprintf(f, "## End-User Experience (Pull Request Runs)\n\n")
		fmt.Fprintf(f, "Metrics for PR-triggered workflow runs - this represents the experience developers have when submitting pull requests.\n\n")
		prStats := calculateDayStats(prRuns)
		fmt.Fprintf(f, "- **PR-triggered runs**: %d (%.1f%% of total)\n", prStats.TotalRuns, float64(prStats.TotalRuns)/float64(allStats.TotalRuns)*100)
		fmt.Fprintf(f, "- **Success rate**: %.1f%%\n", prStats.SuccessRate)
		fmt.Fprintf(f, "- **Flaky rate**: %.1f%%\n", prStats.FlakyRate)
		fmt.Fprintf(f, "- **Average duration**: %.1f min\n", prStats.AvgDuration)
		fmt.Fprintf(f, "- **P95 duration**: %.1f min\n\n", prStats.P95Duration)
	}

	// Group by day for daily trends
	dailyData := make(map[string][]types.WorkflowRun)
	for _, run := range weekRuns {
		day := run.CreatedAt.Format("2006-01-02")
		dailyData[day] = append(dailyData[day], run)
	}

	var days []string
	for day := range dailyData {
		days = append(days, day)
	}
	sort.Strings(days)

	// Daily trends
	fmt.Fprintf(f, "## Daily Trends\n\n")
	fmt.Fprintf(f, "### Overall Metrics (All Runs)\n\n")

	var dailyStatsList []types.DailyStats
	for _, day := range days {
		runs := dailyData[day]
		stats := calculateDayStats(runs)
		stats.Date = day
		dailyStatsList = append(dailyStatsList, stats)
	}

	// P95 Latency over time
	fmt.Fprintf(f, "#### P95 Latency Over Time (minutes)\n\n")
	p95ImagePath := fmt.Sprintf("p95-duration-week-%d-%02d.png", week.Year, week.WeekNumber)
	if err := generateLineGraphImage(filepath.Join(filepath.Dir(outputFile), p95ImagePath), dailyStatsList, func(s types.DailyStats) float64 { return s.P95Duration }, "P95 Duration (minutes)", "Date", "Duration (min)"); err != nil {
		logError("Failed to generate P95 duration graph: %v", err)
	} else {
		fmt.Fprintf(f, "![P95 Duration](./%s)\n\n", p95ImagePath)
	}

	// Flaky rate over time
	fmt.Fprintf(f, "#### Flaky Rate Over Time (%%)\n\n")
	flakyImagePath := fmt.Sprintf("flaky-rate-week-%d-%02d.png", week.Year, week.WeekNumber)
	if err := generateLineGraphImage(filepath.Join(filepath.Dir(outputFile), flakyImagePath), dailyStatsList, func(s types.DailyStats) float64 { return s.FlakyRate }, "Flaky Rate (%)", "Date", "Flaky Rate (%)"); err != nil {
		logError("Failed to generate flaky rate graph: %v", err)
	} else {
		fmt.Fprintf(f, "![Flaky Rate](./%s)\n\n", flakyImagePath)
	}

	// PR-triggered runs trends
	prDailyData := make(map[string][]types.WorkflowRun)
	for _, run := range prRuns {
		day := run.CreatedAt.Format("2006-01-02")
		prDailyData[day] = append(prDailyData[day], run)
	}

	if len(prRuns) > 0 {
		fmt.Fprintf(f, "### End-User Experience (PR-Triggered Runs)\n\n")
		var prDailyStatsList []types.DailyStats
		for _, day := range days {
			if runs, exists := prDailyData[day]; exists {
				stats := calculateDayStats(runs)
				stats.Date = day
				prDailyStatsList = append(prDailyStatsList, stats)
			}
		}

		if len(prDailyStatsList) > 0 {
			// P95 Latency for PR runs
			fmt.Fprintf(f, "#### P95 Latency Over Time - PR Runs (minutes)\n\n")
			prP95ImagePath := fmt.Sprintf("p95-duration-pr-week-%d-%02d.png", week.Year, week.WeekNumber)
			if err := generateLineGraphImage(filepath.Join(filepath.Dir(outputFile), prP95ImagePath), prDailyStatsList, func(s types.DailyStats) float64 { return s.P95Duration }, "P95 Duration - PR Runs (minutes)", "Date", "Duration (min)"); err != nil {
				logError("Failed to generate PR P95 duration graph: %v", err)
			} else {
				fmt.Fprintf(f, "![P95 Duration - PR Runs](./%s)\n\n", prP95ImagePath)
			}

			// Flaky rate for PR runs
			fmt.Fprintf(f, "#### Flaky Rate Over Time - PR Runs (%%)\n\n")
			prFlakyImagePath := fmt.Sprintf("flaky-rate-pr-week-%d-%02d.png", week.Year, week.WeekNumber)
			if err := generateLineGraphImage(filepath.Join(filepath.Dir(outputFile), prFlakyImagePath), prDailyStatsList, func(s types.DailyStats) float64 { return s.FlakyRate }, "Flaky Rate - PR Runs (%)", "Date", "Flaky Rate (%)"); err != nil {
				logError("Failed to generate PR flaky rate graph: %v", err)
			} else {
				fmt.Fprintf(f, "![Flaky Rate - PR Runs](./%s)\n\n", prFlakyImagePath)
			}
		}
	}

	// Top 10 Workflows Overview
	fmt.Fprintf(f, "## Per-Workflow Analysis\n\n")
	fmt.Fprintf(f, "Breakdown by individual workflow to identify improvement opportunities.\n\n")
	fmt.Fprintf(f, "### Top 10 Workflows by Run Count\n\n")
	fmt.Fprintf(f, "| Rank | Workflow | Total Runs | Success Rate | Flaky Rate | Avg Duration (min) | P95 Duration (min) |\n")
	fmt.Fprintf(f, "|------|----------|------------|--------------|------------|--------------------|--------------------|")
	fmt.Fprintf(f, "\n")

	for i, wf := range top10 {
		successRate := 100.0 - (float64(wf.stats.Failures) / float64(wf.stats.TotalRuns) * 100)
		flakyRate := float64(wf.stats.Flaky) / float64(wf.stats.TotalRuns) * 100
		fmt.Fprintf(f, "| %d | %s | %d | %.1f%% | %.1f%% | %.1f | %.1f |\n",
			i+1, wf.name, wf.stats.TotalRuns, successRate, flakyRate,
			wf.stats.AvgDuration, wf.stats.P95Duration)
	}
	fmt.Fprintf(f, "\n")

	// Daily Overview table
	fmt.Fprintf(f, "## Daily Breakdown\n\n")
	fmt.Fprintf(f, "| Date | Total Runs | Success Rate | Flaky Rate | Avg Duration (min) | P95 Duration (min) |\n")
	fmt.Fprintf(f, "|------|------------|--------------|------------|--------------------|--------------------|")
	fmt.Fprintf(f, "\n")

	for _, day := range days {
		runs := dailyData[day]
		stats := calculateDayStats(runs)

		fmt.Fprintf(f, "| %s | %d | %.1f%% | %.1f%% | %.1f | %.1f |\n",
			day,
			stats.TotalRuns,
			stats.SuccessRate,
			stats.FlakyRate,
			stats.AvgDuration,
			stats.P95Duration,
		)
	}

	// Top 5 Flaky and Slow
	fmt.Fprintf(f, "\n## Top 5 Most Flaky Workflows\n\n")

	type flakyWorkflow struct {
		name string
		rate float64
		runs int
	}

	var flaky []flakyWorkflow
	for name, stats := range workflowStats {
		if stats.TotalRuns >= 3 {
			rate := float64(stats.Flaky) / float64(stats.TotalRuns) * 100
			flaky = append(flaky, flakyWorkflow{name, rate, stats.TotalRuns})
		}
	}

	sort.Slice(flaky, func(i, j int) bool {
		return flaky[i].rate > flaky[j].rate
	})

	for i := 0; i < len(flaky) && i < 5; i++ {
		fmt.Fprintf(f, "%d. **%s** - %.1f%% flaky (%d runs)\n", i+1, flaky[i].name, flaky[i].rate, flaky[i].runs)
	}

	fmt.Fprintf(f, "\n## Top 5 Slowest Workflows (P95)\n\n")

	type slowWorkflow struct {
		name     string
		duration float64
		runs     int
	}

	var slow []slowWorkflow
	for name, stats := range workflowStats {
		if stats.TotalRuns >= 3 {
			slow = append(slow, slowWorkflow{name, stats.P95Duration, stats.TotalRuns})
		}
	}

	sort.Slice(slow, func(i, j int) bool {
		return slow[i].duration > slow[j].duration
	})

	for i := 0; i < len(slow) && i < 5; i++ {
		fmt.Fprintf(f, "%d. **%s** - %.1f min P95 (%d runs)\n", i+1, slow[i].name, slow[i].duration, slow[i].runs)
	}

	return nil
}

func generateIndexFile(filename string, weeks []WeekInfo) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "# GitHub Actions Weekly Reports - %s/%s\n\n", owner, repo)
	fmt.Fprintf(f, "**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(f, "## Available Reports\n\n")

	// Reverse order to show most recent first
	for i := len(weeks) - 1; i >= 0; i-- {
		week := weeks[i]
		filename := fmt.Sprintf("week-%d-%02d.md", week.Year, week.WeekNumber)
		fmt.Fprintf(f, "- [Week %d of %d](./%s) (%s to %s)\n",
			week.WeekNumber, week.Year, filename,
			week.StartDate.Format("Jan 02"),
			week.EndDate.Format("Jan 02, 2006"))
	}

	return nil
}

func generateLineGraphImage(imagePath string, data []types.DailyStats, getValue func(types.DailyStats) float64, title, xLabel, yLabel string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data available")
	}

	// Create a new plot
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	// Create plotter points
	pts := make(plotter.XYs, len(data))
	for i, d := range data {
		// Parse date and convert to float64 for x-axis
		t, err := time.Parse("2006-01-02", d.Date)
		if err != nil {
			return fmt.Errorf("failed to parse date %s: %w", d.Date, err)
		}
		pts[i].X = float64(t.Unix())
		pts[i].Y = getValue(d)
	}

	// Create line plotter
	line, err := plotter.NewLine(pts)
	if err != nil {
		return fmt.Errorf("failed to create line: %w", err)
	}
	line.Color = plotutil.Color(0) // Blue color
	line.Width = vg.Points(2)

	// Add line to plot
	p.Add(line)

	// Format x-axis as dates
	p.X.Tick.Marker = plot.TimeTicks{Format: "01-02"}

	// Save the plot to a PNG file
	if err := p.Save(8*vg.Inch, 4*vg.Inch, imagePath); err != nil {
		return fmt.Errorf("failed to save plot: %w", err)
	}

	return nil
}

func filterPRRuns(runs []types.WorkflowRun) []types.WorkflowRun {
	var prRuns []types.WorkflowRun
	for _, run := range runs {
		// Filter for pull_request events (includes pull_request, pull_request_target, etc.)
		if run.Event == "pull_request" || run.Event == "pull_request_target" {
			prRuns = append(prRuns, run)
		}
	}
	return prRuns
}

func fetchWorkflowRuns(ctx context.Context, client *github.Client, owner, repo string, startDate time.Time) ([]types.WorkflowRun, error) {
	logInfo("Fetching runs from %s onwards...", startDate.Format("2006-01-02"))

	// First, fetch the first page to determine total pages
	opts := &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	firstPage, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("API request failed on first page: %w", err)
	}

	logDebug("Got %d runs from first page (Rate limit: %d/%d)",
		len(firstPage.WorkflowRuns), resp.Rate.Remaining, resp.Rate.Limit)

	// Process first page
	var firstPageRuns []types.WorkflowRun
	for _, run := range firstPage.WorkflowRuns {
		if run.CreatedAt.Time.Before(startDate) {
			logDebug("Found run from %s (before cutoff) on first page",
				run.CreatedAt.Time.Format("2006-01-02"))
			// If first page has old runs, return what we have so far
			if len(firstPageRuns) > 0 {
				return firstPageRuns, nil
			}
			return []types.WorkflowRun{}, nil
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

		firstPageRuns = append(firstPageRuns, types.WorkflowRun{
			Name:       run.GetName(),
			Conclusion: conclusion,
			CreatedAt:  run.CreatedAt.Time,
			UpdatedAt:  run.UpdatedAt.Time,
			Attempt:    attempt,
			HeadBranch: headBranch,
			Event:      event,
		})
	}

	// Determine how many pages to fetch
	// resp.LastPage is the last page number (1-indexed), or 0 if there's only one page
	totalPages := resp.LastPage
	if totalPages == 0 || totalPages == 1 {
		// Only one page, return first page results
		return firstPageRuns, nil
	}
	if totalPages > 200 {
		totalPages = 200 // Safety limit
	}

	logInfo("Found %d total pages, fetching remaining %d pages with %d parallel workers...", totalPages, totalPages-1, 5)

	// Channel for page numbers to fetch (skip page 1, already fetched)
	pageChan := make(chan int, totalPages-1)
	// Channel for results
	type pageResult struct {
		page int
		runs []types.WorkflowRun
		err  error
	}
	resultChan := make(chan pageResult, totalPages-1)

	// Fill page channel (skip page 1, already fetched)
	for page := 2; page <= totalPages; page++ {
		pageChan <- page
	}
	close(pageChan)

	// Worker pool
	const numWorkers = 5
	var wg sync.WaitGroup
	var mu sync.Mutex
	stopEarly := false

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for page := range pageChan {
				// Check if we should stop early
				mu.Lock()
				shouldStop := stopEarly
				mu.Unlock()
				if shouldStop {
					return
				}

				logDebug("Worker %d: Fetching page %d...", workerID, page)

				opts := &github.ListWorkflowRunsOptions{
					ListOptions: github.ListOptions{
						Page:    page,
						PerPage: 100,
					},
				}

				runs, resp, err := client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
				if err != nil {
					resultChan <- pageResult{page: page, err: err}
					continue
				}

				logDebug("Worker %d: Got %d runs from page %d (Rate limit: %d/%d)",
					workerID, len(runs.WorkflowRuns), page, resp.Rate.Remaining, resp.Rate.Limit)

				mu.Lock()
				if resp.Rate.Remaining < 10 {
					logInfo("Warning: Low rate limit remaining (%d). Slowing down...", resp.Rate.Remaining)
					// Slow down if rate limit is low
					time.Sleep(1 * time.Second)
				}
				mu.Unlock()

				// Process runs
				var pageRuns []types.WorkflowRun
				foundOld := false
				for _, run := range runs.WorkflowRuns {
					// Check if run is too old
					if run.CreatedAt.Time.Before(startDate) {
						logDebug("Worker %d: Found run from %s (before cutoff) on page %d",
							workerID, run.CreatedAt.Time.Format("2006-01-02"), page)
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

					pageRuns = append(pageRuns, types.WorkflowRun{
						Name:       run.GetName(),
						Conclusion: conclusion,
						CreatedAt:  run.CreatedAt.Time,
						UpdatedAt:  run.UpdatedAt.Time,
						Attempt:    attempt,
						HeadBranch: headBranch,
						Event:      event,
					})
				}

				if foundOld {
					mu.Lock()
					stopEarly = true
					mu.Unlock()
					logInfo("Worker %d: Stopped at page %d after finding runs older than start date", workerID, page)
					// Still send the results we have for this page
				}

				// Send results (non-blocking if channel is full, but it shouldn't be)
				select {
				case resultChan <- pageResult{page: page, runs: pageRuns}:
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	pageResults := make(map[int][]types.WorkflowRun)
	var firstError error
	totalFetched := 0

	// Collect results until channel is closed or we stop early
	for result := range resultChan {
		if result.err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("API request failed on page %d: %w", result.page, result.err)
			}
			continue
		}
		pageResults[result.page] = result.runs
		totalFetched += len(result.runs)

		if len(pageResults)%5 == 0 {
			logInfo("Progress: fetched %d runs across %d pages so far...", totalFetched, len(pageResults))
		}

		// Check if we should stop early
		mu.Lock()
		shouldStop := stopEarly
		mu.Unlock()
		if shouldStop {
			logInfo("Stopping early after receiving %d pages", len(pageResults))
			// Continue reading remaining results that are already in flight
			for len(pageResults) < totalPages-1 {
				select {
				case result, ok := <-resultChan:
					if !ok {
						break
					}
					if result.err == nil {
						pageResults[result.page] = result.runs
						totalFetched += len(result.runs)
					}
				default:
					break
				}
			}
			break
		}
	}

	// Wait for all workers to finish (they should already be done)
	wg.Wait()

	if firstError != nil {
		return nil, firstError
	}

	// Combine results: start with first page, then add others in page order
	var allRuns []types.WorkflowRun
	allRuns = append(allRuns, firstPageRuns...)
	for page := 2; page <= totalPages; page++ {
		if runs, exists := pageResults[page]; exists {
			allRuns = append(allRuns, runs...)
		} else {
			// If page is missing and we stopped early, that's okay
			mu.Lock()
			shouldStop := stopEarly
			mu.Unlock()
			if shouldStop {
				break
			}
		}
	}

	logInfo("Total runs collected: %d", len(allRuns))
	if len(allRuns) > 0 {
		logInfo("Date range: %s to %s",
			allRuns[len(allRuns)-1].CreatedAt.Format("2006-01-02"),
			allRuns[0].CreatedAt.Format("2006-01-02"))
	}

	return allRuns, nil
}

// Helper function to process runs from GitHub API response
func processRuns(githubRuns []*github.WorkflowRun, startDate time.Time) []types.WorkflowRun {
	var runs []types.WorkflowRun
	for _, run := range githubRuns {
		if run.CreatedAt.Time.Before(startDate) {
			continue
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

		runs = append(runs, types.WorkflowRun{
			Name:       run.GetName(),
			Conclusion: conclusion,
			CreatedAt:  run.CreatedAt.Time,
			UpdatedAt:  run.UpdatedAt.Time,
			Attempt:    attempt,
			HeadBranch: headBranch,
			Event:      event,
		})
	}
	return runs
}

func calculateDayStats(runs []types.WorkflowRun) types.DailyStats {
	stats := types.DailyStats{
		TotalRuns:     len(runs),
		WorkflowStats: make(map[string]*types.WorkflowDayStats),
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

	// Calculate averages
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
