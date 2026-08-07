# Service Scheduler

A full-stack appointment-scheduling web application for automotive service
dealerships. Customers pick a dealership, choose a service, and book a time;
the system automatically assigns a qualified, available technician and a free
service bay — handling working schedules, breaks, overlapping appointments, and
concurrent bookings.

Built as a coding challenge submission: a **Go/Gin** REST + WebSocket backend, a
**React/TypeScript/Vite** frontend, and **PostgreSQL 17**, all runnable with a
single `docker compose up`.

---

## Features

- **Multi-dealership quick booking flow** — dealership → service type →
  preferred/any technician → time slot
- **Automatic resource assignment** — the backend picks a technician who is
  qualified for the service, working during the requested window (weekly
  schedule + breaks), and a service bay that is free
- **"Other" service type** — customers can request an ad-hoc service with a
  custom duration in minutes
- **Concurrency-safe** — bookings are created inside transactions with row
  locks (`SELECT ... FOR UPDATE`) and re-checked for overlap before commit, so
  two simultaneous customers cannot double-book the same technician or bay
- **Real-time updates** — the backend broadcasts `appointment.created` events
  over WebSocket (topic = `dealership_id`); the frontend falls back to 5-second
  polling if the socket is unavailable
- **Input validation** — SQL-injection-safe validation with non-blocking hints
  in the UI; strict validation on submit
- **Seed data** — idempotent migrations ship two dealerships, service types,
  bays, technicians with qualifications and weekly schedules

---

## Tech Stack

| Layer      | Technology                                                        |
| ---------- | ----------------------------------------------------------------- |
| Backend    | Go 1.25, Gin, sqlx + lib/pq, gorilla/websocket, logrus, testify   |
| Frontend   | React 19, TypeScript, Vite 8, Tailwind CSS v4, react-toastify     |
| Database   | PostgreSQL 17                                                     |
| Infra      | Docker Compose, nginx (reverse proxy in the frontend container)   |
| Tooling    | Makefile, Render Blueprint (`render.yaml`)                        |

---

## Architecture

The frontend container ships a production build of the React app served by
nginx. nginx also **reverse-proxies `/api/*` and `/ws` to the backend** over the
Docker network, so the browser only ever talks to one origin (no CORS issues
in production):

```
Browser
   │   http://localhost:5173   (or a public URL via Render / a tunnel)
   ▼
nginx  (frontend container)
   │   /api/* and /ws  ─────────────────────────────┐
   │                                               ▼
   │                                        Go API  (backend container)
   └── serves the built React SPA                │
                                                 ▼
                                        PostgreSQL 17 (postgres container)
```

## Prerequisites

- **Docker** with **Docker Compose v2** (recommended path)
- Optional, for local (non-Docker) development:
  - **Go 1.25+**
  - **Node.js 22+** and npm

---

## Quick Start (Docker)

```bash
cp .env.example .env        # then set POSTGRES_USER and POSTGRES_PASSWORD
make up                     # = docker compose up -d --build
```

That's it. The backend's `entrypoint.sh` waits for Postgres, applies
`/migrations/*.sql` (idempotent, safe to re-run on every start), and launches
the API.

- Frontend: **http://localhost:5173**
- Backend: **http://localhost:8080** (e.g. `http://localhost:8080/api/service-types`)

Stop with `make down`. A full teardown (containers + images + volumes) is
`make clean`.

---

## Configuration

Copy `.env.example` to `.env` and adjust:

| Variable              | Default                      | Purpose                                    |
| --------------------- | ---------------------------- | ------------------------------------------ |
| `POSTGRES_DB`         | `appointment_scheduler`      | Database name                             |
| `POSTGRES_USER`       | — (set it)                   | Database user (used by compose + backend)  |
| `POSTGRES_PASSWORD`   | — (set it)                   | Database password                          |
| `POSTGRES_PORT`       | `5432`                       | Host port mapped to Postgres               |
| `VITE_API_URL`        | *(empty → same-origin `/`)*  | Overrides the frontend API base URL        |
| `VITE_WS_URL`         | *(empty → same-origin `/ws`)*| Overrides the frontend WebSocket URL       |
| `FRONTEND_ORIGIN`     | `http://localhost:5173`      | Backend CORS allowed origin                |

> Keep `VITE_API_URL` / `VITE_WS_URL` empty in production: the frontend then
> talks to the same origin it is served from, and nginx proxies `/api` + `/ws`
> to the backend. Setting them overrides that safe default.

---

## Building & Running Manually

### Backend

Run the whole stack in Docker (recommended — migrations are applied by the
container entrypoint):

```bash
make be-build                 # rebuild the backend image (re-runs migrations)
make be                       # tail backend logs
```

