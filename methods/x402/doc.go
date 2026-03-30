// Package x402 implements the x402 bridge for USDC stablecoin payments on
// EVM-compatible blockchains.
//
// The x402 protocol uses EIP-3009 transfer-with-authorization signatures
// to enable gasless USDC payments. A facilitator service verifies the
// signature and executes the on-chain transfer. This method supports the
// charge intent in USDC.
//
// # Key Types
//
//   - [Config] -- facilitator URL, network identifier (e.g., "eip155:8453"
//     for Base), and the client's hex-encoded private key for signing.
//   - [X402Method] -- the [core.Method] implementation.
//   - [Payload] -- contains the EIP-3009 [Authorization] struct and signature.
//   - [Authorization] -- from, to, value, validity window, and nonce.
//
// # Usage
//
//	method, err := x402.New(x402.Config{
//	    FacilitatorURL: "https://x402.example.com",
//	    Network:        "eip155:8453",
//	    PrivateKey:     "0xabc123...",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package x402
