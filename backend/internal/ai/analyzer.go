// Package ai calls the Python LangChain service over HTTP. Again an interface,
// so tests (or a future in-process model) can replace it.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"disputedesk/internal/dispute"
)

type Analyzer interface {
	Analyze(ctx context.Context, d dispute.Dispute) (dispute.Analysis, error)
}

type HTTPAnalyzer struct {
	url    string
	client *http.Client
}

func NewHTTP(url string) *HTTPAnalyzer {
	return &HTTPAnalyzer{url: url, client: &http.Client{Timeout: 60 * time.Second}}
}

func (h *HTTPAnalyzer) Analyze(ctx context.Context, d dispute.Dispute) (dispute.Analysis, error) {
	var a dispute.Analysis
	b, _ := json.Marshal(d)
	req, err := http.NewRequestWithContext(ctx, "POST", h.url+"/analyze", bytes.NewReader(b))
	if err != nil {
		return a, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return a, fmt.Errorf("ai service unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return a, fmt.Errorf("ai service HTTP %d: %s", resp.StatusCode, msg)
	}
	return a, json.NewDecoder(resp.Body).Decode(&a)
}