Run the API natively (for debugging):

```bash
cd service-scheduler-backend
go build ./...
# Start Postgres first (docker compose up -d postgres) and apply migrations,
# then run with the DB pointing at localhost:
POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=postgres \
POSTGRES_PASSWORD=postgres POSTGRES_DB=appointment_scheduler go run ./cmd/api
```

### Frontend

Production build / Docker:

```bash
make fe-build                 # rebuild the frontend image
```

Local development with hot reload (proxies `/api` and `/ws` to
`http://localhost:8080` via `vite.config.ts`):

```bash
cd service-scheduler-frontend
npm install
npm run dev                   # http://localhost:5173
```

Other useful commands: `npm run build` (type-check + bundle), `npm run lint`.

---

## Testing

The test suite validates the **core booking business logic**. The backend tests
are integration-style and need a reachable Postgres with the schema + seed data
loaded — the Docker stack provides exactly that.

```bash
make up                       # ensure Postgres is running
make be-test                  # loads .env, runs: cd service-scheduler-backend && go test ./...
```

Run them directly with:

```bash
cd service-scheduler-backend
POSTGRES_HOST=localhost POSTGRES_PORT=5432 POSTGRES_USER=postgres \
POSTGRES_PASSWORD=postgres POSTGRES_DB=appointment_scheduler go test ./...
```

What they cover:

- `TestBook_NoTechnician` — a booking request for a dealership/time where no
  technician is working is rejected (returns `ErrNoTechnician`).
- `TestBook_OtherServiceRequiresDuration` — an "Other" service booking without
  a positive custom duration is rejected before any DB call (pure unit test).

Frontend checks: `npm run build` (strict `tsc -b` + Vite bundle) and
`npm run lint` (oxlint). Manual end-to-end flow: open `http://localhost:5173`,
pick **Keyloop Demo Dealership**, choose a service and time, and submit a
booking.

---

## API

Base URL: `http://localhost:8080` (or same-origin `/api/...` through nginx).

| Method | Path                          | Purpose                                        |
| ------ | ----------------------------- | ---------------------------------------------- |
| GET    | `/api/dealerships`            | List dealerships                               |
| GET    | `/api/service-types`          | List service types + durations                 |
| GET    | `/api/technicians`            | List qualified technicians for a dealership/service (`dealership_id`, `service_type_id` query params) |
| POST   | `/api/availability`           | Lightweight availability check                 |
| POST   | `/api/quick-booking`          | Create a booking from user-friendly fields     |
| POST   | `/api/bookings`               | Create a booking from existing customer/vehicle IDs |
| GET    | `/ws?dealership_id=<uuid>`    | WebSocket: real-time booking events per dealership |

Example — quick booking:

```json
{
  "customer_first_name": "Jane",
  "customer_last_name": "Doe",
  "customer_email": "jane@example.com",
  "customer_phone": "+49 151 12345678",
  "vehicle_vin": "WVWZZZ1JZXW000001",
  "vehicle_make": "Volkswagen",
  "vehicle_model": "Golf",
  "vehicle_year": 2020,
  "dealership_id": "22222222-2222-2222-2222-222222222222",
  "service_type": "Oil Change",
  "preferred_technician_id": "",
  "desired_start": "2026-08-10T09:00:00+02:00"
}
```

Returns `201` with `{"appointment_id": "...", "message": "Booked"}`.

---

## Deployment

### Render (Blueprint)

`render.yaml` at the repo root deploys the whole stack on Render from the
existing Dockerfiles: a **managed Postgres**, a **private backend service**, and
a **public frontend service**. The frontend nginx proxies `/api` + `/ws` to the
backend over Render's private network, so no CORS or extra env vars are needed.
Push the repo to GitHub and create a **Blueprint** from it; set
`FRONTEND_ORIGIN` in `render.yaml` to your real frontend URL.

### Home machine (free) via tunnel

Run `docker compose up -d` on an always-on machine and expose **only port
5173** with a tunnel; nginx forwards `/api` and `/ws` internally:

- **Cloudflare Tunnel** — `cloudflared` (passes WebSocket upgrades correctly).
- **Tailscale Funnel** — free stable `https://<machine>.<tailnet>.ts.net` URL;
  note that Funnel does not pass WebSocket upgrade requests, so the app
  automatically uses its polling fallback there.

---

## AI Collaboration Narrative

This project was built through an AI-assisted workflow that combined three
tools. This section explains how the AI was guided, how its output was
verified and refined, and how final code quality was ensured.

### High-level strategy for guiding the AI

- **Design the solution logic with ChatGPT, then implement in small slices.**
  The overall problem — how to assign technicians and bays without
  double-booking, handle working schedules, and push real-time updates — was
  discussed first as an architecture conversation (transaction + row locks,
  overlap checks, a WebSocket hub keyed by dealership). Only after the shape of
  the solution was agreed on did coding begin.
