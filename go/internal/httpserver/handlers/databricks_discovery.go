package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/internal/httpserver/errors"
	"github.com/kagent-dev/kagent/go/pkg/auth"
	"github.com/kagent-dev/kagent/go/pkg/client/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// DatabricksDiscoveryHandler handles Databricks Unity Catalog discovery requests.
// It uses credentials from an existing DataSource to query Databricks APIs.
type DatabricksDiscoveryHandler struct {
	*Base
	httpClient *http.Client
}

// NewDatabricksDiscoveryHandler creates a new DatabricksDiscoveryHandler
func NewDatabricksDiscoveryHandler(base *Base) *DatabricksDiscoveryHandler {
	return &DatabricksDiscoveryHandler{
		Base: base,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// databricksConfig holds the configuration needed to call Databricks APIs
type databricksConfig struct {
	workspaceURL string
	token        string
}

// getConfigFromExistingDataSource retrieves Databricks credentials from the first existing DataSource
func (h *DatabricksDiscoveryHandler) getConfigFromExistingDataSource(ctx context.Context) (*databricksConfig, error) {
	// List all DataSources
	dataSourceList := &v1alpha2.DataSourceList{}
	if err := h.KubeClient.List(ctx, dataSourceList); err != nil {
		return nil, fmt.Errorf("failed to list DataSources: %w", err)
	}

	if len(dataSourceList.Items) == 0 {
		return nil, fmt.Errorf("no DataSources found - at least one DataSource must exist for discovery")
	}

	// Find the first Databricks DataSource
	var ds *v1alpha2.DataSource
	for i := range dataSourceList.Items {
		if dataSourceList.Items[i].Spec.Provider == v1alpha2.DataSourceProviderDatabricks &&
			dataSourceList.Items[i].Spec.Databricks != nil {
			ds = &dataSourceList.Items[i]
			break
		}
	}

	if ds == nil {
		return nil, fmt.Errorf("no Databricks DataSource found")
	}

	// Get the secret containing the token
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Namespace: ds.Namespace,
		Name:      ds.Spec.Databricks.CredentialsSecretRef,
	}
	if err := h.KubeClient.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get credentials secret %s: %w", secretKey, err)
	}

	tokenBytes, ok := secret.Data[ds.Spec.Databricks.CredentialsSecretKey]
	if !ok {
		return nil, fmt.Errorf("secret %s does not contain key %s", secretKey, ds.Spec.Databricks.CredentialsSecretKey)
	}

	return &databricksConfig{
		workspaceURL: ds.Spec.Databricks.WorkspaceURL,
		token:        string(tokenBytes),
	}, nil
}

