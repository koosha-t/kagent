"use client";
import React, { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Loader2, Database, ChevronRight, AlertCircle } from "lucide-react";
import { useRouter } from "next/navigation";
import { LoadingState } from "@/components/LoadingState";
import { ErrorState } from "@/components/ErrorState";
import {
  getDatabricksCatalogs,
  getDatabricksSchemas,
  getDatabricksTables,
  createDataSource,
} from "@/app/actions/datasources";
import type {
  DatabricksCatalog,
  DatabricksSchema,
  DatabricksTable,
  CreateDataSourceRequest,
} from "@/types";
import { toast } from "sonner";
import { isResourceNameValid, createRFC1123ValidName } from "@/lib/utils";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";

interface ValidationErrors {
  name?: string;
  namespace?: string;
  catalog?: string;
  schema?: string;
  tables?: string;
}

export default function NewDataSourcePage() {
  const router = useRouter();

  // Form state
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("kagent");
  const [selectedCatalog, setSelectedCatalog] = useState<string>("");
  const [selectedSchema, setSelectedSchema] = useState<string>("");
  const [selectedTables, setSelectedTables] = useState<Set<string>>(new Set());
  const [warehouseId, setWarehouseId] = useState("");

  // Data state
  const [catalogs, setCatalogs] = useState<DatabricksCatalog[]>([]);
  const [schemas, setSchemas] = useState<DatabricksSchema[]>([]);
  const [tables, setTables] = useState<DatabricksTable[]>([]);

  // Loading states
  const [isLoadingCatalogs, setIsLoadingCatalogs] = useState(true);
  const [isLoadingSchemas, setIsLoadingSchemas] = useState(false);
  const [isLoadingTables, setIsLoadingTables] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Error states
  const [loadError, setLoadError] = useState<string | null>(null);
  const [errors, setErrors] = useState<ValidationErrors>({});

  // Load catalogs on mount
  useEffect(() => {
    const fetchCatalogs = async () => {
      setIsLoadingCatalogs(true);
      setLoadError(null);
      try {
        const response = await getDatabricksCatalogs();
        if (response.error || !response.data) {
          throw new Error(response.error || "Failed to fetch catalogs");
        }
        setCatalogs(response.data);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load catalogs";
        setLoadError(message);
        toast.error(message);
      } finally {
        setIsLoadingCatalogs(false);
      }
    };
    fetchCatalogs();
  }, []);

  // Load schemas when catalog changes
  useEffect(() => {
    if (!selectedCatalog) {
      setSchemas([]);
      setSelectedSchema("");
      return;
    }

    const fetchSchemas = async () => {
      setIsLoadingSchemas(true);
      setSchemas([]);
      setSelectedSchema("");
      setTables([]);
      setSelectedTables(new Set());
      try {
        const response = await getDatabricksSchemas(selectedCatalog);
        if (response.error || !response.data) {
          throw new Error(response.error || "Failed to fetch schemas");
        }
        setSchemas(response.data);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load schemas";
        toast.error(message);
      } finally {
        setIsLoadingSchemas(false);
      }
    };
    fetchSchemas();
  }, [selectedCatalog]);

  // Load tables when schema changes
  useEffect(() => {
    if (!selectedCatalog || !selectedSchema) {
      setTables([]);
      setSelectedTables(new Set());
      return;
    }

    const fetchTables = async () => {
      setIsLoadingTables(true);
      setTables([]);
      setSelectedTables(new Set());
      try {
        const response = await getDatabricksTables(selectedCatalog, selectedSchema);
        if (response.error || !response.data) {
          throw new Error(response.error || "Failed to fetch tables");
        }
        setTables(response.data);
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to load tables";
        toast.error(message);
      } finally {
        setIsLoadingTables(false);
      }
    };
    fetchTables();
  }, [selectedCatalog, selectedSchema]);

  // Auto-generate name when schema is selected
  useEffect(() => {
    if (selectedCatalog && selectedSchema && !name) {
      const generatedName = createRFC1123ValidName([selectedCatalog, selectedSchema]);
      if (generatedName && isResourceNameValid(generatedName)) {
        setName(generatedName);
      }
    }
  }, [selectedCatalog, selectedSchema, name]);

  const handleTableToggle = useCallback((tableName: string, checked: boolean) => {
    setSelectedTables((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.add(tableName);
      } else {
        next.delete(tableName);
      }
      return next;
    });
  }, []);

  const handleSelectAllTables = useCallback(() => {
    if (selectedTables.size === tables.length) {
      setSelectedTables(new Set());
    } else {
      setSelectedTables(new Set(tables.map((t) => t.name)));
    }
  }, [tables, selectedTables.size]);

  const validateForm = (): boolean => {
    const newErrors: ValidationErrors = {};

    if (!name.trim()) {
      newErrors.name = "Name is required";
    } else if (!isResourceNameValid(name)) {
      newErrors.name = "Name must be a valid RFC 1123 subdomain name";
    }

    if (!namespace.trim()) {
      newErrors.namespace = "Namespace is required";
    }

    if (!selectedCatalog) {
      newErrors.catalog = "Please select a catalog";
    }

    if (!selectedSchema) {
      newErrors.schema = "Please select a schema";
    }

    if (selectedTables.size === 0) {
      newErrors.tables = "Please select at least one table";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) {
      toast.error("Please fill in all required fields");
      return;
    }

    setIsSubmitting(true);
    setErrors({});

    try {
      const request: CreateDataSourceRequest = {
        name: name.trim(),
        namespace: namespace.trim(),
        catalog: selectedCatalog,
        schema: selectedSchema,
        tables: Array.from(selectedTables),
        warehouseId: warehouseId.trim() || undefined,
      };

      const response = await createDataSource(request);

      if (response.error || !response.data) {
        throw new Error(response.error || "Failed to create data source");
      }

      toast.success("Data source created successfully!");
      router.push("/datasources");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to create data source";
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  if (loadError && catalogs.length === 0) {
    return (
      <ErrorState
        message={loadError}
        description="Make sure at least one DataSource exists with valid Databricks credentials."
      />
    );
  }

  return (
    <div className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center gap-3 mb-8">
          <Database className="h-8 w-8 text-primary" />
          <div>
            <h1 className="text-2xl font-bold">Create New Data Source</h1>
            <p className="text-muted-foreground">
              Connect to a Databricks Unity Catalog schema
            </p>
          </div>
        </div>

        <div className="space-y-6">
          {/* Step 1: Select Catalog */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm">
                  1
                </span>
                Select Catalog
              </CardTitle>
              <CardDescription>
                Choose a Unity Catalog to browse schemas
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoadingCatalogs ? (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading catalogs...
                </div>
              ) : (
                <Select value={selectedCatalog} onValueChange={setSelectedCatalog}>
                  <SelectTrigger className={errors.catalog ? "border-destructive" : ""}>
                    <SelectValue placeholder="Select a catalog..." />
                  </SelectTrigger>
                  <SelectContent>
                    {catalogs.map((catalog) => (
                      <SelectItem key={catalog.name} value={catalog.name}>
                        <div className="flex flex-col">
                          <span>{catalog.name}</span>
                          {catalog.comment && (
                            <span className="text-xs text-muted-foreground">
                              {catalog.comment}
                            </span>
                          )}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {errors.catalog && (
                <p className="text-sm text-destructive mt-1">{errors.catalog}</p>
              )}
            </CardContent>
          </Card>

          {/* Step 2: Select Schema */}
          <Card className={!selectedCatalog ? "opacity-50" : ""}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm">
                  2
                </span>
                Select Schema
              </CardTitle>
              <CardDescription>
                Choose a schema to expose tables from
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoadingSchemas ? (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading schemas...
                </div>
              ) : (
                <Select
                  value={selectedSchema}
                  onValueChange={setSelectedSchema}
                  disabled={!selectedCatalog}
                >
                  <SelectTrigger className={errors.schema ? "border-destructive" : ""}>
                    <SelectValue placeholder="Select a schema..." />
                  </SelectTrigger>
                  <SelectContent>
                    {schemas.map((schema) => (
                      <SelectItem key={schema.name} value={schema.name}>
                        <div className="flex flex-col">
                          <span>{schema.name}</span>
                          {schema.comment && (
                            <span className="text-xs text-muted-foreground">
                              {schema.comment}
                            </span>
                          )}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {errors.schema && (
                <p className="text-sm text-destructive mt-1">{errors.schema}</p>
              )}
            </CardContent>
          </Card>

          {/* Step 3: Select Tables */}
          <Card className={!selectedSchema ? "opacity-50" : ""}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm">
                  3
                </span>
                Select Tables
              </CardTitle>
              <CardDescription>
                Choose which tables to expose as semantic models
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoadingTables ? (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Loading tables...
                </div>
              ) : tables.length === 0 ? (
                <p className="text-muted-foreground">
                  {selectedSchema ? "No tables found in this schema" : "Select a schema first"}
                </p>
              ) : (
                <div className="space-y-3">
                  <div className="flex items-center justify-between pb-2 border-b">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleSelectAllTables}
                      disabled={!selectedSchema}
                    >
                      {selectedTables.size === tables.length ? "Deselect All" : "Select All"}
                    </Button>
                    <span className="text-sm text-muted-foreground">
                      {selectedTables.size} of {tables.length} selected
                    </span>
                  </div>
                  <div className="max-h-64 overflow-y-auto space-y-2">
                    {tables.map((table) => (
                      <label
                        key={table.name}
                        className="flex items-center gap-3 p-2 rounded-md hover:bg-muted cursor-pointer"
                      >
                        <Checkbox
                          checked={selectedTables.has(table.name)}
                          onCheckedChange={(checked) =>
                            handleTableToggle(table.name, checked as boolean)
                          }
                        />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{table.name}</span>
                            <span className="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
                              {table.tableType}
                            </span>
                          </div>
                          {table.comment && (
                            <p className="text-xs text-muted-foreground truncate">
                              {table.comment}
                            </p>
                          )}
                        </div>
                      </label>
                    ))}
                  </div>
                </div>
              )}
              {errors.tables && (
                <p className="text-sm text-destructive mt-1">{errors.tables}</p>
              )}
            </CardContent>
          </Card>

          {/* Step 4: Configure */}
          <Card className={!selectedSchema ? "opacity-50" : ""}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <span className="flex items-center justify-center w-6 h-6 rounded-full bg-primary text-primary-foreground text-sm">
                  4
                </span>
                Configure Data Source
              </CardTitle>
              <CardDescription>
                Set the name and optional settings
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Name *</Label>
                  <Input
                    id="name"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="my-datasource"
                    className={errors.name ? "border-destructive" : ""}
                    disabled={!selectedSchema}
                  />
                  {errors.name && (
                    <p className="text-sm text-destructive">{errors.name}</p>
                  )}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="namespace">Namespace *</Label>
                  <Input
                    id="namespace"
                    value={namespace}
                    onChange={(e) => setNamespace(e.target.value)}
                    placeholder="kagent"
                    className={errors.namespace ? "border-destructive" : ""}
                    disabled={!selectedSchema}
                  />
                  {errors.namespace && (
                    <p className="text-sm text-destructive">{errors.namespace}</p>
                  )}
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="warehouseId">SQL Warehouse ID (Optional)</Label>
                <Input
                  id="warehouseId"
                  value={warehouseId}
                  onChange={(e) => setWarehouseId(e.target.value)}
                  placeholder="Leave empty to use default"
                  disabled={!selectedSchema}
                />
                <p className="text-xs text-muted-foreground">
                  If not provided, the warehouse ID from an existing DataSource will be used.
                </p>
              </div>
            </CardContent>
          </Card>

          {/* Summary */}
          {selectedCatalog && selectedSchema && selectedTables.size > 0 && (
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>
                <strong>Summary:</strong> Creating data source &quot;{name || "unnamed"}&quot; in
                namespace &quot;{namespace}&quot; with {selectedTables.size} table(s) from{" "}
                <code className="px-1 py-0.5 rounded bg-muted">
                  {selectedCatalog}.{selectedSchema}
                </code>
              </AlertDescription>
            </Alert>
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="outline" onClick={() => router.push("/datasources")}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={isSubmitting || !selectedSchema || selectedTables.size === 0}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  Create Data Source
                  <ChevronRight className="h-4 w-4 ml-2" />
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
