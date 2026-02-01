# Project Documentation: Order Preference Full-Stack App

This document describes the project structure, how each part works from backend to frontend, and how the implementation fulfills the original requirements.

---

## 1. Repository Structure

```
Zeshan-Weel-Challenge/
├── backend/                    # Go API server
│   ├── cmd/
│   │   ├── server/main.go      # HTTP server entrypoint, routes, migrations on startup
│   │   ├── migrate/main.go     # Standalone migrate up/down (loads .env)
│   │   └── migrate-create/     # Creates new migration file pair (e.g. 000002_name.up/down.sql)
│   ├── internal/
│   │   ├── db/db.go            # PostgreSQL connection, RunMigrations, RunMigrationsDown, SeedTestUser
│   │   ├── handler/            # HTTP handlers
│   │   │   ├── handler.go      # Handler struct (db, jwt secret)
│   │   │   ├── auth.go         # POST /auth/login
│   │   │   ├── me.go           # GET /me
│   │   │   ├── orders.go       # POST/GET/PUT /orders
│   │   │   ├── summary.go      # GET /orders/{id}/summary (AI order summary, OpenAI/Gemini or fallback)
│   │   │   └── handler_test.go # Backend tests (login, auth guard, order validation, order summary)
│   │   └── middleware/
│   │       ├── auth.go         # JWT RequireAuth, Claims, UserIDFrom
│   │       └── cors.go         # CORS for frontend
│   ├── migrations/
│   │   ├── 000001_init.up.sql  # users, orders tables + seed user
│   │   └── 000001_init.down.sql
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/                   # React SPA
│   ├── src/
│   │   ├── api/client.ts       # login, me, getOrders, createOrder, getOrder, updateOrder, getOrderSummary (fetch + token)
│   │   ├── components/Layout.tsx   # Header (user, Logout), footer, Outlet for pages
│   │   ├── context/AuthContext.tsx  # token/user state, setToken, signOut, me() on mount
│   │   ├── pages/
│   │   │   ├── Login.tsx        # Email/password form, redirect on success to /
│   │   │   └── Preference.tsx   # Two-step: (1) Set preference form, (2) Delivery details & AI order summary
│   │   ├── App.tsx             # Routes, Layout, ProtectedRoute (redirect to /login if not authenticated)
│   │   ├── main.tsx            # BrowserRouter, AuthProvider, App
│   │   ├── index.css           # Global styles, variables, form/button/card classes
│   │   └── test/               # Vitest setup, App/Login/Preference/Summary tests
│   ├── index.html
│   ├── vite.config.ts         # Proxy /auth, /me, /orders to backend; Vitest config
│   ├── nginx.conf             # try_files for SPA (no 404 on /login refresh)
│   ├── Dockerfile             # Build with VITE_API_URL, nginx serve
│   └── package.json
├── docker-compose.yml         # postgres, backend, frontend
├── package.json              # Root scripts: dev:db, dev:backend, dev:frontend, migrate, test, etc.
├── .env / .env.example
├── README.md
└── docs/
    └── PROJECT_DOCUMENTATION.md  # This file
```

---

## 2. Backend: Structure and Flow

### 2.1 Stack and Requirements Fulfillment

| Requirement                  | Implementation                                                                                                                                          |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Language & Stack**         | Go, `net/http` (no framework), PostgreSQL, JWT, `database/sql` (no ORM), golang-migrate                                                                 |
| **REST API, JSON**           | Handlers return JSON; request bodies decoded with `json.Decoder`; errors as `{"error":"..."}`                                                           |
| **JWT authentication**       | Login returns JWT; protected routes use `Authorization: Bearer <token>`; middleware parses token and sets `user_id` in context                          |
| **Seed test user**           | Migration `000001_init.up.sql` inserts one user; `db.SeedTestUser()` on server startup ensures `user@weel.com` / `password` works (Go-generated bcrypt) |
| **Validation on all inputs** | Login: email/password non-empty; orders: preference enum, conditional address/pickup_time, pickup_time in future                                        |
| **Docker-ready**             | Backend Dockerfile multi-stage build; compose runs migrations on startup via `main.go`                                                                  |

### 2.2 Entrypoint and Request Flow

1. **Startup** (`cmd/server/main.go`):

   - Load `.env` (godotenv from repo root or `backend/`).
   - Run `db.RunMigrations()` (golang-migrate up).
   - Open DB pool, then `db.SeedTestUser(pool)`.
   - Create handler and auth middleware; register routes; wrap with CORS; listen on `:8080`.

2. **Routes**:

   - `POST /auth/login` → `h.Login` (no auth).
   - `GET /me`, `GET /orders`, `POST /orders`, `GET /orders/:id`, `PUT /orders/:id`, `GET /orders/:id/summary` → wrapped with `auth(...)` so JWT is required.

