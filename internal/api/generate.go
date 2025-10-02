//go:generate oapi-codegen -generate types   -o types.gen.go  -package api ../../api/openapi.yaml
//go:generate oapi-codegen -generate client  -o client.gen.go -package api ../../api/openapi.yaml

// Package api provides generated types and clients from the OpenAPI specification.
package api
