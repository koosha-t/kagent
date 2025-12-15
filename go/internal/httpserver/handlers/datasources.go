package handlers

import (
	"context"
	"net/http"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/internal/httpserver/errors"
	common "github.com/kagent-dev/kagent/go/internal/utils"
	"github.com/kagent-dev/kagent/go/pkg/auth"
	"github.com/kagent-dev/kagent/go/pkg/client/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// DataSourcesHandler handles DataSource-related HTTP requests.
// It reads DataSource CRDs directly from Kubernetes (not the database).
type DataSourcesHandler struct {
	*Base
}

// NewDataSourcesHandler creates a new DataSourcesHandler
func NewDataSourcesHandler(base *Base) *DataSourcesHandler {
	return &DataSourcesHandler{Base: base}
}

// HandleListDataSources handles GET /api/datasources requests.
// It lists all DataSource CRDs and converts them to DataSourceResponse format.
func (h *DataSourcesHandler) HandleListDataSources(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("datasources-handler").WithValues("operation", "list")
	log.Info("Received request to list DataSources")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "DataSource"}); err != nil {
		w.RespondWithError(err)
		return
	}

	// List all DataSource CRDs from Kubernetes
	dataSourceList := &v1alpha2.DataSourceList{}
	if err := h.KubeClient.List(r.Context(), dataSourceList); err != nil {
		log.Error(err, "Failed to list DataSources from Kubernetes")
		w.RespondWithError(errors.NewInternalServerError("Failed to list DataSources", err))
		return
	}

	// Convert CRDs to API response format
	responses := make([]api.DataSourceResponse, len(dataSourceList.Items))
	for i, ds := range dataSourceList.Items {
		responses[i] = api.DataSourceResponse{
			Ref:                common.GetObjectRef(&ds),
			Provider:           string(ds.Spec.Provider),
			Databricks:         ds.Spec.Databricks,
			SemanticModels:     ds.Spec.SemanticModels,
			AvailableModels:    ds.Status.AvailableModels,
			GeneratedMCPServer: ds.Status.GeneratedMCPServer,
			Connected:          isConditionTrue(ds.Status.Conditions, v1alpha2.DataSourceConditionTypeConnected),
			Ready:              isConditionTrue(ds.Status.Conditions, v1alpha2.DataSourceConditionTypeReady),
		}
	}

	log.Info("Successfully listed DataSources", "count", len(responses))
	data := api.NewResponse(responses, "Successfully listed DataSources", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// isConditionTrue checks if a Kubernetes condition with the given type has status "True".
func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, c := range conditions {
		if c.Type == conditionType {
			return c.Status == metav1.ConditionTrue
		}
	}
	return false
}

// HandleCreateDataSource handles POST /api/datasources requests.
// It creates a new DataSource CRD using credentials from an existing DataSource.
func (h *DataSourcesHandler) HandleCreateDataSource(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("datasources-handler").WithValues("operation", "create")
	log.Info("Received request to create DataSource")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "DataSource"}); err != nil {
		w.RespondWithError(err)
		return
	}

	// Parse request body
	var req api.CreateDataSourceRequest
	if err := DecodeJSONBody(r, &req); err != nil {
		log.Error(err, "Failed to parse request body")
		w.RespondWithError(errors.NewBadRequestError("Invalid request body", err))
		return
	}

	// Validate required fields
	if req.Name == "" {
		w.RespondWithError(errors.NewBadRequestError("name is required", nil))
		return
	}
	if req.Namespace == "" {
		w.RespondWithError(errors.NewBadRequestError("namespace is required", nil))
		return
	}
	if req.Catalog == "" {
		w.RespondWithError(errors.NewBadRequestError("catalog is required", nil))
		return
	}
	if req.Schema == "" {
		w.RespondWithError(errors.NewBadRequestError("schema is required", nil))
		return
	}

	log = log.WithValues("name", req.Name, "namespace", req.Namespace, "catalog", req.Catalog, "schema", req.Schema)

	// Get configuration from an existing DataSource
	existingConfig, err := h.getExistingDatabricksConfig(r.Context())
	if err != nil {
		log.Error(err, "Failed to get configuration from existing DataSource")
		w.RespondWithError(errors.NewInternalServerError("Failed to get Databricks configuration", err))
		return
	}

	// Build semantic models from selected tables
	var semanticModels []v1alpha2.SemanticModelRef
	for _, table := range req.Tables {
		semanticModels = append(semanticModels, v1alpha2.SemanticModelRef{
			Name: table,
		})
	}

	// Create DataSource CRD
	ds := &v1alpha2.DataSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: v1alpha2.DataSourceSpec{
			Provider: v1alpha2.DataSourceProviderDatabricks,
			Databricks: &v1alpha2.DatabricksConfig{
				WorkspaceURL:         existingConfig.workspaceURL,
				CredentialsSecretRef: existingConfig.secretName,
				CredentialsSecretKey: existingConfig.secretKey,
				Catalog:              req.Catalog,
				Schema:               req.Schema,
				WarehouseID:          req.WarehouseID,
			},
			SemanticModels: semanticModels,
		},
	}

	// Use warehouseID from existing config if not provided
	if ds.Spec.Databricks.WarehouseID == "" && existingConfig.warehouseID != "" {
		ds.Spec.Databricks.WarehouseID = existingConfig.warehouseID
	}

	if err := h.KubeClient.Create(r.Context(), ds); err != nil {
		log.Error(err, "Failed to create DataSource")
		w.RespondWithError(errors.NewInternalServerError("Failed to create DataSource", err))
		return
	}

	// Return the created DataSource
	response := api.DataSourceResponse{
		Ref:            common.GetObjectRef(ds),
		Provider:       string(ds.Spec.Provider),
		Databricks:     ds.Spec.Databricks,
		SemanticModels: ds.Spec.SemanticModels,
		Connected:      false,
		Ready:          false,
	}

	log.Info("Successfully created DataSource")
	data := api.NewResponse(response, "Successfully created DataSource", false)
	RespondWithJSON(w, http.StatusCreated, data)
}

// existingDatabricksConfig holds configuration from an existing DataSource
type existingDatabricksConfig struct {
	workspaceURL string
	secretName   string
	secretKey    string
	warehouseID  string
}

// getExistingDatabricksConfig retrieves configuration from the first existing Databricks DataSource
func (h *DataSourcesHandler) getExistingDatabricksConfig(ctx context.Context) (*existingDatabricksConfig, error) {
	dataSourceList := &v1alpha2.DataSourceList{}
	if err := h.KubeClient.List(ctx, dataSourceList); err != nil {
		return nil, err
	}

	for _, ds := range dataSourceList.Items {
		if ds.Spec.Provider == v1alpha2.DataSourceProviderDatabricks && ds.Spec.Databricks != nil {
			return &existingDatabricksConfig{
				workspaceURL: ds.Spec.Databricks.WorkspaceURL,
				secretName:   ds.Spec.Databricks.CredentialsSecretRef,
				secretKey:    ds.Spec.Databricks.CredentialsSecretKey,
				warehouseID:  ds.Spec.Databricks.WarehouseID,
			}, nil
		}
	}

	return nil, errors.NewNotFoundError("No existing Databricks DataSource found", nil)
}
