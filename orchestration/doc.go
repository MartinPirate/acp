// Package orchestration provides smart payment rail selection strategies
// for choosing the optimal payment method based on cost, speed, region,
// and other configurable criteria.
//
// # Strategies
//
// Every strategy implements the [Strategy] interface:
//
//   - [CheapestStrategy] -- selects the method with the lowest total fees
//     (fixed + percentage) using a [FeeTable].
//   - [FastestStrategy] -- selects the method with the shortest settlement time.
//   - [PreferredStrategy] -- matches agent preferences by method name.
//   - [RegionStrategy] -- selects the best method for the payer's ISO 3166 region.
//   - [RoundRobinStrategy] -- distributes selections across methods for load balancing.
//   - [CompositeStrategy] -- chains multiple strategies, trying each in order
//     until one succeeds.
//
// # Orchestrator
//
// [Orchestrator] wraps a [Strategy] and provides thread-safe, hot-swappable
// method selection. [OrchestratedGateway] combines a gateway with an
// orchestrator for automatic select-and-pay workflows.
//
// # Usage
//
//	fees := orchestration.NewFeeTable(map[string]orchestration.FeeInfo{
//	    "card":  {FixedFee: "0.30", PercentFee: 0.029, SettlementTime: 2 * 24 * time.Hour},
//	    "sepa":  {FixedFee: "0.10", PercentFee: 0.001, SettlementTime: 4 * time.Hour},
//	})
//	strategy := &orchestration.CheapestStrategy{FeeTable: fees}
//	orch := orchestration.NewOrchestrator(strategy)
//
//	selected, err := orch.Select(paymentOptions, orchestration.SelectionContext{
//	    Region: "DE",
//	    Amount: "25.00",
//	})
package orchestration