3. **Auth middleware** (`internal/middleware/auth.go`):

   - Reads `Authorization: Bearer <token>`.
   - Parses JWT with shared secret; puts `user_id` from claims into request context.
   - If missing/invalid token → `401 {"error":"unauthorized"}`.

4. **Handlers**:
   - **Login** (`auth.go`): Validates email/password, bcrypt compare, issues JWT with `user_id` and expiry.
   - **Me** (`me.go`): Reads `user_id` from context, fetches user from DB, returns `{id, email}`.
   - **Orders** (`orders.go`): All use `user_id` from context. CreateOrder/UpdateOrder validate preference (IN_STORE | DELIVERY | CURBSIDE), require address + future pickup_time for DELIVERY/CURBSIDE; GetOrder/UpdateOrder filter by `user_id` so users only see their own orders.
   - **Order summary** (`summary.go`): `GET /orders/{id}/summary` returns an AI-generated or fallback summary. Fetches order by id and user_id; builds order description (order number, preference, address, pickup time, creation date). Prompt: "Create the order summary for the customer in one or two complete sentences. Include order number, preference, address, pickup time. Use the following order details: " + orderDesc. Tries **OpenAI** first (when `OPENAI_API_KEY` set; model `gpt-4o-mini`, `max_tokens` 512); then **Gemini** (when `GEMINI_API_KEY` set; model `gemini-1.5-flash`, endpoint `.../generateContent`, request/response structs: `GeminiGenerateContentRequest`, `GeminiContentItem`, `GeminiPart`, `GeminiGenerationConfig`; `GeminiGenerateContentResponse`, `GeminiCandidate`, `GeminiContent`, `GeminiAPIError`; all response parts joined). No key or API failure → plain fallback. Response: `summary`, `source` ("ai" or "fallback"). Logs input prompt and output (with length). Uses `net/http` only; no external SDKs. Disabled gracefully and mockable for tests.

### 2.3 Database

- **Tables**: `users` (id, email, password_hash, created_at), `orders` (id, user_id, preference, address, pickup_time, created_at) with FK to users.
- **Migrations**: `backend/migrations/` with `000001_init.up.sql` / `.down.sql`. Standalone migrate: `npm run migrate` (up), `npm run migrate:down` (down), `npm run migrate:create -- <name>` (new migration pair).
- **Connection**: `internal/db/db.go` builds DSN from env (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME). Used by server and by `cmd/migrate`.

### 2.4 Backend Tests

- **File**: `internal/handler/handler_test.go`.
- **Login**: Success with `user@weel.com`/`password` returns 200 and token; wrong password returns 401.
- **Auth guard**: GET /me without token returns 401.
- **Order validation**: POST /orders with invalid body (e.g. past pickup_time, missing address for DELIVERY) returns 400.
- **Order summary requires auth**: Create an order, then GET /orders/{id}/summary **without** token → 401.
- **Order summary fallback when no AI key**: Create an order, then GET /orders/{id}/summary with auth; when no `OPENAI_API_KEY` or `GEMINI_API_KEY` is set, returns 200 with non-empty `summary` and `source: "fallback"`.
- Tests open real DB (env or defaults); skip if DB unavailable.

---

## 3. Frontend: Structure and Flow

### 3.1 Stack and Requirements Fulfillment

| Requirement                  | Implementation                                                                                                                                                                 |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **React (Vite), TypeScript** | Vite app in `frontend/`, TypeScript strict; build with `npm run build`                                                                                                         |
| **React Router**             | Routes in `App.tsx`: `/login`, `/` (Preference inside Layout); `/summary` redirects to `/`; `ProtectedRoute` redirects unauthenticated users to `/login`                      |
| **React Hook Form + Zod**    | Login and Preference use `useForm` with `zodResolver(schema)`; inline validation errors                                                                                        |
| **localStorage**             | Token stored in `localStorage`; `AuthContext` reads it on load and passes to API client; order id stored for loading/editing order                                             |
| **Pages**                    | Login (email/password, redirect on success to `/`); Preference (two-step: Set preference → Delivery details & AI order summary; Back to step 1; Logout in Layout header)     |
| **Auth-protected routes**    | `ProtectedRoute` checks `useAuth().user`; if not loaded or null, redirect to `/login`                                                                                          |
| **Redirect unauthenticated** | Same as above; plus API client sends `Authorization: Bearer <token>` on me/orders                                                                                             |

### 3.2 Data and Auth Flow

1. **Bootstrap** (`main.tsx`):

   - Renders `BrowserRouter` → `AuthProvider` → `App`.

2. **AuthContext** (`context/AuthContext.tsx`):

   - Holds `token` (from localStorage), `user`, `loading`, `setToken`, `signOut`.
   - On mount, if `token` exists, calls `me()`; on success sets `user`; on failure clears token/user.
   - `setToken` updates localStorage and state; `signOut` clears both.

