# GitHub Actions Analysis - Last 2 Months

**Repository:** kyverno/kyverno
**Generated:** 2026-02-12 16:53:16

## Summary

- **Total runs analyzed**: 1000
- **Overall success rate**: 66.4%
- **Overall flaky rate**: 11.7%
- **Average duration**: 5.5 min
- **P95 duration**: 23.2 min

## Daily Trends

### P95 Latency Over Time (minutes)

```
24.2 │
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
23.7 │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │●                                                                               
23.2 │────────────────────────────────────────────────────────────────────────────────
      02-12                                                       
```

### Average Latency Over Time (minutes)

```
6.5 │
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
6.0 │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │●                                                                               
5.5 │────────────────────────────────────────────────────────────────────────────────
      02-12                                                       
```

### Flaky Rate Over Time (%)

```
12.7 │
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
12.2 │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │●                                                                               
11.7 │────────────────────────────────────────────────────────────────────────────────
      02-12                                                       
```

### Success Rate Over Time (%)

```
67.4 │
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
66.9 │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │                                                                                
      │●                                                                               
66.4 │────────────────────────────────────────────────────────────────────────────────
      02-12                                                       
```

## Top 10 Workflows by Run Count

| Rank | Workflow | Total Runs | Success Rate | Flaky Rate | Avg Duration (min) | P95 Duration (min) |
|------|----------|------------|--------------|------------|--------------------|--------------------|
| 1 | Retry workflow | 176 | 27.8% | 0.0% | 0.3 | 2.0 |
| 2 | Issue and PR comment commands | 84 | 100.0% | 0.0% | 1.6 | 6.0 |
| 3 | Tests | 69 | 66.7% | 11.6% | 11.4 | 25.5 |
| 4 | Conformance tests | 68 | 44.1% | 60.3% | 17.7 | 41.4 |
| 5 | check-milestone-label | 65 | 67.7% | 18.5% | 1.2 | 5.3 |
| 6 | PR Labelling | 64 | 98.4% | 0.0% | 1.3 | 4.2 |
| 7 | Build images | 57 | 71.9% | 14.0% | 5.4 | 11.2 |
| 8 | Build devcontainer | 57 | 78.9% | 14.0% | 4.2 | 9.9 |
| 9 | Verify codegen | 57 | 63.2% | 14.0% | 10.7 | 25.9 |
| 10 | Check actions | 57 | 94.7% | 14.0% | 1.2 | 6.0 |

## Top 10 Workflows - Visual Analysis

### Success Rate (Top 10)

```
 1. Retry workflow                 | █████████████ 27.8%
 2. Issue and PR comment commands  | ██████████████████████████████████████████████████ 100.0%
 3. Tests                          | █████████████████████████████████ 66.7%
 4. Conformance tests              | ██████████████████████ 44.1%
 5. check-milestone-label          | █████████████████████████████████ 67.7%
 6. PR Labelling                   | █████████████████████████████████████████████████ 98.4%
 7. Build images                   | ███████████████████████████████████ 71.9%
 8. Build devcontainer             | ███████████████████████████████████████ 78.9%
 9. Verify codegen                 | ███████████████████████████████ 63.2%
10. Check actions                  | ███████████████████████████████████████████████ 94.7%
```

### Flaky Rate (Top 10)

```
 1. Retry workflow                 |  0.0%
 2. Issue and PR comment commands  |  0.0%
 3. Tests                          | ███████████████████████ 11.6%
 4. Conformance tests              | ██████████████████████████████████████████████████ 60.3%
 5. check-milestone-label          | ████████████████████████████████████ 18.5%
 6. PR Labelling                   |  0.0%
 7. Build images                   | ████████████████████████████ 14.0%
 8. Build devcontainer             | ████████████████████████████ 14.0%
 9. Verify codegen                 | ████████████████████████████ 14.0%
10. Check actions                  | ████████████████████████████ 14.0%
```

### P95 Duration in Minutes (Top 10)

```
 1. Retry workflow                 | ██ 2.0 min
 2. Issue and PR comment commands  | ███████ 6.0 min
 3. Tests                          | ██████████████████████████████ 25.5 min
 4. Conformance tests              | ██████████████████████████████████████████████████ 41.4 min
 5. check-milestone-label          | ██████ 5.3 min
 6. PR Labelling                   | █████ 4.2 min
 7. Build images                   | █████████████ 11.2 min
 8. Build devcontainer             | ███████████ 9.9 min
 9. Verify codegen                 | ███████████████████████████████ 25.9 min
10. Check actions                  | ███████ 6.0 min
```

## Daily Overview Table

