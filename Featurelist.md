# Mind Garden — MVP-v1 Backend & RAG Implementation Plan

This document defines the **required backend, RAG, and integration work** needed to complete **MVP-v1** of Mind Garden.

MVP-v1 is considered complete **only when the AI feature works end-to-end**:
journals → embeddings → retrieval → grounded AI answers.

---

## 🧠 A. Backend Foundation (MUST HAVE)

### Goal
Establish a **trusted application backend** that can securely interact with Supabase and support AI/RAG workflows.

### Core Tasks
- [x] Create Go backend repository / folder
- [x] Set up Go environment and dependencies
- [x] Implement Chi router with middleware
- [ ] Connect Go backend to Supabase PostgreSQL
- [ ] Verify Supabase JWT in Go
- [x] Implement basic health check endpoint

### Outcome
✅ Backend exists (Go)  
✅ Authentication is trusted  
✅ Ready to accept journal + AI workloads  

---

## 📓 B. Journal Ingestion Pipeline (CRITICAL)

This is the **starting point of RAG**.  
If this is incorrect, AI answers will be unreliable.

### Required Tasks
- [x] Backend endpoint to receive journal content
- [x] Chunk journal text (handled by vector store)
- [ ] Store chunk metadata in PostgreSQL
- [x] Generate embeddings for each chunk (using Gemini)
- [x] Persist embeddings into vector store (disk-based JSON)

### Notes
- Chunking must preserve order (`chunk_index`)
- Each chunk must be associated with:
  - user_id
  - journal_id
- No cross-user data access is allowed

### Implementation
- Go handler: `internal/api/handlers/ingest.go`
- LLM service: `internal/service/llm/service.go`
- Vector store: `internal/service/vector/store.go`

### Outcome
✅ Journals become **machine-understandable**  
✅ Foundation for semantic search is complete  

---

## 🧩 C. Vector Store Layer

### Goal
Enable **fast, private, semantic retrieval** of journal content.

### Required Tasks
- [x] Create vector store implementation
- [x] Load vector store from disk on request
- [x] Save vector store back to disk after updates
- [x] Add embeddings to vector store
- [ ] Delete embeddings when a journal is deleted
- [ ] Implement per-user vector store isolation

### Constraints
- One vector store per user (currently global for MVP)
- No shared/global index in production
- Stored on disk for persistence

### Implementation
- Custom Go vector store with cosine similarity search
- Disk persistence via JSON serialization
- Location: `internal/service/vector/store.go`

### Outcome
✅ Semantic search works  
🚧 User data isolation needed for production  

---

## 🤖 D. RAG Ask (REAL AI FEATURE)

This is the **core differentiator** of Mind Garden.

### Required Tasks
- [x] Implement `/ask` endpoint in Go
- [x] Embed the user's question (using Gemini)
- [x] Retrieve top-k relevant chunks from vector store
- [x] Construct grounded RAG prompt
- [x] Call LLM using Groq API
- [x] Return answer to frontend

### Guardrails (MANDATORY)
- [x] "Not enough data" fallback response
- [x] Strict context-only answers
- [x] Graceful error handling for:
  - LLM failures
  - Empty vector store
  - Retrieval errors

### Implementation
- Handler: `internal/api/handlers/ask.go`
- LLM service: `internal/service/llm/service.go` (Gemini embeddings + Groq inference)
- Vector search: `internal/service/vector/store.go`

### Outcome
✅ AI answers feel **trustworthy**  
✅ No hallucinated responses  
✅ User confidence in AI system  

---

## 🔌 E. Frontend ↔ Backend Integration

### Goal
Make the frontend and backend work as **one cohesive system**.

### Required Tasks
- [x] Create API proxy in Next.js (via `/api/ask`)
- [ ] On journal creation → call backend ingestion endpoint
- [x] Wire Ask AI page to backend `/ask` endpoint
- [x] Implement loading and error states in UI

### Notes
- Supabase handles auth & DB
- Go backend handles intelligence & RAG
- Frontend currently proxies through Next.js API routes

### Outcome
✅ Frontend + backend communicate correctly  
✅ AI feature usable from UI  

---

## 🎨 F. UX Polishing (AFTER AI WORKS)

UX improvements should happen **after RAG is functional**.

### Required Tasks
- [ ] Empty states (journals, AI responses)
- [ ] Clear and calm error messaging
- [ ] Dark theme contrast tuning
- [ ] Mobile usability pass

### Constraints
- No new features
- No analytics
- No animations

### Outcome
✅ App feels calm, polished, and trustworthy  

---

## 🚫 Explicitly Out of Scope for MVP-v1

The following **must not** be added in MVP-v1:

- Payments or subscriptions
- Conversation history
- Streaming AI responses
- Editing past journals
- Tags, moods, or analytics
- Notifications or reminders

---

## ✅ Definition of MVP-v1 Done

MVP-v1 is complete when:
- [x] Journals are embedded and searchable
- [x] AI answers are grounded in journal content
- [x] Errors are handled gracefully
- [ ] User data is private and isolated (per-user vector stores)
- [x] The system is stable and explainable

---

## 📌 Recommended Next Step

Convert each checkbox in this document into:
- A GitHub Issue
- With owner assignment
- With acceptance criteria

This document is the **single source of truth** for MVP-v1.

---

## 🏗️ Technical Stack Summary

**Backend**: Go 1.25+
- Router: Chi v5
- AI Embeddings: Google Gemini (text-embedding-004)
- LLM Inference: Groq API
- Vector Store: Custom Go implementation with disk persistence

**Frontend**: Next.js 13+ with TypeScript
- Auth & DB: Supabase
- UI: Radix UI + Tailwind CSS
- Styling: shadcn/ui components

**Architecture**: Clean architecture with internal service layers
