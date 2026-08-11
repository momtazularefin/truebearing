# TrueBearing

> Production LLM evaluation, drift monitoring, and regression alerting platform in Go.

<!-- Badges will be added after GitHub repo setup -->
<!-- ![CI](https://github.com/momtazularefin/truebearing/actions/workflows/ci.yml/badge.svg) -->
<!-- ![License](https://img.shields.io/github/license/momtazularefin/truebearing) -->

## Overview

TrueBearing is a multi-tenant SaaS platform that automates LLM output evaluation. Teams upload datasets and prompt templates, run evaluation jobs using LLM-as-judge and code-based checks, and receive alerts when model quality drifts or regresses.

**Status:** 🏗️ Under active development (M0 scaffold).

## Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Database | PostgreSQL 16 + TimescaleDB |
| Queue | Redis 7 (Streams) |
| Router | chi/v5 |
| Infra | k3s on Hetzner CX22, Terraform, Grafana |
| CI | GitHub Actions |

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- (Optional) Go 1.23+ for local development

### Run with Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

The API will be available at `http://localhost:8080`.

### Health Check

```bash
curl http://localhost:8080/healthz
```

### Local Development

```bash
# Start dependencies only
docker compose up postgres redis -d

# Run the API locally
cp .env.example .env
go run ./cmd/server
```

## Architecture

```
cmd/server/          Entry point, graceful shutdown
internal/
  api/               Chi router, health checks, middleware
  config/            Environment-based configuration
  database/          pgxpool setup, embedded SQL migrations
  queue/             Redis client setup
```

Full architecture diagram will be added in a later milestone.

## API Endpoints

| Method | Path | Status |
|--------|------|--------|
| GET | `/healthz` | ✅ Live |
| GET | `/readyz` | ✅ Live |
| * | `/api/v1/tenants` | 🚧 Stub |
| * | `/api/v1/datasets` | 🚧 Stub |
| * | `/api/v1/prompts` | 🚧 Stub |
| * | `/api/v1/runs` | 🚧 Stub |

## License

[MIT](LICENSE)
