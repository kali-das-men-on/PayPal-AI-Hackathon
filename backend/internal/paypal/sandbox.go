package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"disputedesk/internal/dispute"
)

// Sandbox calls PayPal's Disputes API (https://api-m.sandbox.paypal.com).
// NOTE: endpoints come from PayPal's docs, but the write payloads below have
// NOT been verified against a live sandbox account yet. Test them first.
type Sandbox struct {
	base, id, secret string
	http             *http.Client
	mu               sync.Mutex
	token            string
	expires          time.Time
}

func NewSandbox(id, secret string) *Sandbox {
	return &Sandbox{base: "https://api-m.sandbox.paypal.com", id: id, secret: secret,
		http: &http.Client{Timeout: 20 * time.Second}}
}

func (s *Sandbox) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.expires) {
		return s.token, nil
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", s.base+"/v1/oauth2/token",
		strings.NewReader(url.Values{"grant_type": {"client_credentials"}}.Encode()))
	req.SetBasicAuth(s.id, s.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal token: HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	s.token, s.expires = out.AccessToken, time.Now().Add(time.Duration(out.ExpiresIn-60)*time.Second)
	return s.token, nil
}

// call is one helper for every request: auth header, JSON in, JSON out, error on non-2xx.
func (s *Sandbox) call(ctx context.Context, method, path string, body, out any) error {
	tok, err := s.accessToken(ctx)
	if err != nil {
		return err
	}
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(ctx, method, s.base+path, rd)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("paypal %s %s: HTTP %d: %s", method, path, resp.StatusCode, b)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func (s *Sandbox) ListDisputes(ctx context.Context) ([]dispute.Dispute, error) {
	var resp struct {
		Items []struct {
			DisputeID     string    `json:"dispute_id"`
			Reason        string    `json:"reason"`
			CreateTime    time.Time `json:"create_time"`
			DueDate       time.Time `json:"seller_response_due_date"`
			DisputeAmount struct {
				Currency string `json:"currency_code"`
				Value    string `json:"value"`
			} `json:"dispute_amount"`
		} `json:"items"`
	}
	if err := s.call(ctx, "GET", "/v1/customer/disputes", nil, &resp); err != nil {
		return nil, err
	}
	var out []dispute.Dispute
	for _, it := range resp.Items {
		amt, _ := strconv.ParseFloat(it.DisputeAmount.Value, 64)
		due := it.DueDate
		if due.IsZero() {
			due = it.CreateTime.Add(10 * 24 * time.Hour) // PayPal's default seller window
		}
		// TODO: enrich Evidence from the transactions/shipment APIs; empty for now.
		out = append(out, dispute.Dispute{ID: it.DisputeID, Reason: it.Reason, Amount: amt,
			Currency: it.DisputeAmount.Currency, OpenedAt: it.CreateTime, DueAt: due,
			Status: dispute.StatusNew, Evidence: dispute.Evidence{DeliveryStatus: "none"}})
	}
	return out, nil
}

func (s *Sandbox) Refund(ctx context.Context, id string, amount float64, currency, note string) error {
	body := map[string]any{"note": note, "offer_type": "REFUND",
		"offer_amount": map[string]string{"currency_code": currency, "value": fmt.Sprintf("%.2f", amount)}}
	return s.call(ctx, "POST", "/v1/customer/disputes/"+id+"/make-offer", body, nil)
}

func (s *Sandbox) SubmitEvidence(ctx context.Context, id, note string, ev dispute.Evidence) error {
	body := map[string]any{"evidences": []map[string]any{{
		"evidence_type": "PROOF_OF_FULFILLMENT", "notes": note,
		"evidence_info": map[string]any{"tracking_info": []map[string]string{{
			"carrier_name": "OTHER", "carrier_name_other": ev.Carrier, "tracking_number": ev.TrackingNumber}}},
	}}}
	return s.call(ctx, "POST", "/v1/customer/disputes/"+id+"/provide-evidence", body, nil)
}
