// Package alipay implements Alipay payments for the Chinese market.
//
// Alipay is China's leading digital payment platform with over 1 billion
// users. This method supports the charge intent in CNY, using the Alipay
// Open Platform gateway with RSA2 signature verification.
//
// # Key Types
//
//   - [Config] -- app ID, RSA2 private/public keys, and gateway URL.
//   - [AlipayMethod] -- the [core.Method] implementation.
//   - [Payload] -- Alipay-specific payload containing trade number, merchant
//     order number, and buyer ID.
//
// # Usage
//
//	method, err := alipay.New(alipay.Config{
//	    AppID:           "2021000000000000",
//	    PrivateKey:      "MIIEvQIBADANBg...",
//	    AlipayPublicKey: "MIIBIjANBgkqhk...",
//	    Gateway:         "https://openapi.alipay.com/gateway.do",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package alipay
