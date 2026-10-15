// Package policy is a PURE function: same inputs, same output, no I/O.
// That makes the most important business rules trivial to test.
package policy

import "disputedesk/internal/dispute"

type Policy struct {
	AutoRefundBelow   float64 `json:"auto_refund_below"`
	EscalateAtOrAbove float64 `json:"escalate_at_or_above"`
	MinConfidence     float64 `json:"min_confidence"`
}

func Default() Policy { return Policy{AutoRefundBelow: 25, EscalateAtOrAbove: 200, MinConfidence: 0.7} }

const (
	ActionAutoRefund    = "auto_refund"
	ActionAwaitApproval = "await_approval"
	ActionEscalate      = "escalate"
)

func Decide(p Policy, d dispute.Dispute, a dispute.Analysis) dispute.Decision {
	rec := a.Recommendation
	// Fairness guardrail: never contest a claim when we hold no evidence.
	if rec == "contest" && a.EvidenceStrength == "none" {
		rec = "settle"
	}
	dec := dispute.Decision{Recommendation: rec}
	switch {
	case d.Amount >= p.EscalateAtOrAbove:
		dec.Action, dec.Reason = ActionEscalate, "amount at or above the escalation limit; a person must decide"
	case a.Confidence < p.MinConfidence:
		dec.Action, dec.Reason = ActionEscalate, "AI confidence below the minimum"
	case rec == "settle" && d.Amount <= p.AutoRefundBelow:
		dec.Action, dec.Reason = ActionAutoRefund, "small claim with no solid defense: refund fast and fairly"
	default:
		dec.Action, dec.Reason = ActionAwaitApproval, "within limits; waiting for merchant approval"
	}
	return dec
}
