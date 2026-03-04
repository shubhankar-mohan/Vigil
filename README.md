# Vigil

A Dead Man Switch monitoring service. Detects when expected signals — Prometheus metrics or Loki logs — stop arriving. If a cron job silently stops running, a service stops emitting metrics, or a periodic log line disappears, Vigil catches it.

Traditional monitoring catches errors. Vigil catches **silence**.

## Why

Cron jobs and periodic processes can fail silently — they simply don't run, producing no logs and no errors. Log-based alerts and error alerts can't catch the **absence** of a signal. Vigil watches for expected signals and raises an alert when they go missing.

**Use Vigil when:**
- A cron job should run every hour but you'd only notice days later if it stopped
- A nightly report should complete between 2am-4am
- A data sync should happen at regular intervals but the schedule isn't fixed
- You want to auto-discover recurring patterns in your logs and alert if they disappear

## How It Works

```
Your Apps                          Vigil (:8080)
  push logs ──────► Loki ◄──────── queries Loki (LogQL)
  /metrics  ──────► Prometheus ◄── queries Prometheus (PromQL)
                                      │
                                      ▼
                                   evaluates every 30s
                                   updates dms_switch_status
                                      │
                                      ▼
                    Prometheus ◄── scrapes /metrics
                                      │
                                      ▼
                    Grafana ────► alert: dms_switch_status == 0
                                      │
                                      ▼
                                   Slack / PagerDuty / Email
```

Vigil doesn't send alerts directly. It exposes `dms_switch_status` as a Prometheus metric. Your existing Grafana alerting handles routing, silencing, and escalation — where you already manage it.

## Quick Start

### 1. Add Vigil to your docker-compose

Add this to your existing monitoring `docker-compose.yml`:

```yaml
  vigil:
    image: vigil:latest
    ports:
      - "8080:8080"
    volumes:
      - vigil-data:/data
    environment:
      - PROMETHEUS_URL=http://prometheus:9090
      - LOKI_URL=http://loki:3100
      # If behind basic auth:
      # - PROMETHEUS_USER=admin
      # - PROMETHEUS_PASSWORD=secret
      # - LOKI_USER=admin
      # - LOKI_PASSWORD=secret
      # Optional - for Grafana annotations on state changes:
      # - GRAFANA_URL=http://grafana:3000
      # - GRAFANA_API_TOKEN=your-token
    depends_on:
      - prometheus
      - loki

volumes:
  vigil-data:
```

### 2. Add Vigil scrape target to Prometheus

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  # ... your existing scrape configs ...

  - job_name: 'vigil'
    static_configs:
      - targets: ['vigil:8080']
    scrape_interval: 15s
```

### 3. Create a Grafana alert rule

One alert rule covers **all** switches:

```
Query:        dms_switch_status == 0
For:          0s   (Vigil already applies grace periods)
Labels:       name = {{ $labels.name }}, mode = {{ $labels.mode }}
Annotation:   Switch {{ $labels.name }} is DOWN
```

Route this to your existing contact points (Slack, PagerDuty, email, etc.).

### 4. Open the UI and create switches

Go to `http://localhost:8080` and create your first switch.

## Configuration

All configuration via environment variables:

| Variable | Default | Description |
|---|---|---|
| `PROMETHEUS_URL` | `http://prometheus:9090` | Prometheus server URL |
| `PROMETHEUS_USER` | _(empty)_ | Basic auth username for Prometheus |
| `PROMETHEUS_PASSWORD` | _(empty)_ | Basic auth password for Prometheus |
| `LOKI_URL` | `http://loki:3100` | Loki server URL |
| `LOKI_USER` | _(empty)_ | Basic auth username for Loki |
| `LOKI_PASSWORD` | _(empty)_ | Basic auth password for Loki |
| `GRAFANA_URL` | _(empty)_ | Grafana URL (optional, for annotations) |
| `GRAFANA_API_TOKEN` | _(empty)_ | Grafana API token (optional) |
| `EVAL_INTERVAL` | `30s` | How often to evaluate all switches |
| `LISTEN_ADDR` | `:8080` | HTTP server listen address |
| `DB_PATH` | `/data/vigil.db` | SQLite database file path |

## Loki Endpoints Required

If Loki is behind a reverse proxy (nginx), Vigil needs these endpoints exposed:

```nginx
# Required
location /loki/api/v1/query { proxy_pass http://loki:3100; }
location /loki/api/v1/query_range { proxy_pass http://loki:3100; }

# Optional (for auto-discovery)
location /loki/api/v1/patterns { proxy_pass http://loki:3100; }
```

## Detection Modes

### Frequency Mode

For signals expected at a known interval. Configure:
- **Interval**: expected every N seconds (e.g., 3600 = every hour)
- **Grace period**: extra time before alerting (e.g., 300 = 5 min)
- **Time window** (optional): only monitor during specific hours (e.g., 09:00-17:00)

**Prometheus example** — watch a gauge that stores a unix timestamp:
```
Query:    cron_last_run_timestamp{cron_name="sync_awb"}
Mode:     frequency
Interval: 3600   (every 1 hour)
Grace:    300    (5 min grace)
```

**Loki example** — watch for a specific log line:
```
Query:    {job="diagonAlleyBE_prod"} |= "[CRON] sync_awb completed"
Mode:     frequency
Interval: 3600
Grace:    300
```

### Irregularity Mode

