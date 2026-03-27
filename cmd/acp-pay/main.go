// acp-pay is a CLI tool for testing ACP-enabled endpoints.
// It acts as an agent: hits a URL, handles 402, pays with a mock method, and prints results.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/paideia-ai/acp"
	"github.com/paideia-ai/acp/methods/mock"
	"github.com/paideia-ai/acp/transport/acphttp"
)

func main() {
	url := flag.String("url", "", "URL to request (required)")
	method := flag.String("method", http.MethodGet, "HTTP method")
	maxSpend := flag.String("max-spend", "", "maximum spend per session (e.g., 100.00)")
	currency := flag.String("currency", "USD", "budget currency")
	flag.Parse()

	if *url == "" {
		fmt.Fprintln(os.Stderr, "usage: acp-pay --url <url>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	gateway := acp.NewGateway(
		acp.WithMethod(mock.New(mock.Config{})),
	)

	var clientOpts []acphttp.ClientOption
	if *maxSpend != "" {
		clientOpts = append(clientOpts, acphttp.WithBudget(acp.Budget{
			MaxPerSession: *maxSpend,
			Currency:      acp.Currency(*currency),
		}))
	}

	client := acphttp.NewClient(gateway, clientOpts...)

	req, err := http.NewRequest(*method, *url, nil)
	if err != nil {
		log.Fatalf("failed to create request: %v", err)
	}

	fmt.Printf(">>> %s %s\n", req.Method, req.URL)
	fmt.Println()

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("<<< %s\n", resp.Status)

	// Print payment response header if present.
	if pr := resp.Header.Get(acphttp.HeaderPaymentResponse); pr != "" {
		var settle map[string]any
		if err := acphttp.DecodeHeader(pr, &settle); err == nil {
			formatted, _ := json.MarshalIndent(settle, "  ", "  ")
			fmt.Printf("  Payment Receipt:\n  %s\n", formatted)
		}
	}

	fmt.Println()

	// Print response body.
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		// Try to pretty-print JSON.
		var parsed any
		if err := json.Unmarshal(body, &parsed); err == nil {
			formatted, _ := json.MarshalIndent(parsed, "", "  ")
			fmt.Println(string(formatted))
		} else {
			fmt.Println(string(body))
		}
	}
}
