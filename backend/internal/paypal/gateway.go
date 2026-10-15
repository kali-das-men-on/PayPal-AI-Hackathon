// Package paypal talks to PayPal. The Gateway INTERFACE is the key idea:
// the rest of the app depends on the interface, not on a concrete type,
// so we can swap a Mock for the real Sandbox client without changing any other code.
package paypal

import (
	"context"
	"log"

	"disputedesk/internal/dispute"
)

type Gateway interface {
	ListDisputes(ctx context.Context) ([]dispute.Dispute, error)
	Refund(ctx context.Context, id string, amount float64, currency, note string) error
	SubmitEvidence(ctx context.Context, id, note string, ev dispute.Evidence) error
}

// Mock satisfies Gateway implicitly: Go has no "implements" keyword; having the methods is enough.
type Mock struct{}

func (Mock) ListDisputes(context.Context) ([]dispute.Dispute, error) { return nil, nil }

func (Mock) Refund(_ context.Context, id string, amount float64, currency, _ string) error {
	log.Printf("[mock paypal] refund offer %s %.2f %s", id, amount, currency)
	return nil
}

func (Mock) SubmitEvidence(_ context.Context, id, _ string, ev dispute.Evidence) error {
	log.Printf("[mock paypal] evidence submitted for %s (tracking %s)", id, ev.TrackingNumber)
	return nil
}
