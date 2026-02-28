package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aetherwing-io/fcp-terraform/internal/fcpcore"
	"github.com/aetherwing-io/fcp-terraform/internal/terraform"
)

func main() {
	// Create the Terraform session and adapter
	session, adapter := terraform.NewTerraformSession()

	// Build the reference card
	registry := fcpcore.NewVerbRegistry()
	registry.RegisterMany(terraform.TerraformVerbSpecs())
	refCard := registry.GenerateReferenceCard(terraform.ExtraSections())

	// Create MCP server
	s := server.NewMCPServer(
		"fcp-terraform",
		"0.1.0",
		server.WithToolCapabilities(false),
	)

	// ── terraform tool (batch mutations) ──────────────────
	terraformTool := mcp.NewTool("terraform",
		mcp.WithDescription("Execute terraform operations. Each op string follows the FCP verb DSL.\n\n"+refCard),
		mcp.WithArray("ops",
			mcp.Required(),
			mcp.Description("Array of operation strings"),
		),
	)
	s.AddTool(terraformTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if session.Model == nil {
			return mcp.NewToolResultText("ERROR: No session. Use terraform_session to create one first."), nil
		}

		model, ok := session.Model.(*terraform.TerraformModel)
		if !ok {
			return mcp.NewToolResultText("ERROR: Invalid model type"), nil
		}

		args := req.GetArguments()
		opsRaw, ok2 := args["ops"]
		if !ok2 {
			return mcp.NewToolResultText("ERROR: ops parameter required"), nil
		}

		opsSlice, ok2 := opsRaw.([]interface{})
		if !ok2 {
			return mcp.NewToolResultText("ERROR: ops must be an array of strings"), nil
		}

		var results []string
		for _, opRaw := range opsSlice {
			opStr, ok := opRaw.(string)
			if !ok {
				results = append(results, "ERROR: each op must be a string")
				continue
			}

			parsed := fcpcore.ParseOp(opStr)
			if parsed.Err != nil {
				results = append(results, fmt.Sprintf("ERROR: %s", parsed.Err.Error))
				continue
			}

			result, event := adapter.DispatchOp(parsed.Op, model)
			session.Log.Append(event)
			results = append(results, result)
		}

		return mcp.NewToolResultText(strings.Join(results, "\n")), nil
	})

	// ── terraform_query tool (read-only) ──────────────────
	queryTool := mcp.NewTool("terraform_query",
		mcp.WithDescription("Query terraform state. Read-only.\n\nQueries: plan, graph, describe LABEL, stats, map, status, history [N], list [@selector], find TEXT"),
		mcp.WithString("q",
			mcp.Required(),
			mcp.Description("Query string"),
		),
	)
	s.AddTool(queryTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if session.Model == nil {
			return mcp.NewToolResultText("ERROR: No session. Use terraform_session to create one first."), nil
		}

		model, ok := session.Model.(*terraform.TerraformModel)
		if !ok {
			return mcp.NewToolResultText("ERROR: Invalid model type"), nil
		}

		q := req.GetString("q", "")
		result := terraform.DispatchQuery(q, model, session.Log)
		return mcp.NewToolResultText(result), nil
	})

	// ── terraform_session tool (lifecycle) ────────────────
	sessionTool := mcp.NewTool("terraform_session",
		mcp.WithDescription("terraform lifecycle: new, open, save, checkpoint, undo, redo.\n\nExamples:\n  new \"My Infrastructure\"\n  open ./main.tf\n  save\n  save as:./output.tf\n  checkpoint v1\n  undo\n  undo to:v1\n  redo"),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action: 'new \"Title\"', 'open ./file', 'save', 'save as:./out', 'checkpoint v1', 'undo', 'undo to:v1', 'redo'"),
		),
	)
	s.AddTool(sessionTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := req.GetString("action", "")
		result := session.Dispatch(action)
		return mcp.NewToolResultText(result), nil
	})

	// ── terraform_help tool (reference card) ──────────────
	helpTool := mcp.NewTool("terraform_help",
		mcp.WithDescription("Returns the terraform FCP reference card."),
	)
	s.AddTool(helpTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(refCard), nil
	})

	// Start stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
