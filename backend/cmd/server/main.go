// main wires everything together: read config, build dependencies, start the server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"disputedesk/internal/ai"
	"disputedesk/internal/api"
	"disputedesk/internal/dispute"
	"disputedesk/internal/paypal"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	store := dispute.NewStore()
	var gw paypal.Gateway = paypal.Mock{}

	if id, secret := os.Getenv("PAYPAL_CLIENT_ID"), os.Getenv("PAYPAL_CLIENT_SECRET"); id != "" && secret != "" {
		sb := paypal.NewSandbox(id, secret)
		gw = sb
		items, err := sb.ListDisputes(context.Background())
		if err != nil {
			log.Fatalf("could not load sandbox disputes: %v", err)
		}
		for _, d := range items {
			store.Put(d)
		}
		log.Printf("PayPal SANDBOX mode: loaded %d disputes", len(items))
	} else {
		dispute.SeedDemo(store)
		log.Println("DEMO mode: synthetic disputes, mock PayPal gateway (set PAYPAL_CLIENT_ID/SECRET for sandbox)")
	}

	srv := api.NewServer(store, gw, ai.NewHTTP(env("AI_URL", "http://localhost:8000")))
	addr := ":" + env("PORT", "8080")
	log.Printf("DisputeDesk API listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Routes()))
}
