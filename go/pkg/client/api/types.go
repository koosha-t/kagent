package api

import (
	"github.com/kagent-dev/kagent/go/api/v1alpha1"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/internal/database"
)

// Common types

// APIError represents an error response from the API
type APIError struct {
	Error string `json:"error"`
}

func NewResponse[T any](data T, message string, error bool) StandardResponse[T] {
	return StandardResponse[T]{
		Error:   error,
		Data:    data,
		Message: message,
	}
}

// StandardResponse represents the standard response format used by many endpoints
type StandardResponse[T any] struct {
	Error   bool   `json:"error"`
	Data    T      `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

// Provider represents a provider configuration
type Provider struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Version represents the version information
type VersionResponse struct {
	KAgentVersion string `json:"kagent_version"`
	GitCommit     string `json:"git_commit"`
	BuildDate     string `json:"build_date"`
}

// ModelConfigResponse represents a model configuration response
type ModelConfigResponse struct {
	Ref             string              `json:"ref"`
	ProviderName    string              `json:"providerName"`
	Model           string              `json:"model"`
	APIKeySecret    string              `json:"apiKeySecret"`
	APIKeySecretKey string              `json:"apiKeySecretKey"`
	ModelParams     map[string]any      `json:"modelParams"`
	TLS             *v1alpha2.TLSConfig `json:"tls,omitempty"`
}

// CreateModelConfigRequest represents a request to create a model configuration
type CreateModelConfigRequest struct {
	Ref                     string                            `json:"ref"`
	Provider                Provider                          `json:"provider"`
	Model                   string                            `json:"model"`
	APIKey                  string                            `json:"apiKey"`
	OpenAIParams            *v1alpha2.OpenAIConfig            `json:"openAI,omitempty"`
	AnthropicParams         *v1alpha2.AnthropicConfig         `json:"anthropic,omitempty"`
	AzureParams             *v1alpha2.AzureOpenAIConfig       `json:"azureOpenAI,omitempty"`
	OllamaParams            *v1alpha2.OllamaConfig            `json:"ollama,omitempty"`
	GeminiParams            *v1alpha2.GeminiConfig            `json:"gemini,omitempty"`
	GeminiVertexAIParams    *v1alpha2.GeminiVertexAIConfig    `json:"geminiVertexAI,omitempty"`
	AnthropicVertexAIParams *v1alpha2.AnthropicVertexAIConfig `json:"anthropicVertexAI,omitempty"`
}

// UpdateModelConfigRequest represents a request to update a model configuration
type UpdateModelConfigRequest struct {
	Provider                Provider                          `json:"provider"`
	Model                   string                            `json:"model"`
	APIKey                  *string                           `json:"apiKey,omitempty"`
	OpenAIParams            *v1alpha2.OpenAIConfig            `json:"openAI,omitempty"`
	AnthropicParams         *v1alpha2.AnthropicConfig         `json:"anthropic,omitempty"`
	AzureParams             *v1alpha2.AzureOpenAIConfig       `json:"azureOpenAI,omitempty"`
	OllamaParams            *v1alpha2.OllamaConfig            `json:"ollama,omitempty"`
	GeminiParams            *v1alpha2.GeminiConfig            `json:"gemini,omitempty"`
	GeminiVertexAIParams    *v1alpha2.GeminiVertexAIConfig    `json:"geminiVertexAI,omitempty"`
	AnthropicVertexAIParams *v1alpha2.AnthropicVertexAIConfig `json:"anthropicVertexAI,omitempty"`
}

// Agent types

type AgentResponse struct {
	ID    string          `json:"id"`
	Agent *v1alpha2.Agent `json:"agent"`
	// Config         *adk.AgentConfig       `json:"config"`
	ModelProvider   v1alpha2.ModelProvider `json:"modelProvider"`
	Model           string                 `json:"model"`
	ModelConfigRef  string                 `json:"modelConfigRef"`
	MemoryRefs      []string               `json:"memoryRefs"`
	Tools           []*v1alpha2.Tool       `json:"tools"`
	DeploymentReady bool                   `json:"deploymentReady"`
	Accepted        bool                   `json:"accepted"`
}

// Session types

// SessionRequest represents a session creation/update request
type SessionRequest struct {
	AgentRef *string `json:"agent_ref,omitempty"`
	Name     *string `json:"name,omitempty"`
	ID       *string `json:"id,omitempty"`
}

// Run types

// RunRequest represents a run creation request
type RunRequest struct {
	Task string `json:"task"`
}

// Run represents a run from the database
type Task = database.Task

// Message represents a message from the database
type Message = database.Event

// Session represents a session from the database
type Session = database.Session

// Agent represents an agent from the database
type Agent = database.Agent

// Tool types

// Tool represents a tool from the database
type Tool = database.Tool

// Feedback represents a feedback from the database
type Feedback = database.Feedback

// ToolServer types

// ToolServerResponse represents a tool server response
type ToolServerResponse struct {
	Ref             string              `json:"ref"`
	GroupKind       string              `json:"groupKind"`
	DiscoveredTools []*v1alpha2.MCPTool `json:"discoveredTools"`
}

// Memory types

// MemoryResponse represents a memory response
type MemoryResponse struct {
	Ref             string         `json:"ref"`
	ProviderName    string         `json:"providerName"`
	APIKeySecretRef string         `json:"apiKeySecretRef"`
	APIKeySecretKey string         `json:"apiKeySecretKey"`
	MemoryParams    map[string]any `json:"memoryParams"`
}

// CreateMemoryRequest represents a request to create a memory
type CreateMemoryRequest struct {
	Ref            string                   `json:"ref"`
	Provider       Provider                 `json:"provider"`
	APIKey         string                   `json:"apiKey"`
	PineconeParams *v1alpha1.PineconeConfig `json:"pinecone,omitempty"`
}

// UpdateMemoryRequest represents a request to update a memory
type UpdateMemoryRequest struct {
	PineconeParams *v1alpha1.PineconeConfig `json:"pinecone,omitempty"`
}

// Namespace types

// NamespaceResponse represents a namespace response
type NamespaceResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Provider types

// ProviderInfo represents information about a provider
type ProviderInfo struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"`
	RequiredParams []string `json:"requiredParams"`
	OptionalParams []string `json:"optionalParams"`
}

