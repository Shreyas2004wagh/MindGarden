# Mind Garden

Mind Garden is a privacy-focused journaling app with an AI question-answering layer. Users write journal entries, the Go backend chunks and embeds those entries, stores the vectors in PostgreSQL with pgvector, and answers questions using retrieval-augmented generation (RAG) grounded only in the user's own journal content.

## What It Does

- Email/password and Google sign-in through Supabase Auth.
- Auth-gated journal workspace with entry list, detail view, and new-entry editor.
- Journal storage in Supabase PostgreSQL with row-level security.
- Backend journal ingestion pipeline:
  - verifies the Supabase JWT,
  - stores the journal,
  - queues ingestion work,
  - chunks journal content,
  - generates Gemini embeddings,
  - persists vectors in PostgreSQL.
- Ask AI flow:
  - embeds the user's question,
  - retrieves relevant journal chunks for that user,
  - asks Groq to answer using only retrieved journal context,
  - returns the answer and source metadata.
- Local frontend proxy/middleware protects private routes and redirects unauthenticated users to login.

## Tech Stack

Frontend:
- Next.js 16
- React 18
- TypeScript
- Tailwind CSS
- Radix UI / shadcn-style components
- Supabase SSR client

Backend:
- Go 1.25+
- Chi router
- Supabase PostgreSQL
- pgvector
- Gemini embeddings
- Groq chat completions

## Repository Structure

```text
.
|-- Client/                 # Next.js app
|   |-- app/                # App Router pages and API route proxy
|   |-- components/         # UI and layout components
|   |-- lib/                # Supabase clients, auth helpers, utilities
|   |-- supabase/           # Supabase SQL migration for journals
|   |-- proxy.ts            # Next proxy for auth redirects
|   `-- package.json
|-- Server/                 # Go backend
|   |-- cmd/api             # API entrypoint
|   |-- internal/api        # Routes and handlers
|   |-- internal/service    # Chunking, ingestion, LLM, vector services
|   |-- migrations          # Optional SQL scripts for embeddings/jobs
|   `-- go.mod
|-- Featurelist.md          # MVP implementation plan
`-- README.md
```

## Prerequisites

Install:
- Node.js 20.9 or newer
- npm
- Go 1.25 or newer
- A Supabase project
- A Gemini API key
- A Groq API key

You also need a Supabase database with pgvector available. In Supabase, enable it from the SQL editor:

```sql
create extension if not exists vector;
```

## Environment Variables

Create local env files from the examples.

### Client

```powershell
cd Client
Copy-Item .env.example .env.local
```

Fill in:

```env
NEXT_PUBLIC_SUPABASE_URL=https://your-project.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=your_supabase_anon_key
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Server

```powershell
cd Server
Copy-Item .env.example .env
```

Fill in:

```env
PORT=8080
CLIENT_ORIGIN=http://localhost:3000

SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your_supabase_anon_key
SUPABASE_JWT_SECRET=your_supabase_jwt_secret_if_using_hs256

DATABASE_URL=postgresql://postgres:[password]@[host]:[port]/postgres?sslmode=require

GEMINI_API_KEY=your_gemini_api_key
GROQ_API_KEY=your_groq_api_key
```

`SUPABASE_JWT_SECRET` is required for Supabase projects using HS256 JWT signing. For asymmetric Supabase JWTs, the backend can verify tokens from the project's JWKS endpoint.

## Database Setup

Run the journal table migration in the Supabase SQL editor:

```text
Client/supabase/migrations/20260129112611_create_journals_table.sql
```

This creates:
- `journals`
- indexes for `user_id` and `created_at`
- row-level security policies
- an `updated_at` trigger

The Go server initializes these tables automatically at startup when the database is reachable:
- `embeddings`
- `ingestion_jobs`

Optional SQL scripts are also available:

```text
Server/migrations/001_update_embeddings_schema.sql
Server/migrations/002_create_ingestion_jobs.sql
```

## Local Development

Install frontend dependencies:

```powershell
cd Client
npm install
```

Start the Go backend:

```powershell
cd Server
go run ./cmd/api
```

The backend runs on `http://localhost:8080` by default.

Start the Next.js client:

```powershell
cd Client
npm run dev
```

The client runs on `http://localhost:3000` by default.

If ports are already in use, choose alternatives:

```powershell
# Server
$env:PORT="8091"
$env:CLIENT_ORIGIN="http://localhost:3011"
go run ./cmd/api

# Client
$env:NEXT_PUBLIC_API_URL="http://localhost:8091"
node .\node_modules\next\dist\bin\next dev -p 3011 -H 127.0.0.1
```

## API Endpoints

Backend:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Health check |
| `POST` | `/journals` | Create a journal and queue ingestion |
| `POST` | `/ingest` | Queue ingestion for an existing journal payload |
| `POST` | `/ask` | Retrieve journal context and generate a grounded answer |

Client:

| Method | Path | Purpose |
| --- | --- | --- |
| `POST` | `/api/ask` | Authenticated proxy from Next.js to the Go `/ask` endpoint |

Protected backend endpoints require:

```http
Authorization: Bearer <supabase_access_token>
```

## Validation Commands

Frontend:

```powershell
cd Client
npm run lint
npm run typecheck
npm run build
npm audit --audit-level=moderate
```

Backend:

```powershell
cd Server
go test ./...
```

Repository hygiene:

```powershell
git diff --check
```

## Deployment Notes

Client:
- Deploy from `Client/`.
- Set `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY`, and `NEXT_PUBLIC_API_URL`.
- The client uses Next App Router and `proxy.ts` for auth redirects.

Server:
- Deploy from `Server/`.
- Set all server env vars from `Server/.env.example`.
- Ensure the server can reach Supabase PostgreSQL.
- Set `CLIENT_ORIGIN` to the deployed frontend origin for CORS.

Database:
- Run the Supabase journal migration before using the app.
- Ensure pgvector is enabled.
- Confirm the `journals`, `embeddings`, and `ingestion_jobs` tables exist.

## Troubleshooting

### Login page loads, but auth does not work

Check that the client has:

```env
NEXT_PUBLIC_SUPABASE_URL
NEXT_PUBLIC_SUPABASE_ANON_KEY
```

### Backend starts but journal or AI endpoints fail

Check that `Server/.env` is not empty and includes:

```env
DATABASE_URL
SUPABASE_URL
GEMINI_API_KEY
GROQ_API_KEY
```

### `/ask` returns not enough information

This usually means the user has no ingested embeddings yet, or vector search did not find relevant chunks. Create a journal entry first and confirm ingestion jobs are completing.

### CORS errors

Set `CLIENT_ORIGIN` in `Server/.env` to the exact frontend origin, for example:

```env
CLIENT_ORIGIN=http://localhost:3000
```

### Supabase JWT verification fails

For HS256 projects, set:

```env
SUPABASE_JWT_SECRET=...
```

For asymmetric projects, verify `SUPABASE_URL` is correct so the backend can fetch JWKS.

## Current Limitations

- No journal editing UI.
- No conversation history for AI questions.
- No streaming AI responses.
- Ingestion is background-queued, so newly created entries may not be searchable immediately.
- The app depends on external Gemini and Groq API availability.

## Useful Links

- Supabase project API settings: `https://supabase.com/dashboard/project/_/settings/api`
- Gemini API keys: `https://ai.google.dev`
- Groq console: `https://console.groq.com`
