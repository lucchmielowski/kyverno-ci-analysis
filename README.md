# Kyverno CI Analysis

GitHub Actions workflow analysis tool for Kyverno project.

A Go-based tool for analyzing GitHub Actions workflow runs and generating comprehensive reports and metrics in multiple formats.

## Features

- 📊 **Weekly Reports**: Generate detailed markdown reports with visual graphs for GitHub Actions workflow performance
- 📈 **Multiple Metrics Formats**: Export metrics in Grafana JSON, Prometheus, InfluxDB, and raw JSON formats
- 🔍 **Workflow Analysis**: Track success rates, flaky test rates, duration metrics (average, P95), and more
- 📅 **Flexible Date Ranges**: Analyze workflows by week, custom date ranges, or last week only
- 🎨 **Visual Graphs**: Generate PNG graphs for P95 duration and flaky rate trends
- 📋 **Top Workflows**: Identify the most flaky and slowest workflows automatically

## Installation

### Prerequisites

- Go 1.25.0 or later
- GitHub Personal Access Token with `actions:read` permission

### Build

```bash
# Clone the repository
git clone <repository-url>
cd kyverno-ci-analysis

# Build the reports tool
go build -o bin/reports ./pkg/reports

# Build the metrics tool
go build -o bin/grafana ./pkg/grafana
```

## Configuration

Set your GitHub token as an environment variable:

```bash
export GITHUB_TOKEN=your_github_token_here
```

## Usage

### Reports Tool

Generate weekly markdown reports with visual graphs.

#### Basic Usage

```bash
# Generate reports for the last 8 weeks (default)
./bin/reports

# Generate report for last week only
./bin/reports -last-week

# Generate reports for a specific repository
./bin/reports -owner=myorg -repo=myrepo

# Generate reports for a custom date range
./bin/reports -start-date=2026-01-01 -end-date=2026-01-31

# Generate reports for last N weeks ending on a specific date
./bin/reports -weeks=4 -end-date=2026-01-31
```

#### Command Line Options

- `-owner`: GitHub repository owner (default: `kyverno`)
- `-repo`: GitHub repository name (default: `kyverno`)
- `-output`: Output directory for reports (default: `reports`)
- `-weeks`: Number of weeks to analyze from now or end-date (default: `8`)
- `-last-week`: Generate report only for last week (Mon-Sun)
- `-start-date`: Start date in YYYY-MM-DD format (overrides weeks)
- `-end-date`: End date in YYYY-MM-DD format (defaults to today)

#### Output

The reports tool generates:
- **Markdown files**: Weekly reports in `reports/week-YYYY-WW.md` format
- **Graph images**: PNG files for P95 duration and flaky rate trends
- **Index file**: `reports/README.md` with links to all reports

Each report includes:
- Summary statistics (total runs, success rate, flaky rate, durations)
- Daily trends with visual graphs
- Top 10 workflows by run count
- Daily breakdown table
- Top 5 most flaky workflows
- Top 5 slowest workflows (P95)

### Metrics Tool

Generate metrics in various formats for integration with monitoring systems.

#### Basic Usage

```bash
# Generate metrics for the last 8 weeks (default)
./bin/grafana

# Generate metrics for a specific repository
./bin/grafana -owner=myorg -repo=myrepo

# Generate metrics for a custom date range
./bin/grafana -start-date=2026-01-01 -end-date=2026-01-31

# Generate metrics for last N weeks
./bin/grafana -weeks=4
```

#### Command Line Options

- `-owner`: GitHub repository owner (default: `kyverno`)
- `-repo`: GitHub repository name (default: `kyverno`)
- `-output`: Output directory for metrics (default: `metrics`)
- `-weeks`: Number of weeks to analyze (default: `8`)
- `-start-date`: Start date in YYYY-MM-DD format
- `-end-date`: End date in YYYY-MM-DD format

#### Output Formats

The metrics tool generates files in the output directory:

1. **Grafana JSON** (`grafana.json`): Compatible with Grafana SimpleJson datasource
   - Time series data for overall metrics (success rate, flaky rate, durations, total runs)
   - Per-workflow metrics for top 10 workflows by run count

2. **Prometheus** (`prometheus.txt`): Prometheus text format metrics
   - Overall metrics: `github_actions_success_rate`, `github_actions_flaky_rate`, etc.
   - Per-workflow metrics: `github_actions_workflow_success_rate`, etc.

3. **InfluxDB** (`influxdb.txt`): InfluxDB line protocol format
   - Daily metrics with tags for repository and period
   - Per-workflow metrics with tags for repository and workflow name

4. **Raw JSON** (`raw.json`): Raw JSON format for custom integrations
   - Complete dataset with daily stats and workflow statistics

## Metrics Explained

### Success Rate
Percentage of workflow runs that completed successfully.

### Flaky Rate
Percentage of workflow runs that required multiple attempts (retries).

### Duration Metrics
- **Average Duration**: Mean execution time in minutes
- **P95 Duration**: 95th percentile execution time in minutes

### Workflow Statistics
Per-workflow statistics include:
- Total runs
- Failures count
- Flaky runs count
- Duration metrics

## Examples

### Generate Weekly Reports

```bash
# Generate reports for kyverno/kyverno (default)
export GITHUB_TOKEN=ghp_xxxxx
./bin/reports

# Generate reports for a different repository
./bin/reports -owner=kubernetes -repo=kubernetes -weeks=4
```

### Generate Metrics for Monitoring

```bash
# Generate metrics for Grafana
./bin/grafana -owner=myorg -repo=myrepo -output=./metrics

# Import grafana.json into Grafana using SimpleJson datasource
```

### Custom Date Range

```bash
# Analyze specific month
./bin/reports -start-date=2026-01-01 -end-date=2026-01-31

# Analyze last week only
./bin/reports -last-week
```

## Project Structure

```
kyverno-ci-analysis/
├── pkg/
│   ├── reports/          # Weekly reports generator
│   ├── grafana/          # Metrics generator
│   └── types/            # Shared type definitions
├── reports/              # Generated reports (output)
├── go.mod
└── README.md
```

## Dependencies

- [go-github](https://github.com/google/go-github): GitHub API client
- [gonum/plot](https://github.com/gonum/plot): Graph generation
- [golang.org/x/oauth2](https://golang.org/x/oauth2): OAuth2 authentication

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
