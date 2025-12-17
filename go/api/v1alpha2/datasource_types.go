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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types for DataSource
const (
	// DataSourceConditionTypeConnected indicates whether the data source connection is established
	DataSourceConditionTypeConnected = "Connected"
	// DataSourceConditionTypeReady indicates whether the DataSource is fully reconciled and the ToolServer is created
	DataSourceConditionTypeReady = "Ready"
)

// DataSourceProvider represents the data source provider type.
// Currently only Databricks is supported, but designed to be extensible
// for future data platforms like Snowflake, BigQuery, etc.
// +kubebuilder:validation:Enum=Databricks
type DataSourceProvider string

const (
	DataSourceProviderDatabricks DataSourceProvider = "Databricks"
)

// DatabricksConfig contains Databricks-specific connection settings.
// This follows the same pattern as ModelConfig's provider-specific configs.
type DatabricksConfig struct {
	// WorkspaceURL is the Databricks workspace URL.
	// Example: https://mycompany.cloud.databricks.com
	// +kubebuilder:validation:MinLength=1
	WorkspaceURL string `json:"workspaceUrl"`

	// CredentialsSecretRef is the name of the Secret containing the Databricks token.
	// The secret must exist in the same namespace as the DataSource.
	// +kubebuilder:validation:MinLength=1
	CredentialsSecretRef string `json:"credentialsSecretRef"`

	// CredentialsSecretKey is the key within the secret that contains the token.
	// +kubebuilder:validation:MinLength=1
	CredentialsSecretKey string `json:"credentialsSecretKey"`

	// Catalog is the Unity Catalog name to use.
	// +kubebuilder:validation:MinLength=1
	Catalog string `json:"catalog"`

	// Schema optionally limits discovery to a specific schema within the catalog.
	// If not set, all schemas in the catalog are searched for tables.
	// +optional
	Schema string `json:"schema,omitempty"`

	// WarehouseID is the SQL Warehouse ID for executing queries.
	// If not set, serverless SQL will be used (requires serverless SQL to be enabled).
	// +optional
	WarehouseID string `json:"warehouseId,omitempty"`
}

// SemanticModelRef references a semantic model to expose via the MCP server.
// Users select these from the discovered models shown in status.availableModels.
type SemanticModelRef struct {
	// Name is the semantic model name as it appears in Unity Catalog.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is a human-readable description for the MCP tool.
	// If not provided, the description from Unity Catalog will be used.
	// +optional
	Description string `json:"description,omitempty"`
}

// DiscoveredModel represents a semantic model found in Unity Catalog.
// These are populated by the controller during reconciliation and displayed
// in the UI for users to select which models to expose.
type DiscoveredModel struct {
	// Name is the semantic model name
	Name string `json:"name"`

	// Catalog is the Unity Catalog containing the model
	Catalog string `json:"catalog"`

	// Schema is the schema containing the model
	Schema string `json:"schema"`

	// Description is the model description from Unity Catalog
	// +optional
	Description string `json:"description,omitempty"`
}

// DataSourceSpec defines the desired state of DataSource.
// A DataSource represents a connection to a data fabric (e.g., Databricks)
// and the semantic models to expose to agents via an auto-generated ToolServer.
//
// +kubebuilder:validation:XValidation:rule="self.provider == 'Databricks' && has(self.databricks)",message="databricks config is required when provider is Databricks"
// +kubebuilder:validation:XValidation:rule="!(has(self.databricks) && self.provider != 'Databricks')",message="databricks config must be nil if the provider is not Databricks"
type DataSourceSpec struct {
	// Provider specifies the data platform type.
	// Currently only Databricks is supported.
	// +kubebuilder:default=Databricks
	Provider DataSourceProvider `json:"provider"`

	// Databricks contains Databricks-specific configuration.
	// Required when provider is Databricks.
	// +optional
	Databricks *DatabricksConfig `json:"databricks,omitempty"`

	// SemanticModels is the list of semantic models to expose via the MCP server.
	// If empty, all discovered models from the catalog/schema will be exposed.
	// Users can select specific models after seeing what's available in status.availableModels.
	// +optional
	SemanticModels []SemanticModelRef `json:"semanticModels,omitempty"`
}

// DataSourceStatus defines the observed state of DataSource.
// This is updated by the controller during reconciliation.
type DataSourceStatus struct {
	// ObservedGeneration is the generation of the spec that was last processed.
	// Used to detect if the spec has changed since last reconciliation.
	ObservedGeneration int64 `json:"observedGeneration"`

	// Conditions represent the latest observations of the DataSource's state.
	// Condition types:
	// - Connected: whether we can reach the data source (e.g., Databricks workspace)
	// - Ready: whether the MCP server resources have been created successfully
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// AvailableModels lists all semantic models discovered from the data source.
	// The UI uses this to show users what models they can select in spec.semanticModels.
	// +optional
	AvailableModels []DiscoveredModel `json:"availableModels,omitempty"`

	// GeneratedMCPServer is the name of the auto-generated RemoteMCPServer.
	// Agents reference this RemoteMCPServer to access the data source tools.
	// The format is "{datasource-name}-mcp".
	// +optional
	GeneratedMCPServer string `json:"generatedMCPServer,omitempty"`

	// SecretHash stores a hash of the credentials secret to detect changes.
	// When the secret changes, the controller will re-reconcile to update the MCP server.
	// +optional
	SecretHash string `json:"secretHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=kagent,shortName=ds
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider"
// +kubebuilder:printcolumn:name="Connected",type="string",JSONPath=".status.conditions[?(@.type=='Connected')].status"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="MCPServer",type="string",JSONPath=".status.generatedMCPServer"

// DataSource is the Schema for the datasources API.
// It represents a connection to a data fabric (e.g., Databricks) and enables
// agents to query semantic models through an auto-generated HTTP MCP server.
type DataSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataSourceSpec   `json:"spec,omitempty"`
	Status DataSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataSourceList contains a list of DataSource.
type DataSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataSource{}, &DataSourceList{})
}
