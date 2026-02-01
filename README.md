# Order Preference – Full-Stack App

Full-stack application with **Go** backend, **React** frontend, and **PostgreSQL**. JWT auth, order preferences (IN_STORE, DELIVERY, CURBSIDE), conditional validation, and **AI order summary** (OpenAI or Google Gemini).

**Detailed documentation:** [docs/PROJECT_DOCUMENTATION.md](docs/PROJECT_DOCUMENTATION.md) — project structure, backend/frontend flow, and requirements.

---

## Project features

- **JWT authentication** — Login with email/password; protected routes; token in `Authorization: Bearer` header.
- **Order preferences** — IN_STORE, DELIVERY, CURBSIDE with conditional fields (address and pickup time for DELIVERY/CURBSIDE).
- **Validation** — Backend and frontend: preference enum, required address/pickup time when applicable, pickup time must be in the future.
- **Order CRUD** — Create, read, update order; orders scoped to the authenticated user.
- **AI order summary** — On the Summary page, “Generate AI summary” calls the backend; backend uses **OpenAI** (when `OPENAI_API_KEY` is set) or **Google Gemini** (when `GEMINI_API_KEY` is set) to generate a short summary from order details (order number, preference, address, pickup time, creation date). If no key is set or the API fails, a plain fallback summary is returned so the app never breaks.
- **Graceful AI fallback** — No API key or API error → fallback summary; tests run without keys (mockable).

---

## How to run (for someone who cloned the project)

You can run everything with **Docker** (easiest) or run **PostgreSQL + backend + frontend** locally.

### Option 1: Run with Docker (recommended)

1. **Clone the repo** (if you haven’t already):

   ```bash
   git clone <repo-url>
   cd Zeshan-Weel-Challenge
   ```

