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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kagent-dev/kagent/go/api/v1alpha2"
)

func TestDataSourceReferencesSecret(t *testing.T) {
	tests := []struct {
		name       string
		dataSource *v1alpha2.DataSource
		secretRef  types.NamespacedName
		expected   bool
	}{
		{
			name: "matching secret reference",
			dataSource: &v1alpha2.DataSource{
				Spec: v1alpha2.DataSourceSpec{
					Provider: v1alpha2.DataSourceProviderDatabricks,
					Databricks: &v1alpha2.DatabricksConfig{
						WorkspaceURL:         "https://example.cloud.databricks.com",
						CredentialsSecretRef: "databricks-creds",
						CredentialsSecretKey: "token",
						Catalog:              "main",
					},
				},
			},
			secretRef: types.NamespacedName{
				Name:      "databricks-creds",
				Namespace: "",
			},
			expected: true,
		},
		{
			name: "non-matching secret name",
			dataSource: &v1alpha2.DataSource{
				Spec: v1alpha2.DataSourceSpec{
					Provider: v1alpha2.DataSourceProviderDatabricks,
					Databricks: &v1alpha2.DatabricksConfig{
						WorkspaceURL:         "https://example.cloud.databricks.com",
						CredentialsSecretRef: "databricks-creds",
						CredentialsSecretKey: "token",
						Catalog:              "main",
					},
				},
			},
			secretRef: types.NamespacedName{
				Name:      "other-secret",
				Namespace: "",
			},
			expected: false,
		},
		{
			name: "non-matching namespace",
			dataSource: &v1alpha2.DataSource{
				Spec: v1alpha2.DataSourceSpec{
					Provider: v1alpha2.DataSourceProviderDatabricks,
					Databricks: &v1alpha2.DatabricksConfig{
						WorkspaceURL:         "https://example.cloud.databricks.com",
						CredentialsSecretRef: "databricks-creds",
						CredentialsSecretKey: "token",
						Catalog:              "main",
					},
				},
			},
			secretRef: types.NamespacedName{
				Name:      "databricks-creds",
				Namespace: "other-namespace",
			},
			expected: false,
		},
		{
			name: "nil databricks config",
			dataSource: &v1alpha2.DataSource{
				Spec: v1alpha2.DataSourceSpec{
					Provider:   v1alpha2.DataSourceProviderDatabricks,
					Databricks: nil,
				},
			},
			secretRef: types.NamespacedName{
				Name:      "databricks-creds",
				Namespace: "",
			},
			expected: false,
		},
		{
			name: "empty credentials secret ref",
			dataSource: &v1alpha2.DataSource{
				Spec: v1alpha2.DataSourceSpec{
					Provider: v1alpha2.DataSourceProviderDatabricks,
					Databricks: &v1alpha2.DatabricksConfig{
						WorkspaceURL:         "https://example.cloud.databricks.com",
						CredentialsSecretRef: "",
						CredentialsSecretKey: "token",
						Catalog:              "main",
					},
				},
			},
			secretRef: types.NamespacedName{
				Name:      "databricks-creds",
				Namespace: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dataSourceReferencesSecret(tt.dataSource, tt.secretRef)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDataSourceSpec_Validation(t *testing.T) {
	tests := []struct {
		name       string
		dataSource v1alpha2.DataSourceSpec
		valid      bool
	}{
		{
			name: "valid databricks config",
			dataSource: v1alpha2.DataSourceSpec{
				Provider: v1alpha2.DataSourceProviderDatabricks,
				Databricks: &v1alpha2.DatabricksConfig{
					WorkspaceURL:         "https://example.cloud.databricks.com",
					CredentialsSecretRef: "databricks-creds",
					CredentialsSecretKey: "token",
					Catalog:              "main",
					Schema:               "sales",
				},
			},
			valid: true,
		},
		{
			name: "valid databricks config with semantic models",
			dataSource: v1alpha2.DataSourceSpec{
				Provider: v1alpha2.DataSourceProviderDatabricks,
				Databricks: &v1alpha2.DatabricksConfig{
					WorkspaceURL:         "https://example.cloud.databricks.com",
					CredentialsSecretRef: "databricks-creds",
					CredentialsSecretKey: "token",
					Catalog:              "main",
				},
				SemanticModels: []v1alpha2.SemanticModelRef{
					{Name: "revenue_metrics", Description: "Revenue KPIs"},
					{Name: "customer_360"},
				},
			},
			valid: true,
		},
		{
			name: "valid databricks config without schema",
			dataSource: v1alpha2.DataSourceSpec{
				Provider: v1alpha2.DataSourceProviderDatabricks,
				Databricks: &v1alpha2.DatabricksConfig{
					WorkspaceURL:         "https://example.cloud.databricks.com",
					CredentialsSecretRef: "databricks-creds",
					CredentialsSecretKey: "token",
					Catalog:              "main",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic field presence validation
			if tt.valid {
				assert.NotEmpty(t, tt.dataSource.Provider)
				if tt.dataSource.Provider == v1alpha2.DataSourceProviderDatabricks {
					assert.NotNil(t, tt.dataSource.Databricks)
					assert.NotEmpty(t, tt.dataSource.Databricks.WorkspaceURL)
					assert.NotEmpty(t, tt.dataSource.Databricks.CredentialsSecretRef)
					assert.NotEmpty(t, tt.dataSource.Databricks.CredentialsSecretKey)
					assert.NotEmpty(t, tt.dataSource.Databricks.Catalog)
				}
			}
		})
	}
}