// callDatabricksAPI makes a GET request to the Databricks API
func (h *DatabricksDiscoveryHandler) callDatabricksAPI(ctx context.Context, cfg *databricksConfig, path string) ([]byte, error) {
	url := cfg.workspaceURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cfg.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Databricks API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Databricks API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Databricks API response types
type databricksCatalogsResponse struct {
	Catalogs []databricksCatalog `json:"catalogs"`
}

type databricksCatalog struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

type databricksSchemasResponse struct {
	Schemas []databricksSchema `json:"schemas"`
}

type databricksSchema struct {
	Name        string `json:"name"`
	CatalogName string `json:"catalog_name"`
	Comment     string `json:"comment"`
}

type databricksTablesResponse struct {
	Tables        []databricksTable `json:"tables"`
	NextPageToken string            `json:"next_page_token"`
}

type databricksTable struct {
	Name        string `json:"name"`
	CatalogName string `json:"catalog_name"`
	SchemaName  string `json:"schema_name"`
	TableType   string `json:"table_type"`
	Comment     string `json:"comment"`
}

// HandleListCatalogs handles GET /api/databricks/catalogs
func (h *DatabricksDiscoveryHandler) HandleListCatalogs(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("databricks-discovery").WithValues("operation", "list-catalogs")
	log.Info("Received request to list Databricks catalogs")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "DataSource"}); err != nil {
		w.RespondWithError(err)
		return
	}

	cfg, err := h.getConfigFromExistingDataSource(r.Context())
	if err != nil {
		log.Error(err, "Failed to get Databricks configuration")
		w.RespondWithError(errors.NewInternalServerError("Failed to get Databricks configuration", err))
		return
	}

	body, err := h.callDatabricksAPI(r.Context(), cfg, "/api/2.1/unity-catalog/catalogs")
	if err != nil {
		log.Error(err, "Failed to list catalogs from Databricks")
		w.RespondWithError(errors.NewInternalServerError("Failed to list catalogs from Databricks", err))
		return
	}

	var dbResponse databricksCatalogsResponse
	if err := json.Unmarshal(body, &dbResponse); err != nil {
		log.Error(err, "Failed to parse Databricks response")
		w.RespondWithError(errors.NewInternalServerError("Failed to parse Databricks response", err))
		return
	}

	// Convert to API response format
	catalogs := make([]api.DatabricksCatalog, len(dbResponse.Catalogs))
	for i, c := range dbResponse.Catalogs {
		catalogs[i] = api.DatabricksCatalog{
			Name:    c.Name,
			Comment: c.Comment,
		}
	}

	log.Info("Successfully listed catalogs", "count", len(catalogs))
	data := api.NewResponse(catalogs, "Successfully listed catalogs", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleListSchemas handles GET /api/databricks/catalogs/{catalog}/schemas
func (h *DatabricksDiscoveryHandler) HandleListSchemas(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("databricks-discovery").WithValues("operation", "list-schemas")

	vars := mux.Vars(r)
	catalog := vars["catalog"]
	if catalog == "" {
		w.RespondWithError(errors.NewBadRequestError("catalog is required", nil))
		return
	}

	log = log.WithValues("catalog", catalog)
	log.Info("Received request to list Databricks schemas")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "DataSource"}); err != nil {
		w.RespondWithError(err)
		return
	}

	cfg, err := h.getConfigFromExistingDataSource(r.Context())
	if err != nil {
		log.Error(err, "Failed to get Databricks configuration")
		w.RespondWithError(errors.NewInternalServerError("Failed to get Databricks configuration", err))
		return
	}

	path := fmt.Sprintf("/api/2.1/unity-catalog/schemas?catalog_name=%s", catalog)
	body, err := h.callDatabricksAPI(r.Context(), cfg, path)
	if err != nil {
		log.Error(err, "Failed to list schemas from Databricks")
		w.RespondWithError(errors.NewInternalServerError("Failed to list schemas from Databricks", err))
		return
	}

	var dbResponse databricksSchemasResponse
	if err := json.Unmarshal(body, &dbResponse); err != nil {
		log.Error(err, "Failed to parse Databricks response")
		w.RespondWithError(errors.NewInternalServerError("Failed to parse Databricks response", err))
		return
	}

	// Convert to API response format
	schemas := make([]api.DatabricksSchema, len(dbResponse.Schemas))
	for i, s := range dbResponse.Schemas {
		schemas[i] = api.DatabricksSchema{
			Name:    s.Name,
			Catalog: s.CatalogName,
			Comment: s.Comment,
		}
	}

	log.Info("Successfully listed schemas", "count", len(schemas))
	data := api.NewResponse(schemas, "Successfully listed schemas", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleListTables handles GET /api/databricks/schemas/{catalog}/{schema}/tables
func (h *DatabricksDiscoveryHandler) HandleListTables(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("databricks-discovery").WithValues("operation", "list-tables")

	vars := mux.Vars(r)
	catalog := vars["catalog"]
	schema := vars["schema"]

	if catalog == "" || schema == "" {
		w.RespondWithError(errors.NewBadRequestError("catalog and schema are required", nil))
		return
	}

	log = log.WithValues("catalog", catalog, "schema", schema)
	log.Info("Received request to list Databricks tables")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "DataSource"}); err != nil {
		w.RespondWithError(err)
		return
	}

	cfg, err := h.getConfigFromExistingDataSource(r.Context())
	if err != nil {
		log.Error(err, "Failed to get Databricks configuration")
		w.RespondWithError(errors.NewInternalServerError("Failed to get Databricks configuration", err))
		return
	}

	// Fetch all tables with pagination
	var allTables []databricksTable
	nextPageToken := ""

	for {
		path := fmt.Sprintf("/api/2.1/unity-catalog/tables?catalog_name=%s&schema_name=%s&max_results=100", catalog, schema)
		if nextPageToken != "" {
			path += "&page_token=" + nextPageToken
		}

		body, err := h.callDatabricksAPI(r.Context(), cfg, path)
		if err != nil {
			log.Error(err, "Failed to list tables from Databricks")
			w.RespondWithError(errors.NewInternalServerError("Failed to list tables from Databricks", err))
			return
		}

		var dbResponse databricksTablesResponse
		if err := json.Unmarshal(body, &dbResponse); err != nil {
			log.Error(err, "Failed to parse Databricks response")
			w.RespondWithError(errors.NewInternalServerError("Failed to parse Databricks response", err))
			return
		}

		allTables = append(allTables, dbResponse.Tables...)

		if dbResponse.NextPageToken == "" {
			break
		}
		nextPageToken = dbResponse.NextPageToken
	}

	// Convert to API response format
	tables := make([]api.DatabricksTable, len(allTables))
	for i, t := range allTables {
		tables[i] = api.DatabricksTable{
			Name:      t.Name,
			Catalog:   t.CatalogName,
			Schema:    t.SchemaName,
			TableType: t.TableType,
			Comment:   t.Comment,
		}
	}

	log.Info("Successfully listed tables", "count", len(tables))
	data := api.NewResponse(tables, "Successfully listed tables", false)
	RespondWithJSON(w, http.StatusOK, data)
}