2. **Create a `.env` file** in the project root (same folder as `docker-compose.yml`).  
   Copy from `.env.example` and set at least `DB_PASSWORD` (used by Postgres and backend):

   ```bash
   # Example .env
   DB_PASSWORD=your-secret-password
   ```

   Optional: add `OPENAI_API_KEY` or `GEMINI_API_KEY` for AI order summary; Docker Compose passes them to the backend (see [Environment variables](#environment-variables)).

3. **Start all services**:

   ```bash
   docker compose up --build
   ```

4. **Open the app**

   - Frontend: http://localhost:5173
   - Backend API: http://localhost:8080

5. **Log in** with the seeded user:
   | Field | Value |
   |----------|--------------------|
   | Email | `user@weel.com` |
   | Password | `password` |

Migrations and the seed user run when the backend starts. Wait until all three containers (postgres, backend, frontend) are up, then use the app.

### Option 2: Run locally (PostgreSQL in Docker, backend + frontend on your machine)

1. **Start only PostgreSQL**:

   ```bash
   docker compose up -d postgres
   ```

   Wait a few seconds for the DB to be ready.

2. **Create `.env`** in the project root with DB settings for local backend:

   ```bash
   DB_PASSWORD=your-password
   DB_HOST=localhost
   DB_PORT=5433
   DB_USER=postgres
   DB_NAME=postgres
   ```

   (Docker exposes Postgres on port **5433** on the host.)

3. **Run the backend** (from repo root):

   ```bash
   npm run dev:backend
   ```

   Or:

   ```bash
   cd backend
   go mod tidy
   go run ./cmd/server
   ```

   Migrations and seed user run on startup.

4. **Run the frontend** (in another terminal):

   ```bash
   npm run dev:frontend
   ```

   Or:

   ```bash
   cd frontend
   npm install
   npm run dev
   ```

5. **Open** http://localhost:5173 (or the port Vite shows) and log in with `user@weel.com` / `password`.

6. **Optional:** Add `OPENAI_API_KEY` or `GEMINI_API_KEY` to `.env` for AI order summary on the Summary page.

---

## Environment variables

Create a **`.env`** file in the **project root** if you want to override defaults. Docker Compose and the backend (when run locally) load it.

| Variable           | Where used            | Default / note                                                        |
| ------------------ | --------------------- | --------------------------------------------------------------------- |
| **DB_PASSWORD**    | postgres, backend     | Required in Docker; set in `.env`.                                    |
| **DB_HOST**        | backend               | `postgres` (Docker) or `localhost` (local run).                       |
| **DB_PORT**        | backend               | `5432` (inside Docker) or `5433` (host when only postgres in Docker). |
| **DB_USER**        | backend               | `postgres` (match POSTGRES_USER).                                     |
| **DB_NAME**        | backend               | `postgres` (match POSTGRES_DB).                                       |
| **JWT_SECRET**     | backend               | `dev-secret-change-in-production` — change in production.             |
| **VITE_API_URL**   | frontend (build time) | `http://localhost:8080` — set in Dockerfile for production build.     |
| **OPENAI_API_KEY** | backend (optional)    | If set, AI order summary uses OpenAI.                                 |
| **GEMINI_API_KEY** | backend (optional)    | If set (and no OpenAI key), AI order summary uses Google Gemini.      |

See `.env.example` for a template.

---

## Running tests

**Backend** (requires PostgreSQL; use same DB\_\* as your local backend or let it use defaults):

```bash
npm run test:backend
```

or

```bash
cd backend
go test ./...
```

Tests include:

- Login success/failure, auth guard, order validation, **order summary requires auth**, **order summary returns fallback when no AI key**.

**Frontend**:

```bash
npm run test:frontend
```

or

```bash
cd frontend
npm install
npm test
```

Tests include:

- Summary page loads order data, **Generate AI summary** calls `getOrderSummary` and shows the returned summary and “Generated with AI”.

**Run all tests** (from repo root):

```bash
npm test
```

---

## API endpoints

| Method | Path                | Auth | Description                  |
| ------ | ------------------- | ---- | ---------------------------- |
| POST   | /auth/login         | No   | Login, returns JWT           |
| GET    | /me                 | Yes  | Current user                 |
| POST   | /orders             | Yes  | Create order                 |
| GET    | /orders/:id         | Yes  | Get order                    |
| PUT    | /orders/:id         | Yes  | Update order                 |
| GET    | /orders/:id/summary | Yes  | AI or fallback order summary |

Orders are scoped to the authenticated user. Summary uses OpenAI or Gemini when the corresponding API key is set; otherwise returns a plain fallback.

---

## Architecture overview

- **Backend (Go):** `net/http` mux, `database/sql` + PostgreSQL, JWT in `Authorization: Bearer <token>`, golang-migrate for schema and seed. AI summary via OpenAI or Gemini (net/http only; no extra SDKs).
- **Frontend (React):** Vite + TypeScript, React Router, React Hook Form + Zod, auth and order state in context + localStorage.
- **Docker:** `postgres` with healthcheck; `backend` runs migrations on startup then serves API; `frontend` build served by nginx.

```
Browser → frontend (nginx :5173)  →  API calls → backend (:8080) → PostgreSQL
                                         ↓
                                    OpenAI / Gemini (when keys set)
```

---

## Design decisions

- **No ORM:** Plain SQL and `database/sql`; migrations define schema and seed user.
- **JWT in header:** Stateless auth; frontend stores token in localStorage and sends `Authorization: Bearer <token>`.
- **Conditional order fields:** DELIVERY/CURBSIDE require address and pickup time; IN_STORE does not. Enforced in backend and frontend.
- **pickup_time in future:** Enforced in backend and frontend Zod schema.
- **AI summary backend-proxied:** API keys stay on the server; frontend only calls `GET /orders/:id/summary`. No key or API failure → fallback summary.

---

## Tradeoffs and improvements

- **localStorage for token:** Simple for demo; for production consider HttpOnly cookies or short-lived tokens + refresh.
- **Single seed user:** Enough for demo; production would add registration and user management.
- **CORS `*`:** Fine for local/dev; production should restrict origin.
- **Frontend build in Docker:** Uses `VITE_API_URL` at build time; for production deploy, set the correct API URL.

Possible future improvements: user registration, refresh tokens, rate limiting, E2E tests (e.g. Playwright).
