// Package mock provides a mock payment method for testing ACP integrations.
//
// [MockMethod] implements [core.Method] with configurable failure modes and
// artificial latency. It supports the charge and authorize intents across
// 10 currencies and records all settled transactions for test assertions.
//
// # Key Types
//
//   - [Config] -- set ShouldFail to force errors, SettleDelay to simulate
//     latency.
//   - [MockMethod] -- the [core.Method] implementation.
//   - [Transaction] -- a recorded mock settlement with ID, amount, currency,
//     and timestamp.
//
// # Usage
//
//	method := mock.New(mock.Config{})
//	gateway := acp.NewGateway(acp.WithMethod(method))
//
//	// After running payments through the gateway:
//	txns := method.Transactions()
//	fmt.Println(txns[0].Amount) // "5.99"
//
//	// Test error paths:
//	failMethod := mock.New(mock.Config{ShouldFail: true})
//
//	// Reset recorded state between tests:
//	method.Reset()
package mock
