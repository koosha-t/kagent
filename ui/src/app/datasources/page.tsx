"use client";

import { useState, useEffect } from "react";
import { Database, ChevronDown, ChevronRight, CheckCircle2, XCircle, ExternalLink, Table } from "lucide-react";
import { DataSourceResponse } from "@/types";
import { getDataSources } from "../actions/datasources";
import { Badge } from "@/components/ui/badge";
import Link from "next/link";
import { toast } from "sonner";

export default function DataSourcesPage() {
  const [dataSources, setDataSources] = useState<DataSourceResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [expandedSources, setExpandedSources] = useState<Set<string>>(new Set());

  useEffect(() => {
    fetchDataSources();
  }, []);

  const fetchDataSources = async () => {
    try {
      setIsLoading(true);
      const response = await getDataSources();
      if (!response.error && response.data) {
        const sorted = [...response.data].sort((a, b) =>
          (a.ref || "").localeCompare(b.ref || "")
        );
        setDataSources(sorted);
      } else {
        console.error("Failed to fetch data sources:", response);
        toast.error(response.error || "Failed to fetch data sources");
      }
    } catch (error) {
      console.error("Error fetching data sources:", error);
      toast.error("An error occurred while fetching data sources");
    } finally {
      setIsLoading(false);
    }
  };

  const toggleSource = (ref: string) => {
    setExpandedSources((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(ref)) {
        newSet.delete(ref);
      } else {
        newSet.add(ref);
      }
      return newSet;
    });
  };

  const StatusIndicator = ({ ok, label }: { ok: boolean; label: string }) => (
    <div className="flex items-center gap-1 text-sm">
      {ok ? (
        <CheckCircle2 className="h-4 w-4 text-green-500" />
      ) : (
        <XCircle className="h-4 w-4 text-red-500" />
      )}
      <span className={ok ? "text-green-700 dark:text-green-400" : "text-red-700 dark:text-red-400"}>{label}</span>
    </div>
  );

  return (
    <div className="mt-12 mx-auto max-w-6xl px-6">
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold">Data Sources</h1>
          <Link href="/servers" className="text-blue-600 hover:text-blue-800 text-sm">
            View MCP Servers →
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="flex flex-col items-center justify-center h-[200px] border rounded-lg bg-secondary/5">
          <div className="animate-pulse h-6 w-6 rounded-full bg-primary/10 mb-4"></div>
          <p className="text-muted-foreground">Loading data sources...</p>
        </div>
      ) : dataSources.length > 0 ? (
        <div className="space-y-4">
          {dataSources.map((ds) => {
            if (!ds.ref) return null;
            const isExpanded = expandedSources.has(ds.ref);

            return (
              <div key={ds.ref} className="border rounded-md overflow-hidden">
                <div className="bg-secondary/10 p-4">
                  <div className="flex items-center justify-between">
                    <div
                      className="flex items-center gap-3 cursor-pointer flex-1"
                      onClick={() => toggleSource(ds.ref)}
                    >
                      {isExpanded ? (
                        <ChevronDown className="h-5 w-5" />
                      ) : (
                        <ChevronRight className="h-5 w-5" />
                      )}
                      <Database className="h-5 w-5 text-blue-500" />
                      <div>
                        <div className="font-medium">{ds.ref}</div>
                        <div className="text-xs text-muted-foreground">
                          {ds.databricks?.workspaceUrl}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge variant="outline">{ds.provider}</Badge>
                      <StatusIndicator ok={ds.connected} label="Connected" />
                      <StatusIndicator ok={ds.ready} label="Ready" />
                    </div>
                  </div>
                </div>

                {isExpanded && (
                  <div className="p-4 space-y-4">
                    {ds.databricks && (
                      <div>
                        <h3 className="font-medium text-sm mb-2">Configuration</h3>
                        <div className="grid grid-cols-2 gap-2 text-sm">
                          <div className="text-muted-foreground">Catalog:</div>
                          <div className="font-mono">{ds.databricks.catalog}</div>
                          {ds.databricks.schema && (
                            <>
                              <div className="text-muted-foreground">Schema:</div>
                              <div className="font-mono">{ds.databricks.schema}</div>
                            </>
                          )}
                          {ds.databricks.warehouseId && (
                            <>
                              <div className="text-muted-foreground">Warehouse:</div>
                              <div className="font-mono">{ds.databricks.warehouseId}</div>
                            </>
                          )}
                          <div className="text-muted-foreground">Credentials:</div>
                          <div className="font-mono">
                            {ds.databricks.credentialsSecretRef}/{ds.databricks.credentialsSecretKey}
                          </div>
                        </div>
                      </div>
                    )}

                    {ds.generatedMCPServer && (
                      <div>
                        <h3 className="font-medium text-sm mb-2">Generated MCP Server</h3>
                        <Link
                          href="/servers"
                          className="inline-flex items-center gap-1 text-blue-600 hover:text-blue-800"
                        >
                          <ExternalLink className="h-4 w-4" />
                          {ds.generatedMCPServer}
                        </Link>
                      </div>
                    )}

                    {ds.availableModels && ds.availableModels.length > 0 && (
                      <div>
                        <h3 className="font-medium text-sm mb-2">
                          Available Models ({ds.availableModels.length})
                        </h3>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                          {ds.availableModels.map((model) => (
                            <div
                              key={model.name}
                              className="p-2 border rounded-md bg-secondary/5"
                            >
                              <div className="flex items-center gap-2">
                                <Table className="h-4 w-4 text-muted-foreground" />
                                <span className="font-medium text-sm">{model.name}</span>
                              </div>
                              <div className="text-xs text-muted-foreground mt-1">
                                {model.catalog}.{model.schema}
                                {model.description && ` - ${model.description}`}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {ds.semanticModels && ds.semanticModels.length > 0 && (
                      <div>
                        <h3 className="font-medium text-sm mb-2">
                          Selected Models ({ds.semanticModels.length})
                        </h3>
                        <div className="flex flex-wrap gap-2">
                          {ds.semanticModels.map((model) => (
                            <Badge key={model.name} variant="secondary">
                              {model.name}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center h-[300px] text-center p-4 border rounded-lg bg-secondary/5">
          <Database className="h-12 w-12 text-muted-foreground mb-4 opacity-20" />
          <h3 className="font-medium text-lg">No data sources configured</h3>
          <p className="text-muted-foreground mt-1 mb-4">
            Create a DataSource using kubectl to connect to your data fabric.
          </p>
          <code className="text-sm bg-muted px-3 py-2 rounded">
            kubectl apply -f datasource.yaml
          </code>
        </div>
      )}
    </div>
  );
}
