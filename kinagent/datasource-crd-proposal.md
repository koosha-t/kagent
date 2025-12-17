# DataSource CRD Proposal

## Overview

This proposal introduces a new `DataSource` Custom Resource Definition (CRD) to Kagent, enabling first-class integration with data fabrics like Databricks. The DataSource CRD allows users to connect to data platforms, discover available semantic models, and expose them to agents via auto-generated MCP ToolServers.

## Motivation

Currently, integrating data platforms with Kagent agents requires manually creating ToolServers with embedded connection logic. This approach has limitations:

- No standardized way to represent data fabric connections
- Manual configuration of credentials and connection details per ToolServer
- No discovery mechanism for available datasets/semantic models
- Difficult to reuse data connections across multiple agents

A dedicated DataSource CRD provides:

- First-class abstraction for data fabric connections
- Automatic discovery of semantic models from Unity Catalog
- Centralized credential management
- Auto-generated ToolServers that agents can reference

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Kagent UI                                   │
│  1. User creates DataSource (points to Databricks workspace)        │
│  2. UI fetches available datasets/semantic models from Unity Catalog│
│  3. User selects which semantic models to expose                    │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    DataSource CRD                                   │
│  - Connection details (workspace URL, credentials)                  │
│  - Selected semantic models to expose                               │
└─────────────────────────────────────────────────────────────────────┘
                                │
                                │ Controller reconciles
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│              Auto-generated ToolServer                              │
│  - Managed by DataSource controller                                 │
│  - Owner reference → DataSource (garbage collected together)        │
│  - Exposes MCP tools: query_model, describe_model, list_models, etc.│
└─────────────────────────────────────────────────────────────────────┘
                                │
                                │ Agent references ToolServer
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Agent                                       │
│  tools:                                                             │
│    - toolServer:                                                    │
│        ref: sales-fabric-mcp  # auto-generated ToolServer           │
│      tools: ["*"]                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

## CRD Specification

### DataSource CRD

```yaml
apiVersion: kagent.dev/v1alpha1
kind: DataSource
metadata:
  name: sales-fabric
  namespace: default
spec:
  # Provider type - extensible for future data platforms
  provider: Databricks  # Enum: Databricks (future: Snowflake, BigQuery, etc.)

  # Provider-specific configuration
  databricks:
    # Databricks workspace URL
    workspaceUrl: https://mycompany.cloud.databricks.com

    # Reference to secret containing authentication token
    credentialsSecretRef:
      name: databricks-credentials
      key: token

    # Unity Catalog settings
    catalog: main
    schema: sales  # Optional: scope discovery to specific schema

  # Semantic models to expose (selected from discovered models)
  semanticModels:
    - name: revenue_metrics
      description: "Revenue KPIs by region and time period"
    - name: customer_360
      description: "Unified customer view with attributes and segments"

status:
  conditions:
    - type: Connected
      status: "True"
      reason: ConnectionSuccessful
      message: "Successfully connected to Databricks workspace"
    - type: Ready
      status: "True"
      reason: ToolServerCreated
      message: "ToolServer sales-fabric-mcp created successfully"

  # Models discovered from Unity Catalog (populated by controller)
  availableModels:
    - name: revenue_metrics
      catalog: main
      schema: sales
      description: "Revenue metrics semantic model"
    - name: customer_360
      catalog: main
      schema: sales
      description: "Customer 360 semantic model"
    - name: inventory_model
      catalog: main
      schema: operations
      description: "Inventory tracking model"

  # Reference to the auto-generated ToolServer
  generatedToolServer:
    name: sales-fabric-mcp
    namespace: default

  observedGeneration: 1
```

### Generated ToolServer

The DataSource controller automatically creates and manages a ToolServer:

```yaml
apiVersion: kagent.dev/v1alpha1
kind: ToolServer
metadata:
  name: sales-fabric-mcp
  namespace: default
  ownerReferences:
    - apiVersion: kagent.dev/v1alpha1
      kind: DataSource
      name: sales-fabric
      controller: true
      blockOwnerDeletion: true
  labels:
    kagent.dev/datasource: sales-fabric
    kagent.dev/provider: databricks
spec:
  description: "Auto-generated MCP server for DataSource sales-fabric (Databricks)"
  config:
    type: stdio
    stdio:
      command: /usr/local/bin/databricks-mcp-server
      args:
        - --workspace-url=$(DATABRICKS_WORKSPACE_URL)
        - --catalog=main
        - --schema=sales
        - --models=revenue_metrics,customer_360
      env:
        - name: DATABRICKS_WORKSPACE_URL
          value: "https://mycompany.cloud.databricks.com"
      envFrom:
        - secretRef:
            name: databricks-credentials
```

### Agent Usage

Agents reference the auto-generated ToolServer:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: sales-analyst
spec:
  description: "Sales data analyst agent"
  modelConfig:
    ref: azure-openai
  systemMessage: |
    You are a sales data analyst. Use the available tools to query
    semantic models and answer questions about sales data.
  tools:
    - toolServer:
        ref: sales-fabric-mcp
      tools: ["*"]  # Or specific tools: ["query_model", "describe_model"]
