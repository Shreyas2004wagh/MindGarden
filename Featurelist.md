# Mind Garden — MVP-v1 Backend & RAG Implementation Plan

This document defines the **required backend, RAG, and integration work** needed to complete **MVP-v1** of Mind Garden.

MVP-v1 is considered complete **only when the AI feature works end-to-end**:
journals → embeddings → retrieval → grounded AI answers.

---

## 🧠 A. Backend Foundation (MUST HAVE)

### Goal
Establish a **trusted application backend** that can securely interact with Supabase and support AI/RAG workflows.

### Core Tasks
- [ ] Create FastAPI backend repository / folder
- [ ] Set up Python environment
  - Virtual environment
  - `requirements.txt`
- [ ] Connect FastAPI to Supabase PostgreSQL
- [ ] Verify Supabase JWT in FastAPI
- [ ] Implement basic health check endpoint

### Outcome
✅ Backend exists  
✅ Authentication is trusted  
✅ Ready to accept journal + AI workloads  

---

## 📓 B. Journal Ingestion Pipeline (CRITICAL)

This is the **starting point of RAG**.  
If this is incorrect, AI answers will be unreliable.

### Required Tasks
- [ ] Backend endpoint to receive journal content
- [ ] Chunk journal text (300–500 tokens, with overlap)
- [ ] Store chunk metadata in PostgreSQL
- [ ] Generate embeddings for each chunk
- [ ] Persist embeddings into FAISS (per user)

### Notes
- Chunking must preserve order (`chunk_index`)
- Each chunk must be associated with:
  - user_id
  - journal_id
- No cross-user data access is allowed

### Outcome
✅ Journals become **machine-understandable**  
✅ Foundation for semantic search is complete  

---

## 🧩 C. Vector Store (FAISS) Layer

### Goal
Enable **fast, private, semantic retrieval** of journal content.

### Required Tasks
- [ ] Create per-user FAISS index
- [ ] Load FAISS index from disk on request
- [ ] Save FAISS index back to disk after updates
- [ ] Add embeddings to FAISS index
- [ ] Delete embeddings when a journal is deleted

### Constraints
- One FAISS index per user
- No shared/global index
- Stored on disk for persistence

### Outcome
✅ Semantic search works  
✅ User data is fully isolated  

---

## 🤖 D. RAG Ask (REAL AI FEATURE)

This is the **core differentiator** of Mind Garden.

### Required Tasks
- [ ] Implement `/ask` endpoint in FastAPI
- [ ] Embed the user’s question
- [ ] Retrieve top-k relevant chunks from FAISS
- [ ] Construct grounded RAG prompt
- [ ] Call LLM using Groq API
- [ ] Return answer and metadata (e.g., journal dates)

### Guardrails (MANDATORY)
- [ ] “Not enough data” fallback response
- [ ] Strict context-only answers
- [ ] Graceful error handling for:
  - LLM failures
  - Empty vector store
  - Retrieval errors

### Outcome
✅ AI answers feel **trustworthy**  
✅ No hallucinated responses  
✅ User confidence in AI system  

---

## 🔌 E. Frontend ↔ Backend Integration

### Goal
Make the frontend and backend work as **one cohesive system**.

### Required Tasks
- [ ] Create API client for FastAPI (separate from Supabase)
- [ ] On journal creation → call backend ingestion endpoint
- [ ] Wire Ask AI page to backend `/ask` endpoint
- [ ] Implement loading and error states in UI

### Notes
- Supabase handles auth & DB
- FastAPI handles intelligence & RAG
- Frontend must send Supabase JWT to backend

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
- Journals are embedded and searchable
- AI answers are grounded in journal content
- Errors are handled gracefully
- User data is private and isolated
- The system is stable and explainable

---

## 📌 Recommended Next Step

Convert each checkbox in this document into:
- A GitHub Issue
- With owner assignment
- With acceptance criteria

This document is the **single source of truth** for MVP-v1.