- **Keep the AI on a tight leash with precise context.** Prompts included exact
  file paths, the relevant code snippets, the stack (Go/Gin/sqlx, React/Vite),
  and the expected behavior. Changes were requested as focused edits ("fix the
  technicians endpoint to accept a service_type UUID", "split the form logic
  into a custom hook") rather than open-ended rewrites, which kept diffs small
  and reviewable.
- **Give acceptance criteria, not vague wishes.** For example: "service-types
  must still return 200 through nginx at `:5173`", or "the form must not block
  typing but must validate on submit". This made the AI's output directly
  checkable.
- **Ask for one thing at a time.** Backend work, then containerization, then
  frontend polish, then deployment — so a failure at any step was easy to
  isolate.

### Process for verifying and refining the AI's output

Every AI-generated change was treated as a draft and verified before being
accepted:

- **Backend verification.** Each change was compiled with `go build ./...`, the
  migration replay in `entrypoint.sh` was confirmed idempotent, and every
  endpoint was exercised with `curl` — happy paths and error paths. When the
  technicians endpoint returned an empty list, root-cause analysis revealed the
  frontend sent a service-type **UUID** while the backend resolved it by
  **name**; the fix (accept a UUID via a duration lookup fallback) was verified
  by calling the endpoint with both name and UUID.
- **Frontend verification.** TypeScript strict build (`tsc -b`) and `npm run
  lint` were run after every change. UI behavior was confirmed in the browser
  (e.g. the regex input guard initially *blocked typing mid-way*; it was
  changed to a non-blocking validation hint with enforcement on submit).
- **End-to-end verification.** The whole stack was tested the way production
  runs it: requests through the nginx proxy on `:5173`, not just at the
  backend on `:8080`. This caught a broken WebSocket proxy and a mismatch in
  the nginx upstream hostname that local-only testing would have missed.
- **Refinement loops.** Bugs found during verification (an unused import that
  broke the test build, a deleted `nginx.conf` that broke `/api`, stale browser
  caches serving old bundles) were fed back to the AI as concrete reproduction
  reports until the behavior matched the requirement.

### How final quality was ensured

- **Tests for the core logic.** A Go test suite validates the booking service:
  rejection when no qualified/working technician exists, and validation that an
  "Other" service requires a positive duration. Tests are integration-style
  against the real seeded Postgres, plus a pure unit test that needs no DB.
- **Compile-time safety.** The frontend uses strict TypeScript (`tsc -b`) and a
  lint step (`oxlint`) so type errors and dead code fail the build instead of
  shipping.
- **Idempotent migrations.** Seed data uses `ON CONFLICT` upserts and runs on
  every backend start, so environments are reproducibly seeded without special
  setup steps.
- **Concurrency correctness.** Booking is transactional with row locks and an
  in-transaction overlap re-check, protecting the core promise: two users can
  never be assigned the same technician or bay at the same time.
- **Graceful degradation.** If the WebSocket cannot be established (e.g. behind
  a proxy that strips upgrade headers), the UI falls back to periodic polling,
  so the application never breaks in a hostile network environment.
- **Human review at every commit.** Each change was reviewed by a human before
  being committed, which caught the mistakes the automation couldn't (e.g. the
  AI deleting a needed nginx config while editing elsewhere).

### Tools used

- **ChatGPT** — early architectural discussion: solution logic, API design,
  and edge cases for the booking engine and WebSocket notification flow.
- **GitHub Copilot** — inline code completion and quick edits while coding.
- **opencode (an agentic coding assistant)** — implementing features,
  containerization, debugging, and deployment in tight feedback loops, with
  every suggestion verified via builds, tests, and live `curl`/browser checks.

---

## Makefile Reference

| Command      | Purpose                                               |
| ------------ | ----------------------------------------------------- |
| `make up`    | Start all services (`docker compose up -d`)           |
| `make down`  | Stop all services                                     |
| `make build` | Build all images                                      |
| `make ps`    | Show running services                                 |
| `make logs`  | Tail logs from all services                           |
| `make be`    | Tail backend logs                                     |
| `make be-build` | Rebuild the backend image (re-applies migrations)  |
| `make be-test` | Run backend Go tests (requires Postgres)           |
| `make fe`    | Tail frontend logs                                    |
| `make fe-build` | Rebuild the frontend image                         |
| `make fe-dev`   | Run the frontend Vite dev server                   |
| `make db`    | Open a `psql` shell in the Postgres container         |
| `make db-reset` | Recreate the database volume (drops data)          |
| `make clean` | Remove containers, images and volumes                 |
