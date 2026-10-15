# DisputeDesk

An AI agent that helps small PayPal sellers answer payment disputes on time, and settles valid claims fairly instead of fighting them. Built for the "Build What's Next with PayPal and AI" hackathon.

**Honest status:** runs fully in **demo mode** (synthetic disputes, mock PayPal gateway). The PayPal **sandbox** client lists disputes and has refund/evidence calls, but those write payloads have **not been verified against a live sandbox account**. Verify them before claiming sandbox support in your submission.

## Run it (three terminals)

Requirements: Go 1.22+, Python 3.10+, Node 18+.

```bash
# 1. AI service (LangChain, served by uvicorn)
cd ai-service && python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
uvicorn main:app --port 8000          # heuristic engine; no API key needed

# 2. Go backend
cd backend && go run ./cmd/server     # http://localhost:8080

# 3. Frontend
cd frontend && npm install && npm run dev   # http://localhost:5173
```

**Use a real model:** `export DD_MODEL="anthropic:claude-sonnet-5-5"` and `export ANTHROPIC_API_KEY=...` before starting uvicorn.
**Use PayPal sandbox:** `export PAYPAL_CLIENT_ID=... PAYPAL_CLIENT_SECRET=...` before starting the Go server.

## Test

`cd backend && go test ./...`

See [ARCHITECTURE.md](ARCHITECTURE.md) for how everything fits together.

## License

MIT (see LICENSE). Replace the name in LICENSE before publishing.
