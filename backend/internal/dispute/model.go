// Package dispute holds the core data types and the in-memory store.
// It imports nothing from the rest of the app, so every other package can depend on it.
package dispute

import "time"

type Status string

const (
	StatusNew              Status = "new"
	StatusAwaitingApproval Status = "awaiting_approval"
	StatusNeedsHuman       Status = "needs_human"
	StatusResolved         Status = "resolved"
	StatusRejected         Status = "rejected"
)

// Evidence is the facts we hold about the order. Struct tags (`json:"..."`)
// control the field names used when the struct is turned into JSON.
type Evidence struct {
	Carrier        string `json:"carrier,omitempty"`
	TrackingNumber string `json:"tracking_number,omitempty"`
	DeliveryStatus string `json:"delivery_status"` // delivered | in_transit | none
	DeliveredAt    string `json:"delivered_at,omitempty"`
	BuyerMessage   string `json:"buyer_message,omitempty"`
}

// Analysis is what the AI service returns.
type Analysis struct {
	Category         string  `json:"category"`
	Recommendation   string  `json:"recommendation"`    // contest | settle
	EvidenceStrength string  `json:"evidence_strength"` // strong | weak | none
	Confidence       float64 `json:"confidence"`
	Reasoning        string  `json:"reasoning"`
	DraftResponse    string  `json:"draft_response"`
	Engine           string  `json:"engine"` // which model (or "heuristic") produced this
}

// Decision is what the policy engine chose to do with an Analysis.
type Decision struct {
	Action         string `json:"action"`         // auto_refund | await_approval | escalate
	Recommendation string `json:"recommendation"` // effective: contest | settle
	Reason         string `json:"reason"`
}

type Dispute struct {
	ID       string    `json:"id"`
	Reason   string    `json:"reason"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
	Buyer    string    `json:"buyer"`
	Item     string    `json:"item"`
	OpenedAt time.Time `json:"opened_at"`
	DueAt    time.Time `json:"due_at"`
	Status   Status    `json:"status"`
	Evidence Evidence  `json:"evidence"`
	Analysis *Analysis `json:"analysis,omitempty"` // pointer: nil means "not analyzed yet"
	Decision *Decision `json:"decision,omitempty"`
}

type AuditEntry struct {
	At        time.Time `json:"at"`
	DisputeID string    `json:"dispute_id"`
	Actor     string    `json:"actor"` // agent | merchant | system
	Event     string    `json:"event"`
	Detail    string    `json:"detail"`
}
