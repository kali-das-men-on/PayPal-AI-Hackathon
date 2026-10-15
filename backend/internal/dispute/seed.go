package dispute

import "time"

// SeedDemo loads three synthetic disputes (demo mode only). All data is fake.
func SeedDemo(s *Store) {
	now := time.Now()
	day := 24 * time.Hour
	s.Put(Dispute{
		ID: "PP-D-1001", Reason: "MERCHANDISE_OR_SERVICE_NOT_RECEIVED", Amount: 42.00, Currency: "USD",
		Buyer: "buyer.one@example.com", Item: "Soy candle gift set",
		OpenedAt: now.Add(-3 * day), DueAt: now.Add(2 * day), Status: StatusNew,
		Evidence: Evidence{Carrier: "USPS", TrackingNumber: "DEMO9400111899", DeliveryStatus: "delivered",
			DeliveredAt: now.Add(-2 * day).Format(time.RFC3339), BuyerMessage: "Never arrived."},
	})
	s.Put(Dispute{
		ID: "PP-D-1002", Reason: "MERCHANDISE_OR_SERVICE_NOT_RECEIVED", Amount: 18.50, Currency: "USD",
		Buyer: "buyer.two@example.com", Item: "Enamel pin",
		OpenedAt: now.Add(-1 * day), DueAt: now.Add(6 * day), Status: StatusNew,
		Evidence: Evidence{DeliveryStatus: "none", BuyerMessage: "Where is my order?"},
	})
	s.Put(Dispute{
		ID: "PP-D-1003", Reason: "MERCHANDISE_OR_SERVICE_NOT_AS_DESCRIBED", Amount: 310.00, Currency: "USD",
		Buyer: "buyer.three@example.com", Item: "Handmade ceramic lamp",
		OpenedAt: now.Add(-1 * day), DueAt: now.Add(9 * day), Status: StatusNew,
		Evidence: Evidence{Carrier: "UPS", TrackingNumber: "DEMO1Z999AA1", DeliveryStatus: "delivered",
			DeliveredAt: now.Add(-4 * day).Format(time.RFC3339), BuyerMessage: "Arrived chipped."},
	})
}