For signals that occur at irregular but roughly predictable intervals. Vigil learns the pattern from historical data and alerts when the signal is overdue.

- **Min samples**: how many data points to collect before activating (default: 4)
- **Tolerance multiplier**: how many times the median interval before alerting (default: 2x)

```
Query:         {job="myapp"} |= "batch processing complete"
Mode:          irregularity
Min samples:   4
Tolerance:     2.0
```

Vigil computes the **median interval** between occurrences and alerts if `elapsed > tolerance * median`.

## Switch States

```
    NEW ──── first signal ──── UP
                                │
                      signal    │  no signal within
                      arrives   │  expected window
                        │       │
                        │       ▼
                        └──── GRACE
                                │
                      signal    │  grace period
                      arrives   │  expires
                        │       │
                        ▼       ▼
                       UP     DOWN ── signal arrives ── UP

    LEARNING:  Irregularity mode — collecting initial data points.
    PAUSED:    Manually paused. No evaluation.
```

## Exposed Prometheus Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `dms_switch_status` | gauge | name, mode, signal | 1 = healthy, 0 = violated |
| `dms_last_signal_timestamp` | gauge | name | Unix timestamp of last signal |
| `dms_expected_at_timestamp` | gauge | name | Unix timestamp of next expected signal |
| `dms_state_duration_seconds` | gauge | name, state | Seconds in current state |
| `dms_eval_total` | counter | name, result | Evaluation count (pass/fail) |

## API

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/metrics` | Prometheus metrics |
| GET | `/api/dashboard` | Dashboard summary |
| GET | `/api/switches` | List all switches |
| POST | `/api/switches` | Create switch |
| GET | `/api/switches/:id` | Get switch detail |
| PUT | `/api/switches/:id` | Update switch |
| DELETE | `/api/switches/:id` | Delete switch |
| POST | `/api/switches/:id/pause` | Pause evaluation |
| POST | `/api/switches/:id/resume` | Resume evaluation |
| POST | `/api/switches/test-query` | Test a PromQL/LogQL query |
| GET | `/api/switches/:id/history` | Evaluation history |
| GET | `/api/auto-rules` | List auto-discovery rules |
| POST | `/api/auto-rules` | Create auto-discovery rule |
| PUT | `/api/auto-rules/:id` | Update rule |
| DELETE | `/api/auto-rules/:id` | Delete rule |

### Test Query

Before creating a switch, test your query:

```bash
# Prometheus
curl -X POST http://localhost:8080/api/switches/test-query \
  -H "Content-Type: application/json" \
  -d '{"signal":"prometheus","query":"node_time_seconds{job=\"nodeexporter\"}"}'

# Loki
curl -X POST http://localhost:8080/api/switches/test-query \
  -H "Content-Type: application/json" \
  -d '{"signal":"loki","query":"{job=\"diagonAlleyBE_prod\"} |= \"completed\""}'
```

### Create Switch

```bash
curl -X POST http://localhost:8080/api/switches \
  -H "Content-Type: application/json" \
  -d '{
    "name": "sync_awb_cron",
    "signal": "loki",
    "query": "{job=\"diagonAlleyBE_prod\"} |= \"[CRON] sync_awb completed\"",
    "mode": "frequency",
    "interval_seconds": 3600,
    "grace_seconds": 300
  }'
```

## Auto-Discovery

Vigil can automatically scan Loki for recurring log patterns and create switches for them.

```bash
curl -X POST http://localhost:8080/api/auto-rules \
  -H "Content-Type: application/json" \
  -d '{
    "loki_selector": "{job=\"diagonAlleyBE_prod\"}",
    "pattern": "[CRON]*",
    "min_samples": 4,
    "tolerance_multiplier": 2.0
  }'
```

This scans Loki every hour for patterns matching `[CRON]*` in the specified job, and auto-creates irregularity-mode switches for any recurring patterns found.

## Docker Build

```bash
# Build image
docker build -t vigil .

# Run standalone
docker run -d \
  --name vigil \
  -p 8080:8080 \
  -v vigil-data:/data \
  -e PROMETHEUS_URL=http://prometheus:9090 \
  -e LOKI_URL=http://loki:3100 \
  vigil
```

## Development

```bash
# Prerequisites: Go 1.23+, Node 18+

# Run backend
DB_PATH=./vigil.db \
PROMETHEUS_URL=https://metrics.example.com \
PROMETHEUS_USER=admin \
PROMETHEUS_PASSWORD=secret \
LOKI_URL=https://logs.example.com \
LOKI_USER=admin \
LOKI_PASSWORD=secret \
LISTEN_ADDR=:8181 \
go run ./cmd/vigil

# Run frontend (separate terminal)
cd web
npm install
npm run dev
# Opens at http://localhost:5173, proxies API to :8181
```

## What Touches What

| Component | Changes Needed |
|---|---|
| Your apps | **NONE** (zero code changes) |
| Loki | Expose read API endpoints if behind proxy |
| Prometheus | Add 3-line scrape config for Vigil |
| Grafana | Add 1 alert rule: `dms_switch_status == 0` |
| docker-compose | Add Vigil service block |

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go |
| HTTP Router | chi |
| Database | SQLite (persisted via Docker volume) |
| Metrics | prometheus/client_golang |
| Frontend | React + TypeScript + Vite |
| Docker | Multi-stage build (Node + Go + Alpine) |
