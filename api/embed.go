// Package api provides embedded API documentation files for the ACP protocol.
package api

import "embed"

// Docs contains the embedded OpenAPI specification and Swagger UI files.
//
//go:embed openapi.yaml swagger-ui
var Docs embed.FS
