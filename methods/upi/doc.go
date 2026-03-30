// Package upi implements UPI (Unified Payments Interface) payments via
// Razorpay for the Indian market.
//
// UPI is India's real-time inter-bank payment system. This method supports
// the charge intent in INR only, using a merchant VPA (Virtual Payment
// Address) for collections.
//
// # Key Types
//
//   - [Config] -- Razorpay API credentials and merchant VPA.
//   - [UPIMethod] -- the [core.Method] implementation.
//   - [Payload] -- UPI-specific payload containing VPA, transaction
//     reference, and UPI transaction ID.
//
// # Usage
//
//	method, err := upi.New(upi.Config{
//	    APIKey:      "rzp_live_...",
//	    APISecret:   "secret",
//	    MerchantVPA: "merchant@upi",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	gateway := acp.NewGateway(acp.WithMethod(method))
package upi