| Date | Total Runs | Success Rate | Flaky Rate | Avg Duration (min) | P95 Duration (min) |
|------|------------|--------------|------------|--------------------|--------------------|
| 2026-02-12 | 1000 | 66.4% | 11.7% | 5.5 | 23.2 |

## Complete Workflow Analysis

### Build devcontainer
- Total runs: 57
- Failure rate: 21.1%
- Flaky rate: 14.0%
- Avg duration: 4.2 min
- P95 duration: 9.9 min

### Build images
- Total runs: 57
- Failure rate: 28.1%
- Flaky rate: 14.0%
- Avg duration: 5.4 min
- P95 duration: 11.2 min

### Caching
- Total runs: 10
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 12.7 min
- P95 duration: 13.3 min

### Check actions
- Total runs: 57
- Failure rate: 5.3%
- Flaky rate: 14.0%
- Avg duration: 1.2 min
- P95 duration: 6.0 min

### Cherry-pick on PR merge
- Total runs: 13
- Failure rate: 15.4%
- Flaky rate: 0.0%
- Avg duration: 0.3 min
- P95 duration: 1.2 min

### Configured Graph Update: go_modules in /hack/api-group-resources, /hack/controller-gen #1242011540
- Total runs: 1
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 0.5 min
- P95 duration: 0.5 min

### Conformance tests
- Total runs: 68
- Failure rate: 55.9%
- Flaky rate: 60.3%
- Avg duration: 17.7 min
- P95 duration: 41.4 min

### FOSSA
- Total runs: 9
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 1.3 min
- P95 duration: 2.1 min

### Issue and PR comment commands
- Total runs: 84
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 1.6 min
- P95 duration: 6.0 min

### Lint
- Total runs: 57
- Failure rate: 29.8%
- Flaky rate: 14.0%
- Avg duration: 6.2 min
- P95 duration: 12.1 min

### Load Tests
- Total runs: 37
- Failure rate: 54.1%
- Flaky rate: 13.5%
- Avg duration: 18.4 min
- P95 duration: 45.9 min

### PR Labelling
- Total runs: 64
- Failure rate: 1.6%
- Flaky rate: 0.0%
- Avg duration: 1.3 min
- P95 duration: 4.2 min

### PR update
- Total runs: 11
- Failure rate: 36.4%
- Flaky rate: 0.0%
- Avg duration: 0.4 min
- P95 duration: 1.1 min

### Publish images
- Total runs: 12
- Failure rate: 100.0%
- Flaky rate: 0.0%
- Avg duration: 20.8 min
- P95 duration: 25.3 min

### Retry workflow
- Total runs: 176
- Failure rate: 72.2%
- Flaky rate: 0.0%
- Avg duration: 0.3 min
- P95 duration: 2.0 min

### Scorecards supply-chain security
- Total runs: 9
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 1.1 min
- P95 duration: 1.7 min

### Sonarcloud workflow
- Total runs: 11
- Failure rate: 0.0%
- Flaky rate: 0.0%
- Avg duration: 3.3 min
- P95 duration: 4.0 min

### Tests
- Total runs: 69
- Failure rate: 33.3%
- Flaky rate: 11.6%
- Avg duration: 11.4 min
- P95 duration: 25.5 min

### Verify codegen
- Total runs: 57
- Failure rate: 36.8%
- Flaky rate: 14.0%
- Avg duration: 10.7 min
- P95 duration: 25.9 min

### check-milestone-label
- Total runs: 65
- Failure rate: 32.3%
- Flaky rate: 18.5%
- Avg duration: 1.2 min
- P95 duration: 5.3 min

### cli
- Total runs: 57
- Failure rate: 31.6%
- Flaky rate: 14.0%
- Avg duration: 7.4 min
- P95 duration: 13.9 min

### helm-test
- Total runs: 19
- Failure rate: 5.3%
- Flaky rate: 15.8%
- Avg duration: 0.9 min
- P95 duration: 7.7 min

## Top 5 Most Flaky Workflows (All)

1. **Conformance tests** - 60.3% flaky (68 runs)
2. **check-milestone-label** - 18.5% flaky (65 runs)
3. **helm-test** - 15.8% flaky (19 runs)
4. **Build devcontainer** - 14.0% flaky (57 runs)
5. **Check actions** - 14.0% flaky (57 runs)

## Top 5 Slowest Workflows by P95 (All)

1. **Load Tests** - 45.9 min P95 (37 runs)
2. **Conformance tests** - 41.4 min P95 (68 runs)
3. **Verify codegen** - 25.9 min P95 (57 runs)
4. **Tests** - 25.5 min P95 (69 runs)
5. **Publish images** - 25.3 min P95 (12 runs)
