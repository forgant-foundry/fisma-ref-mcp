package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/store"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	version     = "0.1.0"
	serviceName = "fisma-ref-mcp"
)

// NewServer creates an MCP server with all FISMA reference tools registered.
func NewServer(st *store.Store) *server.MCPServer {
	s := server.NewMCPServer(serviceName, version,
		server.WithToolCapabilities(false),
	)
	registerTools(s, st)
	return s
}

// ServeHTTP starts the streamable HTTP MCP transport on addr (e.g., ":8080").
func ServeHTTP(ctx context.Context, s *server.MCPServer, addr string) error {
	h := server.NewStreamableHTTPServer(s)
	srv := &http.Server{Addr: addr, Handler: h}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// ServeStdio starts the stdio MCP transport (for Claude Desktop and similar clients).
func ServeStdio(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

func registerTools(s *server.MCPServer, st *store.Store) {
	s.AddTool(
		mcp.NewTool("search_controls",
			mcp.WithDescription("Semantic search across NIST SP 800-53 Rev 5 control text. Returns ranked controls matching the query."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(`Natural-language description of what you are looking for, e.g. "multi-factor authentication" or "audit log retention".`),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default 10, max 50)."),
			),
			mcp.WithString("family",
				mcp.Description(`Optional control family filter, e.g. "AC" for Access Control.`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleSearch(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("get_control",
			mcp.WithDescription("Retrieve the full text of a specific NIST SP 800-53 Rev 5 control by its identifier."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description(`Control identifier, e.g. "AC-1", "SI-3", or "AC-2(1)" for enhancements.`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetControl(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("list_families",
			mcp.WithDescription("List all NIST SP 800-53 Rev 5 control families with their IDs and titles."),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFamilies(ctx, st)
		},
	)

	s.AddTool(
		mcp.NewTool("get_family",
			mcp.WithDescription("List all base controls (excluding enhancements) within a NIST SP 800-53 Rev 5 control family."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description(`Two-letter family identifier, e.g. "AC" for Access Control or "SI" for System and Information Integrity.`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFamily(ctx, st, req)
		},
	)
}

func handleSearch(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return nil, err
	}

	limit := req.GetInt("limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	results, err := st.SearchControls(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	if family := req.GetString("family", ""); family != "" {
		family = strings.ToUpper(family)
		filtered := results[:0]
		for _, r := range results {
			if r.Control.FamilyID == family {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	return jsonResult(results)
}

func handleGetControl(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return nil, err
	}

	c, err := st.GetControl(ctx, id)
	if err != nil {
		return nil, err
	}
	return jsonResult(c)
}

func handleListFamilies(ctx context.Context, st *store.Store) (*mcp.CallToolResult, error) {
	families, err := st.ListFamilies(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(families)
}

func handleGetFamily(ctx context.Context, st *store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return nil, err
	}

	controls, err := st.GetFamily(ctx, id)
	if err != nil {
		return nil, err
	}
	return jsonResult(controls)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}
