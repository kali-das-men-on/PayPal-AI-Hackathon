// Package api is the HTTP layer: it translates requests into calls on the
// store / PayPal / AI / policy packages and writes JSON back. No business rules live here.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"disputedesk/internal/ai"
	"disputedesk/internal/dispute"
	"disputedesk/internal/paypal"
	"disputedesk/internal/policy"
)

// Server holds its dependencies as fields (dependency injection by constructor).
type Server struct {
	store    *dispute.Store
	gw       paypal.Gateway // interface, not a concrete type
	analyzer ai.Analyzer    // interface
	mu       sync.RWMutex
	pol      policy.Policy
}

func NewServer(store *dispute.Store, gw paypal.Gateway, an ai.Analyzer) *Server {
	return &Server{store: store, gw: gw, analyzer: an, pol: policy.Default()}
}

// Routes uses Go 1.22's pattern-based ServeMux: "METHOD /path/{wildcard}".
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"ok": "true"}) })
	mux.HandleFunc("GET /api/disputes", s.listDisputes)
	mux.HandleFunc("GET /api/disputes/{id}", s.getDispute)
	mux.HandleFunc("POST /api/disputes/{id}/analyze", s.analyze)
	mux.HandleFunc("POST /api/disputes/{id}/approve", s.approve)
	mux.HandleFunc("POST /api/disputes/{id}/reject", s.reject)
	mux.HandleFunc("GET /api/policy", s.getPolicy)
	mux.HandleFunc("PUT /api/policy", s.putPolicy)
	mux.HandleFunc("GET /api/audit", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, s.store.Audit()) })
	return logging(mux)
}

// logging is middleware: a function that wraps a handler and returns a handler.
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func (s *Server) listDisputes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.store.List())
}

func (s *Server) getDispute(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	writeJSON(w, 200, d)
}

// analyze: AI proposes, policy disposes. The AI never acts directly.
func (s *Server) analyze(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := s.store.Get(id)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	a, err := s.analyzer.Analyze(ctx, d)
	if err != nil {
		writeErr(w, 502, err)
		return
	}
	dec := policy.Decide(s.currentPolicy(), d, a)
	s.store.Log(id, "agent", "analyzed", a.Engine+": "+a.Recommendation+" ("+a.EvidenceStrength+" evidence)")

	status := dispute.StatusAwaitingApproval
	switch dec.Action {
	case policy.ActionEscalate:
		status = dispute.StatusNeedsHuman
	case policy.ActionAutoRefund:
		d.Decision = &dec
		if err := s.execute(ctx, d, a.DraftResponse); err != nil {
			writeErr(w, 502, err)
			return
		}
		status = dispute.StatusResolved
		s.store.Log(id, "agent", "auto_refund", dec.Reason)
	}
	if status != dispute.StatusResolved {
		s.store.Log(id, "agent", dec.Action, dec.Reason)
	}
	updated, _ := s.store.Update(id, func(x *dispute.Dispute) { x.Analysis, x.Decision, x.Status = &a, &dec, status })
	writeJSON(w, 200, updated)
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := s.store.Get(id)
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	if d.Analysis == nil || d.Decision == nil || d.Status == dispute.StatusResolved {
		writeErr(w, 409, errors.New("dispute is not awaiting a decision"))
		return
	}
	var body struct {
		Response string `json:"response"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body) // body is optional
	note := body.Response
	if note == "" {
		note = d.Analysis.DraftResponse
	}
	if err := s.execute(r.Context(), d, note); err != nil {
		writeErr(w, 502, err)
		return
	}
	s.store.Log(id, "merchant", "approved", d.Decision.Recommendation)
	updated, _ := s.store.Update(id, func(x *dispute.Dispute) { x.Status = dispute.StatusResolved })
	writeJSON(w, 200, updated)
}

func (s *Server) reject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := s.store.Update(id, func(x *dispute.Dispute) { x.Status = dispute.StatusRejected })
	if err != nil {
		writeErr(w, 404, err)
		return
	}
	s.store.Log(id, "merchant", "rejected", "merchant will handle manually")
	writeJSON(w, 200, updated)
}

// execute performs the real-world action chosen by the policy engine.
func (s *Server) execute(ctx context.Context, d dispute.Dispute, note string) error {
	if d.Decision.Recommendation == "contest" {
		return s.gw.SubmitEvidence(ctx, d.ID, note, d.Evidence)
	}
	return s.gw.Refund(ctx, d.ID, d.Amount, d.Currency, note)
}

func (s *Server) currentPolicy() policy.Policy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pol
}

func (s *Server) getPolicy(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, s.currentPolicy())
}

func (s *Server) putPolicy(w http.ResponseWriter, r *http.Request) {
	var p policy.Policy
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.MinConfidence < 0 || p.MinConfidence > 1 {
		writeErr(w, 400, errors.New("invalid policy"))
		return
	}
	s.mu.Lock()
	s.pol = p
	s.mu.Unlock()
	s.store.Log("-", "merchant", "policy_updated", "")
	writeJSON(w, 200, p)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}
