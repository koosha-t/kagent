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

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/internal/controller/reconciler"
)

var (
	dataSourceControllerLog = ctrl.Log.WithName("datasource-controller")
)

// DataSourceController reconciles a DataSource object.
// It creates and manages Deployment, Service, and RemoteMCPServer resources for each DataSource,
// enabling agents to access data fabric semantic models via HTTP MCP tools.
type DataSourceController struct {
	Scheme     *runtime.Scheme
	Reconciler reconciler.KagentReconciler
}

// +kubebuilder:rbac:groups=kagent.dev,resources=datasources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kagent.dev,resources=datasources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kagent.dev,resources=datasources/finalizers,verbs=update
// +kubebuilder:rbac:groups=kagent.dev,resources=remotemcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is called whenever a DataSource changes.
// It delegates to the central reconciler which contains the business logic.
func (r *DataSourceController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	return ctrl.Result{}, r.Reconciler.ReconcileKagentDataSource(ctx, req)
}

// SetupWithManager sets up the controller with the Manager.
// It configures:
// - Primary watch on DataSource resources
// - Owns relationship for auto-generated Deployment, Service, and RemoteMCPServer resources (enables garbage collection)
// - Watch on Secrets to trigger re-reconciliation when credentials change
func (r *DataSourceController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			NeedLeaderElection: ptr.To(true),
		}).
		// Primary resource: DataSource
		// GenerationChangedPredicate ensures we only reconcile when spec changes,
		// not on every status update (prevents infinite loops)
		For(&v1alpha2.DataSource{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		// Owns: Deployment resources for the MCP server pods
		// When the Deployment changes, reconcile the parent DataSource
		// Enables garbage collection - when DataSource is deleted, Deployment is automatically deleted
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		// Owns: Service resources for the MCP server
		// Enables garbage collection - when DataSource is deleted, Service is automatically deleted
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		// Owns: RemoteMCPServer resources that agents reference
		// Enables garbage collection - when DataSource is deleted, RemoteMCPServer is automatically deleted
		Owns(&v1alpha2.RemoteMCPServer{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		// Watches: Secret resources (for credential changes)
		// When a secret changes, find all DataSources that reference it
		// and trigger reconciliation for each one
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				requests := []reconcile.Request{}

				for _, ds := range r.findDataSourcesUsingSecret(ctx, mgr.GetClient(), types.NamespacedName{
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
				}) {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      ds.Name,
							Namespace: ds.Namespace,
						},
					})
				}

				return requests
			}),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("datasource").
		Complete(r)
}

// findDataSourcesUsingSecret returns DataSources that reference the given secret.
// This enables re-reconciliation when credentials change.
func (r *DataSourceController) findDataSourcesUsingSecret(ctx context.Context, cl client.Client, secretRef types.NamespacedName) []*v1alpha2.DataSource {
	var dataSources []*v1alpha2.DataSource

	var dataSourceList v1alpha2.DataSourceList
	if err := cl.List(ctx, &dataSourceList); err != nil {
		dataSourceControllerLog.Error(err, "failed to list DataSources in order to reconcile Secret update")
		return dataSources
	}

	for i := range dataSourceList.Items {
		ds := &dataSourceList.Items[i]

		if dataSourceReferencesSecret(ds, secretRef) {
			dataSources = append(dataSources, ds)
		}
	}

	return dataSources
}

// dataSourceReferencesSecret checks if a DataSource references the given secret.
func dataSourceReferencesSecret(ds *v1alpha2.DataSource, secretRef types.NamespacedName) bool {
	// Secrets must be in the same namespace as the DataSource
	if ds.Namespace != secretRef.Namespace {
		return false
	}

	// Check if secret is referenced as Databricks credentials
	if ds.Spec.Databricks != nil &&
		ds.Spec.Databricks.CredentialsSecretRef != "" &&
		ds.Spec.Databricks.CredentialsSecretRef == secretRef.Name {
		return true
	}

	return false
}
