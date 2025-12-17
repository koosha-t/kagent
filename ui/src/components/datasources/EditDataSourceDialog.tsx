"use client";

import { useState, useEffect, useCallback } from "react";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { DataSourceResponse, DatabricksTable } from "@/types";
import { getDatabricksTables, updateDataSource } from "@/app/actions/datasources";
import { toast } from "sonner";
import { Loader2, Table, Database } from "lucide-react";

interface EditDataSourceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  dataSource: DataSourceResponse | null;
  onSuccess: () => void;
}

export function EditDataSourceDialog({ open, onOpenChange, dataSource, onSuccess }: EditDataSourceDialogProps) {
  const [tables, setTables] = useState<DatabricksTable[]>([]);
  const [selectedTables, setSelectedTables] = useState<Set<string>>(new Set());
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const catalog = dataSource?.databricks?.catalog || "";
  const schema = dataSource?.databricks?.schema || "";

  // Fetch available tables when dialog opens
  const fetchTables = useCallback(async () => {
    if (!catalog || !schema) return;

    setIsLoading(true);
    try {
      const response = await getDatabricksTables(catalog, schema);
      if (!response.error && response.data) {
        setTables(response.data);
      } else {
        toast.error(response.error || "Failed to fetch tables");
      }
    } catch (error) {
      console.error("Error fetching tables:", error);
      toast.error("Failed to fetch tables");
    } finally {
      setIsLoading(false);
    }
  }, [catalog, schema]);

  // Initialize selected tables from dataSource
  useEffect(() => {
    if (open && dataSource) {
      const currentTables = new Set(
        dataSource.semanticModels?.map((m) => m.name) || []
      );
      setSelectedTables(currentTables);
      fetchTables();
    }
  }, [open, dataSource, fetchTables]);

  const handleTableToggle = (tableName: string, checked: boolean) => {
    setSelectedTables((prev: Set<string>) => {
      const newSet = new Set(prev);
      if (checked) {
        newSet.add(tableName);
      } else {
        newSet.delete(tableName);
      }
      return newSet;
    });
  };

  const handleSave = async () => {
    if (!dataSource?.ref) return;

    setIsSaving(true);
    try {
      const response = await updateDataSource(dataSource.ref, {
        tables: Array.from(selectedTables),
      });

      if (!response.error) {
        toast.success("Data source updated successfully");
        onSuccess();
        onOpenChange(false);
      } else {
        toast.error(response.error || "Failed to update data source");
      }
    } catch (error) {
      console.error("Error updating data source:", error);
      toast.error("Failed to update data source");
    } finally {
      setIsSaving(false);
    }
  };

  const handleSelectAll = () => {
    setSelectedTables(new Set(tables.map((t: DatabricksTable) => t.name)));
  };

  const handleClearAll = () => {
    setSelectedTables(new Set());
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl max-h-[80vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Edit Data Source
          </DialogTitle>
          <DialogDescription>
            Update the table selection for this data source. Catalog and schema cannot be changed.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 flex-1 overflow-hidden flex flex-col">
          {/* Read-only info */}
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Catalog:</span>
              <span className="ml-2 font-mono">{catalog}</span>
            </div>
            <div>
              <span className="text-muted-foreground">Schema:</span>
              <span className="ml-2 font-mono">{schema}</span>
            </div>
          </div>

          {/* Table selection */}
          <div className="flex-1 overflow-hidden flex flex-col">
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-medium">
                Select Tables ({selectedTables.size} selected)
              </h4>
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleSelectAll}
                  disabled={isLoading || tables.length === 0}
                >
                  Select All
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClearAll}
                  disabled={isLoading || selectedTables.size === 0}
                >
                  Clear All
                </Button>
              </div>
            </div>

            {isLoading ? (
              <div className="flex items-center justify-center h-[200px]">
                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                <span className="ml-2 text-muted-foreground">Loading tables...</span>
              </div>
            ) : tables.length > 0 ? (
              <div className="border rounded-md overflow-y-auto max-h-[300px]">
                {tables.map((table: DatabricksTable) => (
                  <label
                    key={table.name}
                    className="flex items-center gap-3 p-3 hover:bg-secondary/10 cursor-pointer border-b last:border-b-0"
                  >
                    <Checkbox
                      checked={selectedTables.has(table.name)}
                      onCheckedChange={(checked: boolean) =>
                        handleTableToggle(table.name, checked)
                      }
                    />
                    <div className="flex items-center gap-2 flex-1 min-w-0">
                      <Table className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                      <div className="truncate">
                        <span className="font-medium text-sm">{table.name}</span>
                        {table.comment && (
                          <span className="text-xs text-muted-foreground ml-2 truncate">
                            {table.comment}
                          </span>
                        )}
                      </div>
                    </div>
                    <span className="text-xs text-muted-foreground flex-shrink-0">
                      {table.tableType}
                    </span>
                  </label>
                ))}
              </div>
            ) : (
              <div className="flex items-center justify-center h-[200px] text-muted-foreground">
                No tables found in this schema.
              </div>
            )}
          </div>
        </div>

        <DialogFooter className="mt-4">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={isSaving || selectedTables.size === 0}>
            {isSaving && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
