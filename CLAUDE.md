# fcp-terraform

## Project Overview
MCP server that lets LLMs generate Terraform HCL through intent-level operation strings.
Go implementation using `hclwrite` natively (Tier 1 — no adapter layer, direct HCL AST manipulation).

## Architecture
3-layer architecture:
1. **MCP Server (Intent Layer)** — `cmd/fcp-terraform/main.go` — Registers 4 MCP tools, dispatches to adapter
2. **Semantic Model (Domain)** — `internal/terraform/` — In-memory Terraform graph, verb handlers, selectors, queries
3. **FCP Core** — `internal/fcpcore/` — Tokenizer, parsed-op, verb registry, event log, session, formatter

## Key Directories
- `cmd/fcp-terraform/` — Entry point, MCP server setup
- `internal/terraform/` — Domain: model, adapter, handlers, queries, selectors, values, verb specs
- `internal/fcpcore/` — Shared FCP framework (Go port of fcp-core)

## Commands
- `go test ./...` — Run all tests
- `go build ./cmd/fcp-terraform` — Build binary

## Conventions
- Go 1.25, standard library style
- `hclwrite` for HCL generation (no string concatenation)
- `mcp-go` for MCP protocol (stdio transport)
- Tests colocated with source (`*_test.go`)
