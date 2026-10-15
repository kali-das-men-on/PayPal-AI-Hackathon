# DisputeDesk Architecture (a Go-first guide)

## The big picture

```
 Browser (React, Vite on Node)         port 5173
   │  api.js  ── fetch("/api/...") ──► Vite proxy
   ▼
 Go backend  (the brain + the rules)   port 8080
   │  ├─ dispute/  data types + in-memory store
   │  ├─ policy/   pure decision rules   (can the agent act alone?)
   │  ├─ paypal/   Gateway interface: Mock | Sandbox
   │  ├─ ai/       Analyzer interface: calls the Python service
   │  └─ api/      HTTP handlers, routes, middleware
   ▼
 Python AI service (LangChain)         port 8000, run by uvicorn
```

**Why this split?** The AI *proposes*; the Go backend *decides and acts*. The model never touches PayPal directly. Its output goes through `policy.Decide`, a plain function with tests. That is the trust story you'll tell in the demo.

**A note on the stack you asked for:** `uvicorn` is a Python server, so it cannot run Go. It runs the small LangChain service. Go uses its own built-in `net/http` server. "JS for API" became `frontend/src/api.js`, the one file that knows the backend URLs.

## Go concepts you need, in the order you'll meet them

**1. Packages and `internal/`.** Every folder is one package; every file starts with `package <name>`. Anything under `internal/` can only be imported from inside this module, so it stays private. Names starting with a capital letter (`Store`, `NewServer`) are exported; lowercase names are private to the package.

**2. Structs and JSON tags.** Go has no classes. A `struct` is a bundle of fields, and methods are attached separately: `func (s *Store) Get(...)`. The backtick tags like `` `json:"due_at"` `` control the JSON field name. See `dispute/model.go`.

**3. Pointers.** `*Analysis` means "maybe an Analysis, or nil." We use it so "not analyzed yet" is representable. A method with a pointer receiver (`s *Store`) can modify the struct; without the `*` it works on a copy.

**4. Errors are values.** No exceptions. Functions return `(result, error)` and you check `if err != nil`. You will write this a lot. It's verbose but makes every failure path visible.

**5. Interfaces (the most important idea in this codebase).** An interface is a list of methods. Any type that has those methods satisfies it automatically; there is no `implements` keyword. `paypal.Gateway` has three methods; `Mock` and `Sandbox` both have them. The `api.Server` only knows `Gateway`, so changing from demo to real PayPal is a one-line change in `main.go`. The same trick makes the AI client replaceable.

**6. Constructors and dependency injection.** Go has no constructors, so we write `NewServer(store, gw, analyzer)`. Dependencies are passed in as arguments and stored in struct fields. Nothing is global. Read `cmd/server/main.go` first: it is the "wiring diagram."

**7. Concurrency and mutexes.** Go's `net/http` handles each request in its own goroutine (a lightweight thread). Two requests can touch the store at once, so `dispute.Store` guards its map with a `sync.RWMutex`. `defer mu.Unlock()` guarantees the unlock even if the function returns early. Rule of thumb: if data is shared and mutable, lock it.

**8. `context.Context`.** Carries deadlines and cancellation down a call chain. `analyze` wraps the request in a 60-second timeout, and the HTTP client to the AI service honors it, so a stuck model can't hang the server.

**9. HTTP handlers and the router.** A handler is `func(w http.ResponseWriter, r *http.Request)`. Go 1.22's `ServeMux` supports `"POST /api/disputes/{id}/analyze"` and `r.PathValue("id")`, so you need no router library. *Middleware* (`logging`) is a function that takes a handler and returns a wrapped handler.

**10. Table-driven tests.** `policy_test.go` lists cases in a slice and loops over them. Run `go test ./...`.

## Reading order for the backend

1. `cmd/server/main.go`: how pieces are assembled.
2. `internal/dispute/model.go`: the vocabulary.
3. `internal/policy/policy.go`: the rules (short; read it twice).
4. `internal/api/server.go`: the `analyze` handler is the heart: *fetch → AI proposes → policy decides → execute → audit*.
5. `internal/paypal/gateway.go`, then `sandbox.go`.
6. `internal/dispute/store.go`: locking.

## The request flow (what happens when you click "Run agent")

1. `api.js` → `POST /api/disputes/{id}/analyze` → Vite proxies it to Go.
2. `api.Server.analyze` loads the dispute from the store.
3. `ai.HTTPAnalyzer` posts it to the Python service. LangChain returns a structured `Analysis` (or the heuristic engine does when no model is configured; the response says which).
4. `policy.Decide` applies the merchant's limits and the fairness guardrail. It returns `auto_refund`, `await_approval` or `escalate`.
5. For `auto_refund`, the Gateway is called immediately. Otherwise the dispute waits for the merchant's Approve button.
6. Every step is written to the audit log.

## Extending it (good first Go exercises)

- Replace the in-memory `Store` with SQLite behind the same method set.
- Add a PayPal webhook handler (`POST /webhooks/paypal`) to ingest new disputes.
- Add a goroutine with a `time.Ticker` that flags disputes nearing their deadline.
- Add handler tests with `net/http/httptest`.
