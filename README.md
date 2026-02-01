# Order Preference – Full-Stack App

Full-stack application with **Go** backend, **React** frontend, and **PostgreSQL**. JWT auth, order preferences (IN_STORE, DELIVERY, CURBSIDE), conditional validation, and **AI order summary** (OpenAI or Google Gemini).

---

## Project features

- **JWT authentication** — Login with email/password; protected routes; token in `Authorization: Bearer` header.
- **Order preferences** — IN_STORE, DELIVERY, CURBSIDE with conditional fields (address and pickup time for DELIVERY/CURBSIDE).
- **Validation** — Backend and frontend: preference enum, required address/pickup time when applicable, pickup time must be in the future.
- **Order CRUD** — Create, read, update order; orders scoped to the authenticated user.
- **AI order summary** — On the Summary page, “Generate AI summary” calls the backend; backend uses **OpenAI** (when `OPENAI_API_KEY` is set) or **Google Gemini** (when `GEMINI_API_KEY` is set) to generate a short summary from order details. If no key is set or the API fails, a plain fallback summary is returned.
- **Graceful AI fallback** — No API key or API error → fallback summary; tests run without keys (mockable).

---

## Setup

Use the **npm scripts** in `package.json` from the project root.

### Option 1: Run with Docker (recommended)

1. Create a `.env` file in the project root (copy from `.env.example`). Set at least `DB_PASSWORD`. Optionally add `OPENAI_API_KEY` or `GEMINI_API_KEY` for AI order summary.

2. Start all services:

   ```bash
   npm run up
   ```

3. Open the app:
   - Frontend: http://localhost:5173  
   - Backend API: http://localhost:8080  

4. Log in with the seeded user: **Email** `user@weel.com` / **Password** `password`

Migrations and the seed user run when the backend starts.

### Option 2: Run locally (PostgreSQL in Docker, backend + frontend on your machine)

1. Start PostgreSQL:

   ```bash
   npm run dev:db
   ```

   Wait a few seconds for the DB to be ready.

2. Create `.env` in the project root with DB settings (e.g. `DB_PASSWORD`, `DB_HOST=localhost`, `DB_PORT=5433`, `DB_USER=postgres`, `DB_NAME=postgres`).

3. Run backend and frontend (in one command):

   ```bash
   npm run dev
   ```

   Or in separate terminals: `npm run dev:backend` and `npm run dev:frontend`.

4. Open http://localhost:5173 and log in with `user@weel.com` / `password`.

---

## Available scripts (package.json)

| Script | Description |
|--------|-------------|
| `npm run up` | Start all services with Docker (postgres, backend, frontend). |
| `npm run down` | Stop Docker services. |
| `npm run dev:db` | Start only PostgreSQL in Docker. |
| `npm run dev:backend` | Run the Go backend locally. |
| `npm run dev:frontend` | Run the React frontend locally. |
| `npm run dev` | Run backend and frontend together (use after `dev:db`). |
| `npm run test` | Run backend and frontend tests. |
| `npm run test:backend` | Run Go tests only. |
| `npm run test:frontend` | Run frontend tests only. |
| `npm run migrate` | Run database migrations. |

Detailed documentation: [docs/PROJECT_DOCUMENTATION.md](docs/PROJECT_DOCUMENTATION.md)
