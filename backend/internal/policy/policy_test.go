package policy

import (
	"testing"

	"disputedesk/internal/dispute"
)

// Table-driven test: the idiomatic Go way to cover many cases compactly.
func TestDecide(t *testing.T) {
	p := Default()
	tests := []struct {
		name   string
		amount float64
		a      dispute.Analysis
		action string
		rec    string
	}{
		{"strong evidence mid amount waits for approval", 42, dispute.Analysis{Recommendation: "contest", EvidenceStrength: "strong", Confidence: 0.9}, ActionAwaitApproval, "contest"},
		{"no evidence small amount auto refunds", 18.5, dispute.Analysis{Recommendation: "settle", EvidenceStrength: "none", Confidence: 0.9}, ActionAutoRefund, "settle"},
		{"guardrail: contest with no evidence becomes settle", 18.5, dispute.Analysis{Recommendation: "contest", EvidenceStrength: "none", Confidence: 0.9}, ActionAutoRefund, "settle"},
		{"large amount escalates", 310, dispute.Analysis{Recommendation: "contest", EvidenceStrength: "strong", Confidence: 0.95}, ActionEscalate, "contest"},
		{"low confidence escalates", 42, dispute.Analysis{Recommendation: "contest", EvidenceStrength: "weak", Confidence: 0.4}, ActionEscalate, "contest"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Decide(p, dispute.Dispute{Amount: tc.amount}, tc.a)
			if got.Action != tc.action || got.Recommendation != tc.rec {
				t.Fatalf("got %+v, want action=%s rec=%s", got, tc.action, tc.rec)
			}
		})
	}
}