3. **API client** (`api/client.ts`):

   - All requests use `BASE` from `VITE_API_URL` (or empty for proxy). Token read from localStorage; `Authorization: Bearer <token>` set for me/orders.
   - Exposes: `login`, `me`, `getOrders`, `createOrder`, `getOrder`, `updateOrder`, `getOrderSummary(orderId)` (typed with `OrderPreference`, `Order`).

4. **Layout** (`components/Layout.tsx`):

   - Wraps all non-login content via `<Outlet />`. Header: app name, user email, Logout button. Footer: author and email. Login page is full-width without header/footer.

5. **Pages**:

   - **Login**: Form with email/password; Zod schema (email format, non-empty password). On submit calls `login()`; on success `setToken`, `navigate('/')`. If already logged in, redirect to `/`.
   - **Preference** (single page, two steps):
     - **Step 1 — Set preference**: Select IN_STORE / DELIVERY / CURBSIDE; if DELIVERY/CURBSIDE, show address + datetime-local. Zod validates future pickup_time and required address. On submit: create or update order (via `orderId` in localStorage), then switch to step 2. If user already has an order, "Next" button also goes to step 2. On load: if `orderId` in localStorage, `getOrder(orderId)`; else `getOrders()` and use latest order to pre-fill form.
     - **Step 2 — Delivery details & summary**: Left column shows order details (order #, preference, address, pickup time, created). Right column: "Order summary" with "Generate AI summary" / "Regenerate" button calling `getOrderSummary(order.id)` (backend-proxied; OpenAI or Gemini when key set, else fallback). Displays summary text and "Generated with AI" when `source === "ai"`. "Back" returns to step 1. Logout is in the Layout header.

6. **Routing and guard** (`App.tsx`):
   - All routes under `Layout` except login: `<Route path="/" element={<Layout />}>` with `<Outlet />`.
   - `/` (index) → ProtectedRoute → Preference.
   - `/summary` → redirects to `/` (no separate Summary page).
   - `/login` → Login (no Layout).
   - `*` → redirect to `/`.
   - `ProtectedRoute`: if `loading` show loading UI; if `!user` redirect to `/login`; else render children.

### 3.3 Frontend Tests

- **App.test.tsx**: Unauthenticated visit to `/` redirects to login; heading “Sign in” is shown.
- **Login.test.tsx**: Success stores token; validation shows “Email required” when empty.
- **Preference.test.tsx**: Past datetime for DELIVERY is rejected; createOrder not called; me() mocked for AuthProvider.
- **Summary.test.tsx**: Tests the Preference step-2 / AI summary flow: (1) With mocked getOrder, summary reflects backend order data (order id, preference, address). (2) **AI summary**: With mocked getOrder and getOrderSummary (returns `{ summary: '...', source: 'ai' }`), click "Generate AI summary" → getOrderSummary(orderId) is called, summary text and "Generated with AI" are displayed.
- Vitest + jsdom; `@testing-library/react`, `jest-dom`, `user-event`; mocks for `api/client`.

---

## 4. User Flow (End-to-End)

1. **User opens app**  
   Frontend (Vite dev or nginx) serves SPA. React loads; AuthProvider reads token from localStorage; if present, calls GET /me with Bearer token; backend validates JWT, returns user; frontend sets user and shows app or login.

2. **Login**  
   User submits email/password. Frontend POST /auth/login with JSON body. Backend checks user in DB, bcrypt compare; if OK, returns JWT. Frontend stores token, calls me(), then navigates to `/` (Preference). Layout shows header (user email, Logout) and footer.

3. **Preference — Step 1 (Set preference)**  
   User sees "Set preference" form: IN_STORE / DELIVERY / CURBSIDE; if DELIVERY or CURBSIDE, address and pickup time (datetime-local) appear. Zod validates future pickup_time and required address. On load, if `orderId` in localStorage, frontend GET /orders/:id and pre-fills form; else GET /orders and uses latest order if any. On submit: POST /orders (new) or PUT /orders/:id (editing). Backend validates (enum, conditional fields, future pickup_time), inserts/updates order for `user_id` from JWT, returns order. Frontend stores order id in localStorage and switches to **step 2**. User can also click "Next" (if order already exists) to go to step 2 without re-saving.

4. **Preference — Step 2 (Delivery details & summary)**  
   Page shows "Delivery details & summary". Left: order details (order #, preference, address, pickup time, created) from current order. Right: "Order summary" with "Generate AI summary" / "Regenerate" button. Clicking it: frontend GET /orders/:id/summary with Bearer token. Backend builds order description, calls OpenAI (if OPENAI_API_KEY set) or Gemini (if GEMINI_API_KEY set); returns `{ summary, source }` (or fallback). Frontend displays summary and "Generated with AI" when source is ai. "Back" returns to step 1 (same page, step 1). Logout in header clears token and orderId, navigates to `/login`.

5. **Docker**  
   Compose starts postgres (with healthcheck), then backend (runs migrations + seed, listens 8080; optional OPENAI_API_KEY, GEMINI_API_KEY from .env), then frontend (nginx serves built SPA on 5173). Browser hits frontend; API calls go to backend (same host or VITE_API_URL). Backend connects to postgres by service name.

---

## 5. Requirements Checklist

### General

- Time-boxed, clean, production-style code, not overengineered: **Yes** (minimal deps, clear structure).
- Repo structure: `/frontend`, `/backend`, `docker-compose.yml`, `README.md`: **Yes**.
- Run with `docker compose up --build`: **Yes**.
- Environment variables: **Yes** (backend: DB\_\*, JWT_SECRET, optional OPENAI_API_KEY/GEMINI_API_KEY; frontend: VITE_API_URL; compose uses .env).
- Basic but meaningful tests: **Yes** (backend: login, auth guard, order validation, order summary requires auth, order summary fallback when no AI key; frontend: login redirect, auth guard, past datetime, summary data, AI summary button calls getOrderSummary and shows result).
- Correctness, clarity, simplicity: **Yes**.

### Backend

- Go, net/http, PostgreSQL, JWT, database/sql, golang-migrate: **Yes**.
- REST API, JSON: **Yes**.
- JWT-based auth: **Yes** (login issues JWT; protected routes use middleware).
- Seed one test user via migrations + SeedTestUser: **Yes**.
- Validation on all inputs: **Yes** (login; order preference enum, conditional address/pickup_time, future pickup_time).
- Docker-ready: **Yes**.
- Endpoints: POST /auth/login, GET /me, GET /orders, POST /orders, GET /orders/:id, PUT /orders/:id, GET /orders/:id/summary: **Yes** (all implemented; orders scoped by user; summary returns AI or fallback).
- Business rules: Login email/password; preference enum IN_STORE, DELIVERY, CURBSIDE; conditional fields; pickup_time future; orders belong to user: **Yes**.
- DB: users (id, email, password_hash), orders (id, user_id, preference, address, pickup_time, created_at), FK: **Yes**.
- Tests: Login success/failure, auth guard, order validation, order summary requires auth, order summary fallback when no AI key: **Yes**.

### Frontend

- React (Vite), TypeScript, React Router, React Hook Form, Zod, localStorage: **Yes**.
- Pages: Login (validation, redirect); Delivery Preference (two-step: set preference → delivery details & AI summary; Back to step 1; Logout in Layout): **Yes**.
- Auth-protected routes, inline validation, persist auth + order state, redirect unauthenticated: **Yes**.
- Tests: Login redirect, auth guard, past datetime rejected, summary reflects data, AI summary button calls getOrderSummary and shows summary + "Generated with AI": **Yes**.

### Docker

- docker-compose: postgres, backend, frontend: **Yes**.
- Backend waits for DB and runs migrations: **Yes** (depends_on postgres healthy; main.go runs migrations on start).
- Frontend talks to backend: **Yes** (via VITE_API_URL or same host; CORS on backend).

### AI order summary (bonus)

- Backend-proxied: **Yes** (GET /orders/:id/summary; no keys in frontend).
- Disabled gracefully: **Yes** (no OPENAI_API_KEY or GEMINI_API_KEY → fallback summary; API failure → fallback).
- Mockable for tests: **Yes** (tests run without keys; frontend mocks getOrderSummary).

### README

- How to run (npm scripts: `npm run up`, `npm run dev`, etc.), project features, default user, available scripts: **Yes** (README); full detail in this doc.

---

## 6. How to Run and Scripts

- **Full stack (Docker):** `docker compose up --build` → frontend :5173, backend :8080, postgres :5433. Optional: add `OPENAI_API_KEY` and/or `GEMINI_API_KEY` to `.env` for AI order summary; Compose passes them to the backend.
- **Local dev:**
  - `npm run dev:db` → start Postgres.
  - `npm run dev:backend` → backend (loads .env, runs migrations + seed).
  - `npm run dev:frontend` → Vite dev server (proxies API).
  - Or `npm run dev` to run backend + frontend together (concurrently).
- **Migrations:**
  - `npm run migrate` or `npm run migrate:up` → apply migrations.
  - `npm run migrate:down` → roll back all.
  - `npm run migrate:create -- <name>` → create new migration pair in `backend/migrations/`.
- **Tests:**
  - `npm run test:backend` (Go; needs DB for integration tests).
  - `npm run test:frontend` (Vitest).
  - `npm run test` runs both.

Default login after migrations/seed: **user@weel.com** / **password**.