// SessionRunsResponse represents the response for session runs
type SessionRunsResponse struct {
	Status bool `json:"status"`
	Data   any  `json:"data"`
}

// SessionRunsData represents the data part of session runs response
type SessionRunsData struct {
	Runs []any `json:"runs"`
}

// DataSourceResponse represents a data source response.
// This corresponds to the DataSource CRD defined in go/api/v1alpha2/datasource_types.go
type DataSourceResponse struct {
	// Ref is the namespace/name format identifier (e.g., "default/sales-databricks")
	Ref string `json:"ref"`

	// Provider is the data platform type (currently only "Databricks")
	Provider string `json:"provider"`

	// Databricks contains the Databricks-specific connection configuration
	Databricks *v1alpha2.DatabricksConfig `json:"databricks,omitempty"`

	// SemanticModels are the models explicitly selected by the user to expose
	SemanticModels []v1alpha2.SemanticModelRef `json:"semanticModels,omitempty"`

	// AvailableModels are all models discovered from Unity Catalog
	AvailableModels []v1alpha2.DiscoveredModel `json:"availableModels,omitempty"`

	// GeneratedMCPServer is the name of the auto-generated RemoteMCPServer
	GeneratedMCPServer string `json:"generatedMCPServer,omitempty"`

	// Connected indicates if the data source connection is established
	Connected bool `json:"connected"`

	// Ready indicates if the DataSource is fully reconciled (MCP server created)
	Ready bool `json:"ready"`
}

// DatabricksCatalog represents a Databricks Unity Catalog.
type DatabricksCatalog struct {
	// Name is the catalog name
	Name string `json:"name"`
	// Comment is the catalog description
	Comment string `json:"comment,omitempty"`
}

// DatabricksSchema represents a schema within a Databricks catalog.
type DatabricksSchema struct {
	// Name is the schema name
	Name string `json:"name"`
	// Catalog is the parent catalog name
	Catalog string `json:"catalog"`
	// Comment is the schema description
	Comment string `json:"comment,omitempty"`
}

// DatabricksTable represents a table within a Databricks schema.
type DatabricksTable struct {
	// Name is the table name
	Name string `json:"name"`
	// Catalog is the parent catalog name
	Catalog string `json:"catalog"`
	// Schema is the parent schema name
	Schema string `json:"schema"`
	// TableType is the type of table (MANAGED, EXTERNAL, VIEW)
	TableType string `json:"tableType"`
	// Comment is the table description
	Comment string `json:"comment,omitempty"`
}

// CreateDataSourceRequest represents a request to create a new DataSource.
type CreateDataSourceRequest struct {
	// Name is the DataSource name (RFC 1123 compliant)
	Name string `json:"name"`
	// Namespace is the Kubernetes namespace
	Namespace string `json:"namespace"`
	// Catalog is the Databricks Unity Catalog name
	Catalog string `json:"catalog"`
	// Schema is the Databricks schema name
	Schema string `json:"schema"`
	// Tables is the list of table names to expose as semantic models
	Tables []string `json:"tables"`
	// WarehouseID is the optional SQL Warehouse ID
	WarehouseID string `json:"warehouseId,omitempty"`
}