```

## MCP Server Tools

The Databricks MCP server exposes the following tools:

| Tool | Description | Parameters |
|------|-------------|------------|
| `query_model` | Query a semantic model using natural language | `model_name`, `question` |
| `execute_sql` | Execute SQL directly against the data fabric | `sql` |
| `describe_model` | Get schema and entity descriptions for a model | `model_name` |
| `list_models` | List all available semantic models | - |
| `get_model_metrics` | Get available metrics/measures in a model | `model_name` |
| `get_model_dimensions` | Get available dimensions in a model | `model_name` |

### Example Tool Invocations

```json
// query_model
{
  "model_name": "revenue_metrics",
  "question": "What was the total revenue by region for Q4 2024?"
}

// describe_model
{
  "model_name": "customer_360"
}

// execute_sql
{
  "sql": "SELECT region, SUM(revenue) FROM sales.revenue GROUP BY region"
}
```

## Controller Logic

### DataSource Controller Responsibilities

1. **Validate Connection**: Test connectivity to Databricks workspace on create/update
2. **Discover Models**: Query Unity Catalog API to populate `status.availableModels`
3. **Create ToolServer**: Generate and manage the child ToolServer resource
4. **Update Status**: Reflect connection state and available models in status
5. **Handle Updates**: Reconcile ToolServer when DataSource spec changes
6. **Cleanup**: ToolServer is garbage collected via owner reference when DataSource is deleted

### Reconciliation Flow

```
DataSource Created/Updated
        │
        ▼
┌───────────────────┐
│ Validate Spec     │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Test Connection   │──── Fail ──▶ Update status.conditions[Connected]=False
└───────────────────┘
        │ Success
        ▼
┌───────────────────┐
│ Discover Models   │──── Store in status.availableModels
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Create/Update     │
│ ToolServer        │
└───────────────────┘
        │
        ▼
┌───────────────────┐
│ Update Status     │──── status.conditions[Ready]=True
│ (generatedToolServer) │
└───────────────────┘
```

## UI Integration

### DataSource Creation Flow

1. User navigates to "Data Sources" section in Kagent UI
2. User clicks "Create DataSource" and selects provider (Databricks)
3. User enters connection details:
   - Workspace URL
   - Credentials (stored as Secret)
   - Catalog/Schema scope
4. UI calls backend to test connection and discover available models
5. User selects which semantic models to expose
6. DataSource CR is created
7. Controller creates ToolServer
8. DataSource appears in list with "Ready" status

### Agent Configuration Flow

1. User creates/edits an Agent
2. In "Tools" section, user sees available ToolServers
3. Auto-generated DataSource ToolServers are labeled/grouped for easy identification
4. User selects the DataSource ToolServer and desired tools
5. Agent is saved with ToolServer reference

## Implementation Plan

### Phase 1: Core CRD and Controller
- [ ] Define DataSource CRD types in `go/api/v1alpha1/datasource_types.go`
- [ ] Generate CRD manifests
- [ ] Implement DataSource controller with basic reconciliation
- [ ] Add ToolServer generation logic

### Phase 2: Databricks MCP Server
- [ ] Create MCP server binary for Databricks
- [ ] Implement Unity Catalog discovery
- [ ] Implement semantic model query tools
- [ ] Package as container image

### Phase 3: UI Integration
- [ ] Add DataSource list/create/edit pages
- [ ] Implement model discovery UI
- [ ] Update Agent configuration to show DataSource ToolServers

### Phase 4: Testing and Documentation
- [ ] Unit tests for controller
- [ ] Integration tests with Databricks
- [ ] User documentation
- [ ] Example manifests

## Future Considerations

### Additional Providers

The CRD is designed to support additional data platforms:

```yaml
spec:
  provider: Snowflake
  snowflake:
    accountUrl: https://myorg.snowflakecomputing.com
    warehouse: COMPUTE_WH
    database: SALES
    schema: PUBLIC
    credentialsSecretRef:
      name: snowflake-credentials
```

```yaml
spec:
  provider: BigQuery
  bigquery:
    projectId: my-gcp-project
    dataset: sales
    credentialsSecretRef:
      name: bigquery-sa-key
```

### Governance Features (Post-PoC)

- Row-level security integration
- Query audit logging
- Rate limiting / query quotas
- Allowed namespace restrictions

## Open Questions

1. **Model refresh frequency**: How often should the controller re-discover available models from Unity Catalog?

2. **Credential rotation**: How to handle credential updates without disrupting running agents?

3. **Error handling**: What happens to agents when DataSource connection fails?

4. **Caching**: Should the MCP server cache schema metadata to reduce API calls?

## References

- [Databricks Unity Catalog API](https://docs.databricks.com/api/workspace/catalogs)
- [Databricks Semantic Models](https://docs.databricks.com/en/ai-bi/semantic-model.html)
- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Kagent ToolServer CRD](../go/api/v1alpha1/toolserver_types.go)
