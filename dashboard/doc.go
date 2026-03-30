// Package dashboard provides a web-based monitoring dashboard for ACP
// payment transactions and method status.
//
// It serves an embedded static UI, a REST API for transaction statistics,
// and an integrated Swagger UI for the ACP OpenAPI specification.
//
// # Key Types
//
//   - [Transaction] -- a recorded payment event (method, amount, success).
//   - [TransactionStore] -- thread-safe in-memory ring buffer of transactions.
//   - [Stats] -- aggregate statistics: total volume, success rate, and
//     per-method breakdowns via [MethodStats].
//   - [Server] -- the HTTP server that serves the dashboard UI, the stats
//     API, and the Swagger documentation.
//
// # API Endpoints
//
//   - GET /api/stats -- aggregate payment statistics.
//   - GET /api/transactions?limit=N -- recent transactions.
//   - GET /api/methods -- registered payment method status.
//   - GET /api/health -- server health check.
//   - GET /docs/ -- Swagger UI for the ACP OpenAPI spec.
//
// # Usage
//
//	store := dashboard.NewTransactionStore(1000)
//	srv := dashboard.NewServer(dashboard.Config{ListenAddr: ":9090"}, store)
//	srv.SetMethods([]dashboard.MethodInfo{
//	    {Name: "card", Status: "active"},
//	    {Name: "upi", Status: "active"},
//	})
//	log.Fatal(srv.ListenAndServe())
package dashboard
