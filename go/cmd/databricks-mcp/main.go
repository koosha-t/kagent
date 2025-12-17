/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var (
	workspaceURL = flag.String("workspace-url", "", "Databricks workspace URL (required)")
	catalog      = flag.String("catalog", "", "Unity Catalog name (required)")
	schema       = flag.String("schema", "", "Schema name (optional)")
	warehouseID  = flag.String("warehouse-id", "", "SQL Warehouse ID for executing queries (optional, uses serverless if not set)")
	models       = flag.String("models", "", "Comma-separated list of semantic models/tables to expose (if empty, discovers all)")
	transport    = flag.String("transport", "stdio", "Transport mode: stdio or streamable-http")
	httpPort     = flag.Int("port", 8080, "HTTP port when using streamable-http transport")
)

func main() {
	flag.Parse()

	// Validate required flags
	if *workspaceURL == "" {
		log.Fatal("--workspace-url is required")
	}
	if *catalog == "" {
		log.Fatal("--catalog is required")
	}

	// Get token from environment (injected by the ToolServer from secret)
	token := os.Getenv("DATABRICKS_TOKEN")
	if token == "" {
		log.Fatal("DATABRICKS_TOKEN environment variable is required")
	}

	// Parse model names
	var modelNames []string
	if *models != "" {
		modelNames = strings.Split(*models, ",")
		for i := range modelNames {
			modelNames[i] = strings.TrimSpace(modelNames[i])
		}
	}

	// Create Databricks client
	client := NewDatabricksClient(*workspaceURL, token, *catalog, *schema, *warehouseID)

	// Test connection on startup
	log.Println("Testing Databricks connection...")
	if err := client.TestConnection(context.Background()); err != nil {
		log.Fatalf("Failed to connect to Databricks: %v", err)
	}
	log.Println("Databricks connection successful")

	// Create MCP server
	s := server.NewMCPServer(
		"databricks-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	registerTools(s, client, modelNames)

	// Start server based on transport mode
	switch *transport {
	case "streamable-http":
		addr := fmt.Sprintf(":%d", *httpPort)
		log.Printf("Starting streamable-http server on %s", addr)
		httpServer := server.NewStreamableHTTPServer(s)

		// Create a mux to handle both health checks and MCP requests
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		mux.Handle("/mcp", httpServer)

		srv := &http.Server{
			Addr:    addr,
			Handler: mux,
		}
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	default: // stdio
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// DatabricksClient handles communication with Databricks Unity Catalog and SQL API
type DatabricksClient struct {
	workspaceURL string
	token        string
	catalog      string
	schema       string
	warehouseID  string
	httpClient   *http.Client
}

// NewDatabricksClient creates a new Databricks client
func NewDatabricksClient(workspaceURL, token, catalog, schema, warehouseID string) *DatabricksClient {
	// Normalize workspace URL (remove trailing slash if present)
	workspaceURL = strings.TrimSuffix(workspaceURL, "/")
	return &DatabricksClient{
		workspaceURL: workspaceURL,
		token:        token,
		catalog:      catalog,
		schema:       schema,
		warehouseID:  warehouseID,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// doRequest performs an authenticated HTTP request to Databricks
func (c *DatabricksClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	url := c.workspaceURL + path

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// TestConnection verifies the connection to Databricks by checking catalog access
func (c *DatabricksClient) TestConnection(ctx context.Context) error {
	path := fmt.Sprintf("/api/2.1/unity-catalog/catalogs/%s", c.catalog)
	_, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to access catalog %q: %w", c.catalog, err)
	}
	return nil
}

// TableInfo represents a table/view from Unity Catalog
type TableInfo struct {
	Name        string `json:"name"`
	CatalogName string `json:"catalog_name"`
	SchemaName  string `json:"schema_name"`
	TableType   string `json:"table_type"`
	Comment     string `json:"comment,omitempty"`
}

// ListTablesResponse represents the response from the list tables API
type ListTablesResponse struct {
	Tables        []TableInfo `json:"tables"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

// ListTables lists tables/views in the configured catalog/schema
func (c *DatabricksClient) ListTables(ctx context.Context) ([]TableInfo, error) {
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables?catalog_name=%s", c.catalog)
	if c.schema != "" {
		path += fmt.Sprintf("&schema_name=%s", c.schema)
	}
	path += "&max_results=100"

	var allTables []TableInfo
	for {
		respBody, err := c.doRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp ListTablesResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		allTables = append(allTables, resp.Tables...)

		if resp.NextPageToken == "" {
			break
		}
		path = fmt.Sprintf("/api/2.1/unity-catalog/tables?catalog_name=%s&page_token=%s", c.catalog, resp.NextPageToken)
	}

	return allTables, nil
}

// ColumnInfo represents a column from Unity Catalog
type ColumnInfo struct {
	Name         string `json:"name"`
	TypeName     string `json:"type_name"`
	TypeText     string `json:"type_text"`
	Position     int    `json:"position"`
	Comment      string `json:"comment,omitempty"`
	Nullable     bool   `json:"nullable"`
	PartitionKey bool   `json:"partition_index,omitempty"`
}

// TableDetails represents detailed table information
type TableDetails struct {
	Name        string       `json:"name"`
	CatalogName string       `json:"catalog_name"`
	SchemaName  string       `json:"schema_name"`
	TableType   string       `json:"table_type"`
	Comment     string       `json:"comment,omitempty"`
	Columns     []ColumnInfo `json:"columns"`
	Properties  map[string]string `json:"properties,omitempty"`
	CreatedAt   int64        `json:"created_at,omitempty"`
	UpdatedAt   int64        `json:"updated_at,omitempty"`
}

// GetTable retrieves detailed information about a specific table
func (c *DatabricksClient) GetTable(ctx context.Context, tableName string) (*TableDetails, error) {
	fullName := c.getFullTableName(tableName)
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%s", fullName)

	respBody, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var table TableDetails
	if err := json.Unmarshal(respBody, &table); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &table, nil
}

// getFullTableName constructs the full table name (catalog.schema.table)
func (c *DatabricksClient) getFullTableName(tableName string) string {
	// If tableName already contains dots, assume it's fully qualified
	if strings.Contains(tableName, ".") {
		return tableName
	}
	if c.schema != "" {
		return fmt.Sprintf("%s.%s.%s", c.catalog, c.schema, tableName)
	}
	// If no schema specified, we need to include it
	return tableName
}

// SQLStatementRequest represents a SQL execution request
type SQLStatementRequest struct {
	WarehouseID string `json:"warehouse_id,omitempty"`
	Catalog     string `json:"catalog,omitempty"`
	Schema      string `json:"schema,omitempty"`
	Statement   string `json:"statement"`
	WaitTimeout string `json:"wait_timeout,omitempty"`
	RowLimit    int    `json:"row_limit,omitempty"`
	Format      string `json:"format,omitempty"`
	Disposition string `json:"disposition,omitempty"`
}

// SQLStatementResponse represents the response from SQL execution
type SQLStatementResponse struct {
	StatementID string `json:"statement_id"`
	Status      struct {
		State string `json:"state"`
		Error *struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		} `json:"error,omitempty"`
	} `json:"status"`
	Manifest *struct {
		Format     string `json:"format"`
		Schema     *SQLSchema `json:"schema"`
		TotalRowCount int64 `json:"total_row_count"`
		TruncatedByRowLimit bool `json:"truncated"`
	} `json:"manifest,omitempty"`
	Result *struct {
		RowCount   int64           `json:"row_count"`
		DataArray  [][]any         `json:"data_array,omitempty"`
		Chunk      json.RawMessage `json:"chunk_index,omitempty"`
	} `json:"result,omitempty"`
}

// SQLSchema represents the schema of SQL results
type SQLSchema struct {
	ColumnCount int `json:"column_count"`
	Columns     []struct {
		Name     string `json:"name"`
		TypeName string `json:"type_name"`
		TypeText string `json:"type_text"`
		Position int    `json:"position"`
	} `json:"columns"`
}

// ExecuteSQL executes a SQL statement against the Databricks warehouse
func (c *DatabricksClient) ExecuteSQL(ctx context.Context, sql string, maxRows int) (*SQLStatementResponse, error) {
	req := SQLStatementRequest{
		Statement:   sql,
		Catalog:     c.catalog,
		WaitTimeout: "30s",
		Format:      "JSON_ARRAY",
		Disposition: "INLINE",
	}

	if c.schema != "" {
		req.Schema = c.schema
	}

	if c.warehouseID != "" {
		req.WarehouseID = c.warehouseID
	}

	if maxRows > 0 {
		req.RowLimit = maxRows
	}

	respBody, err := c.doRequest(ctx, "POST", "/api/2.0/sql/statements", req)
	if err != nil {
		return nil, err
	}

	var resp SQLStatementResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for execution errors
	if resp.Status.Error != nil {
		return nil, fmt.Errorf("SQL execution error (%s): %s", resp.Status.Error.ErrorCode, resp.Status.Error.Message)
	}

	// If the statement is still running, poll for completion
	if resp.Status.State == "PENDING" || resp.Status.State == "RUNNING" {
		return c.pollStatementStatus(ctx, resp.StatementID)
	}

	return &resp, nil
}

// pollStatementStatus polls for statement completion
func (c *DatabricksClient) pollStatementStatus(ctx context.Context, statementID string) (*SQLStatementResponse, error) {
	path := fmt.Sprintf("/api/2.0/sql/statements/%s", statementID)

	for i := 0; i < 60; i++ { // Poll for up to 60 seconds
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}

		respBody, err := c.doRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var resp SQLStatementResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if resp.Status.Error != nil {
			return nil, fmt.Errorf("SQL execution error (%s): %s", resp.Status.Error.ErrorCode, resp.Status.Error.Message)
		}

		if resp.Status.State == "SUCCEEDED" {
			return &resp, nil
		}

		if resp.Status.State == "FAILED" || resp.Status.State == "CANCELED" || resp.Status.State == "CLOSED" {
			return nil, fmt.Errorf("SQL statement %s", resp.Status.State)
		}
	}

	return nil, fmt.Errorf("statement execution timed out")
}

// registerTools registers all MCP tools with the server
func registerTools(s *server.MCPServer, client *DatabricksClient, modelNames []string) {
	// Tool 1: list_tables - List available tables in the catalog/schema
	s.AddTool(
		mcp.NewTool("list_tables",
			mcp.WithDescription("List all available tables and views in the configured Unity Catalog and schema"),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tables, err := client.ListTables(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list tables: %w", err)
			}

			// Filter tables if specific models are configured
			if len(modelNames) > 0 {
				var filtered []TableInfo
				for _, t := range tables {
					if containsModel(modelNames, t.Name) {
						filtered = append(filtered, t)
					}
				}
				tables = filtered
			}

			result := make([]map[string]string, len(tables))
			for i, t := range tables {
				result[i] = map[string]string{
					"name":        t.Name,
					"catalog":     t.CatalogName,
					"schema":      t.SchemaName,
					"type":        t.TableType,
					"description": t.Comment,
				}
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Tool 2: describe_table - Get schema/metadata for a table
	s.AddTool(
		mcp.NewTool("describe_table",
			mcp.WithDescription("Get detailed schema and metadata for a table, including columns, types, and descriptions"),
			mcp.WithString("table_name",
				mcp.Required(),
				mcp.Description("Name of the table to describe (can be just the table name or fully qualified catalog.schema.table)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tableName, err := request.RequireString("table_name")
			if err != nil {
				return nil, fmt.Errorf("table_name is required: %w", err)
			}

			// Validate table is in the allowed list (if restrictions are set)
			if len(modelNames) > 0 && !containsModel(modelNames, tableName) {
				return nil, fmt.Errorf("table %q is not available. Use list_tables to see available tables", tableName)
			}

			table, err := client.GetTable(ctx, tableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get table details: %w", err)
			}

			// Format columns for output
			columns := make([]map[string]any, len(table.Columns))
			for i, col := range table.Columns {
				columns[i] = map[string]any{
					"name":        col.Name,
					"type":        col.TypeText,
					"nullable":    col.Nullable,
					"description": col.Comment,
					"position":    col.Position,
				}
			}

			result := map[string]any{
				"name":        table.Name,
				"catalog":     table.CatalogName,
				"schema":      table.SchemaName,
				"type":        table.TableType,
				"description": table.Comment,
				"columns":     columns,
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Tool 3: execute_sql - Execute SQL against the Databricks warehouse
	s.AddTool(
		mcp.NewTool("execute_sql",
			mcp.WithDescription("Execute a SQL query against the Databricks SQL warehouse. Returns results as a JSON array."),
			mcp.WithString("sql",
				mcp.Required(),
				mcp.Description("SQL query to execute"),
			),
			mcp.WithNumber("max_rows",
				mcp.Description("Maximum number of rows to return (default: 100, max: 10000)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sql, err := request.RequireString("sql")
			if err != nil {
				return nil, fmt.Errorf("sql is required: %w", err)
			}

			maxRows := request.GetInt("max_rows", 100)
			if maxRows > 10000 {
				maxRows = 10000
			}
			if maxRows <= 0 {
				maxRows = 100
			}

			resp, err := client.ExecuteSQL(ctx, sql, maxRows)
			if err != nil {
				return nil, fmt.Errorf("failed to execute SQL: %w", err)
			}

			// Build result
			result := map[string]any{
				"status":   resp.Status.State,
				"row_count": 0,
				"columns":  []string{},
				"data":     [][]any{},
			}

			if resp.Manifest != nil && resp.Manifest.Schema != nil {
				columnNames := make([]string, len(resp.Manifest.Schema.Columns))
				for i, col := range resp.Manifest.Schema.Columns {
					columnNames[i] = col.Name
				}
				result["columns"] = columnNames
				result["total_row_count"] = resp.Manifest.TotalRowCount
				result["truncated"] = resp.Manifest.TruncatedByRowLimit
			}

			if resp.Result != nil {
				result["row_count"] = resp.Result.RowCount
				if resp.Result.DataArray != nil {
					result["data"] = resp.Result.DataArray
				}
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Tool 4: get_sample_data - Get sample rows from a table
	s.AddTool(
		mcp.NewTool("get_sample_data",
			mcp.WithDescription("Get sample data from a table to understand its structure and content"),
			mcp.WithString("table_name",
				mcp.Required(),
				mcp.Description("Name of the table to sample"),
			),
			mcp.WithNumber("num_rows",
				mcp.Description("Number of sample rows to return (default: 10, max: 100)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tableName, err := request.RequireString("table_name")
			if err != nil {
				return nil, fmt.Errorf("table_name is required: %w", err)
			}

			numRows := request.GetInt("num_rows", 10)
			if numRows > 100 {
				numRows = 100
			}
			if numRows <= 0 {
				numRows = 10
			}

			// Validate table is in the allowed list (if restrictions are set)
			if len(modelNames) > 0 && !containsModel(modelNames, tableName) {
				return nil, fmt.Errorf("table %q is not available. Use list_tables to see available tables", tableName)
			}

			fullTableName := client.getFullTableName(tableName)
			sql := fmt.Sprintf("SELECT * FROM %s LIMIT %d", fullTableName, numRows)

			resp, err := client.ExecuteSQL(ctx, sql, numRows)
			if err != nil {
				return nil, fmt.Errorf("failed to get sample data: %w", err)
			}

			// Build result
			result := map[string]any{
				"table":    tableName,
				"columns":  []string{},
				"data":     [][]any{},
				"row_count": 0,
			}

			if resp.Manifest != nil && resp.Manifest.Schema != nil {
				columnNames := make([]string, len(resp.Manifest.Schema.Columns))
				for i, col := range resp.Manifest.Schema.Columns {
					columnNames[i] = col.Name
				}
				result["columns"] = columnNames
			}

			if resp.Result != nil {
				result["row_count"] = resp.Result.RowCount
				if resp.Result.DataArray != nil {
					result["data"] = resp.Result.DataArray
				}
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Tool 5: get_table_stats - Get statistics about a table
	s.AddTool(
		mcp.NewTool("get_table_stats",
			mcp.WithDescription("Get basic statistics about a table including row count and column statistics"),
			mcp.WithString("table_name",
				mcp.Required(),
				mcp.Description("Name of the table to analyze"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tableName, err := request.RequireString("table_name")
			if err != nil {
				return nil, fmt.Errorf("table_name is required: %w", err)
			}

			// Validate table is in the allowed list (if restrictions are set)
			if len(modelNames) > 0 && !containsModel(modelNames, tableName) {
				return nil, fmt.Errorf("table %q is not available. Use list_tables to see available tables", tableName)
			}

			fullTableName := client.getFullTableName(tableName)
			sql := fmt.Sprintf("SELECT COUNT(*) as row_count FROM %s", fullTableName)

			resp, err := client.ExecuteSQL(ctx, sql, 1)
			if err != nil {
				return nil, fmt.Errorf("failed to get table stats: %w", err)
			}

			rowCount := int64(0)
			if resp.Result != nil && len(resp.Result.DataArray) > 0 && len(resp.Result.DataArray[0]) > 0 {
				if count, ok := resp.Result.DataArray[0][0].(float64); ok {
					rowCount = int64(count)
				}
			}

			// Get table metadata for additional info
			table, err := client.GetTable(ctx, tableName)
			if err != nil {
				return nil, fmt.Errorf("failed to get table details: %w", err)
			}

			result := map[string]any{
				"table":        tableName,
				"row_count":    rowCount,
				"column_count": len(table.Columns),
				"table_type":   table.TableType,
				"columns":      make([]map[string]string, len(table.Columns)),
			}

			for i, col := range table.Columns {
				result["columns"].([]map[string]string)[i] = map[string]string{
					"name": col.Name,
					"type": col.TypeText,
				}
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)

	// Tool 6: search_tables - Search for tables by name or description
	s.AddTool(
		mcp.NewTool("search_tables",
			mcp.WithDescription("Search for tables in the catalog by name pattern"),
			mcp.WithString("pattern",
				mcp.Required(),
				mcp.Description("Search pattern (case-insensitive substring match on table name)"),
			),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			pattern, err := request.RequireString("pattern")
			if err != nil {
				return nil, fmt.Errorf("pattern is required: %w", err)
			}

			tables, err := client.ListTables(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list tables: %w", err)
			}

			// Filter by pattern and model restrictions
			pattern = strings.ToLower(pattern)
			var matches []TableInfo
			for _, t := range tables {
				if strings.Contains(strings.ToLower(t.Name), pattern) ||
				   strings.Contains(strings.ToLower(t.Comment), pattern) {
					if len(modelNames) == 0 || containsModel(modelNames, t.Name) {
						matches = append(matches, t)
					}
				}
			}

			result := make([]map[string]string, len(matches))
			for i, t := range matches {
				result[i] = map[string]string{
					"name":        t.Name,
					"catalog":     t.CatalogName,
					"schema":      t.SchemaName,
					"type":        t.TableType,
					"description": t.Comment,
				}
			}

			jsonBytes, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("failed to marshal result: %w", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		},
	)
}

// containsModel checks if the model/table name is in the allowed list
func containsModel(models []string, name string) bool {
	// If no models specified, all are allowed
	if len(models) == 0 {
		return true
	}
	// Also check just the table name without catalog.schema prefix
	baseName := name
	if parts := strings.Split(name, "."); len(parts) > 0 {
		baseName = parts[len(parts)-1]
	}
	for _, m := range models {
		if m == name || m == baseName {
			return true
		}
	}
	return false
}
