package handlers

import (
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
