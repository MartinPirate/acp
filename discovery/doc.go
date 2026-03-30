// Package discovery provides service discovery and health checking for ACP
// payment methods and facilitators.
//
// A [Registry] stores [ServiceEntry] records describing available payment
// services, their supported methods, intents, currencies, regions, and
// amount ranges. Services can be queried with [Query] filters.
//
// # Key Types
//
//   - [ServiceEntry] -- describes a registered payment service with status
//     tracking ([StatusActive], [StatusDegraded], [StatusOffline]).
//   - [Registry] -- thread-safe registry with register, unregister, and
//     filtered discovery.
//   - [HealthChecker] -- background goroutine that periodically pings
//     service health endpoints and updates their status.
//   - [WellKnownHandler] -- serves the /.well-known/acp-services endpoint
//     as JSON, returning all active services.
//
// # Usage
//
//	registry := discovery.NewRegistry()
//	registry.Register(discovery.ServiceEntry{
//	    ID:         "stripe-cards",
//	    Name:       "Stripe Card Payments",
//	    Type:       discovery.ServiceTypeMethod,
//	    URL:        "https://pay.example.com",
//	    Methods:    []string{"card"},
//	    Currencies: []core.Currency{core.USD, core.EUR},
//	    Status:     discovery.StatusActive,
//	    HealthURL:  "https://pay.example.com/health",
//	})
//
//	// Start background health checking.
//	hc := discovery.NewHealthChecker(registry, 30*time.Second)
//	hc.Start(ctx)
//
//	// Discover services matching a query.
//	results := registry.Discover(discovery.Query{
//	    Currencies: []core.Currency{core.USD},
//	    Status:     discovery.StatusActive,
//	})
package discovery
