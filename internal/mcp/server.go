package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/forgant-foundry/fisma-ref-mcp/internal/rel_store"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	version     = "0.1.0"
	serviceName = "fisma-ref-mcp"
)

// NewServer creates an MCP server with all FISMA reference tools registered.
func NewServer(st *rel_store.Store) *server.MCPServer {
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

func registerTools(s *server.MCPServer, st *rel_store.Store) {
	s.AddTool(
		mcp.NewTool("search",
			mcp.WithDescription("Semantic search across all indexed documents — NIST SP 800-53 Rev 5 controls and FY 2025 IG FISMA metrics. Returns ranked results with source provenance."),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description(`Natural-language description of what you are looking for, e.g. "multi-factor authentication" or "identity management maturity".`),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of results to return (default 10, max 50)."),
			),
			mcp.WithString("source",
				mcp.Description(`Optional source filter: "nist_800_53" for SP 800-53 controls, "fisma_fy2025" for FISMA metrics, "nist_csf_v2" for CSF 2.0 subcategories. Omit to search all.`),
			),
			mcp.WithString("family",
				mcp.Description(`Optional NIST control family filter, e.g. "AC". Only applies to nist_800_53 results.`),
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

	s.AddTool(
		mcp.NewTool("list_fisma_metrics",
			mcp.WithDescription("List FY 2025 IG FISMA evaluation metrics. Optionally filter by domain."),
			mcp.WithString("domain",
				mcp.Description(`Optional domain filter, e.g. "Identity Management and Access Control". Omit to list all 35 metrics.`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListFismaMetrics(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("get_fisma_metric",
			mcp.WithDescription("Retrieve a single FY 2025 IG FISMA evaluation metric by its numeric ID, including maturity level descriptions and criteria references."),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("Metric ID (1–35)."),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetFismaMetric(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("list_csf_functions",
			mcp.WithDescription("List the six NIST CSF 2.0 functions (Govern, Identify, Protect, Detect, Respond, Recover) with their categories."),
			mcp.WithString("function",
				mcp.Description(`Optional function ID to filter categories, e.g. "GV" for Govern. Omit to list all functions with all categories.`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleListCSFFunctions(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("get_csf_subcategory",
			mcp.WithDescription("Retrieve a single NIST CSF 2.0 subcategory by its identifier, including the outcome statement and implementation examples."),
			mcp.WithString("id",
				mcp.Required(),
				mcp.Description(`CSF 2.0 subcategory identifier, e.g. "GV.OC-01" or "PR.AA-01".`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetCSFSubcategory(ctx, st, req)
		},
	)

	s.AddTool(
		mcp.NewTool("get_metrics_by_control",
			mcp.WithDescription("Find all FY 2025 IG FISMA metrics that reference a given NIST SP 800-53 control ID."),
			mcp.WithString("control_id",
				mcp.Required(),
				mcp.Description(`NIST SP 800-53 control identifier, e.g. "AC-2" or "SI-3".`),
			),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleGetMetricsByControl(ctx, st, req)
		},
	)
}

func handleSearch(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return nil, err
	}

	limit := req.GetInt("limit", 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	source := req.GetString("source", "")
	results, err := st.Search(ctx, query, limit, source)
	if err != nil {
		return nil, err
	}

	// Family filter applies only to NIST control results.
	if family := strings.ToUpper(req.GetString("family", "")); family != "" {
		filtered := results[:0]
		for _, r := range results {
			if r.Source != "nist_800_53" || strings.HasPrefix(r.ID, family+"-") {
				filtered = append(filtered, r)
			}
		}
		results = filtered
	}

	return jsonResult(results)
}

func handleGetControl(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleListFamilies(ctx context.Context, st *rel_store.Store) (*mcp.CallToolResult, error) {
	families, err := st.ListFamilies(ctx)
	if err != nil {
		return nil, err
	}
	return jsonResult(families)
}

func handleGetFamily(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleListFismaMetrics(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domain := req.GetString("domain", "")
	metrics, err := st.ListFismaMetrics(ctx, domain)
	if err != nil {
		return nil, err
	}
	return jsonResult(metrics)
}

func handleGetFismaMetric(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetInt("id", 0)
	if id <= 0 {
		return nil, fmt.Errorf("id must be a positive integer")
	}
	m, err := st.GetFismaMetric(ctx, id)
	if err != nil {
		return nil, err
	}
	return jsonResult(m)
}

func handleGetMetricsByControl(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	controlID, err := req.RequireString("control_id")
	if err != nil {
		return nil, err
	}
	metrics, err := st.GetMetricsByControl(ctx, controlID)
	if err != nil {
		return nil, err
	}
	return jsonResult(metrics)
}

func handleListCSFFunctions(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	functionID := req.GetString("function", "")

	fns, err := st.ListCSFFunctions(ctx)
	if err != nil {
		return nil, err
	}

	cats, err := st.ListCSFCategories(ctx, functionID)
	if err != nil {
		return nil, err
	}

	// Group categories under their functions.
	type catEntry struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Text  string `json:"text"`
	}
	type fnEntry struct {
		ID         string     `json:"id"`
		Title      string     `json:"title"`
		Text       string     `json:"text"`
		Categories []catEntry `json:"categories"`
	}

	catsByFn := make(map[string][]catEntry)
	for _, c := range cats {
		catsByFn[c.FunctionID] = append(catsByFn[c.FunctionID], catEntry{c.ID, c.Title, c.Text})
	}

	var out []fnEntry
	for _, f := range fns {
		if functionID != "" && strings.ToUpper(functionID) != f.ID {
			continue
		}
		out = append(out, fnEntry{
			ID:         f.ID,
			Title:      f.Title,
			Text:       f.Text,
			Categories: catsByFn[f.ID],
		})
	}
	return jsonResult(out)
}

func handleGetCSFSubcategory(ctx context.Context, st *rel_store.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return nil, err
	}
	s, err := st.GetCSFSubcategory(ctx, id)
	if err != nil {
		return nil, err
	}
	return jsonResult(s)
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}
