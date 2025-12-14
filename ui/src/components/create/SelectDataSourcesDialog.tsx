import { useState, useEffect, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Search, Database, PlusCircle, XCircle } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import type { DataSourceResponse } from "@/types";

interface SelectDataSourcesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  availableDataSources: DataSourceResponse[];
  selectedDataSources: DataSourceResponse[];
  onDataSourcesSelected: (dataSources: DataSourceResponse[]) => void;
  loading: boolean;
}

// Helper to extract display name from ref (e.g., "kagent/predicthq" -> "predicthq")
const getDataSourceDisplayName = (ds: DataSourceResponse): string => {
  const parts = ds.ref.split("/");
  return parts.length > 1 ? parts[1] : ds.ref;
};

// Helper to get namespace from ref
const getDataSourceNamespace = (ds: DataSourceResponse): string => {
  const parts = ds.ref.split("/");
  return parts.length > 1 ? parts[0] : "";
};

export const SelectDataSourcesDialog: React.FC<SelectDataSourcesDialogProps> = ({
  open,
  onOpenChange,
  availableDataSources,
  selectedDataSources,
  onDataSourcesSelected,
  loading,
}) => {
  const [searchTerm, setSearchTerm] = useState("");
  const [localSelectedDataSources, setLocalSelectedDataSources] = useState<DataSourceResponse[]>([]);

  // Initialize state when dialog opens
  useEffect(() => {
    if (open) {
      setLocalSelectedDataSources(selectedDataSources);
      setSearchTerm("");
    }
  }, [open, selectedDataSources]);

  // Filter to only show ready DataSources with a generated MCP server
  const readyDataSources = useMemo(() => {
    return availableDataSources.filter(ds => ds.ready && ds.generatedMCPServer);
  }, [availableDataSources]);

  // Filter by search term
  const filteredDataSources = useMemo(() => {
    if (!searchTerm) return readyDataSources;

    const searchLower = searchTerm.toLowerCase();
    return readyDataSources.filter(ds => {
      const name = getDataSourceDisplayName(ds).toLowerCase();
      const provider = ds.provider.toLowerCase();
      const catalog = ds.databricks?.catalog?.toLowerCase() || "";
      const schema = ds.databricks?.schema?.toLowerCase() || "";

      return name.includes(searchLower) ||
             provider.includes(searchLower) ||
             catalog.includes(searchLower) ||
             schema.includes(searchLower);
    });
  }, [readyDataSources, searchTerm]);

  const isDataSourceSelected = (ds: DataSourceResponse): boolean => {
    return localSelectedDataSources.some(selected => selected.ref === ds.ref);
  };

  const handleAddDataSource = (ds: DataSourceResponse) => {
    if (isDataSourceSelected(ds)) return;
    setLocalSelectedDataSources(prev => [...prev, ds]);
  };

  const handleRemoveDataSource = (ds: DataSourceResponse) => {
    setLocalSelectedDataSources(prev => prev.filter(selected => selected.ref !== ds.ref));
  };

  const handleSave = () => {
    onDataSourcesSelected(localSelectedDataSources);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const clearAllSelected = () => {
    setLocalSelectedDataSources([]);
  };

  const highlightMatch = (text: string, highlight: string) => {
    if (!highlight || !text) return text;
    const parts = text.split(new RegExp(`(${highlight.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&')})`, 'gi'));
    return parts.map((part, i) =>
      part.toLowerCase() === highlight.toLowerCase() ? <mark key={i} className="bg-yellow-200 px-0 py-0 rounded">{part}</mark> : part
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleCancel}>
      <DialogContent className="max-w-4xl max-h-[80vh] h-[70vh] flex flex-col p-0">
        <DialogHeader className="p-6 pb-4 border-b">
          <DialogTitle className="text-xl">Select Data Sources</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            Attach data sources to your agent. The agent will have access to the tools provided by each data source.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-1 overflow-hidden">
          {/* Left Panel: Available Data Sources */}
          <div className="w-1/2 border-r flex flex-col p-4 space-y-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search data sources..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-10 pr-4 py-2 h-10"
              />
            </div>

            <ScrollArea className="flex-1 -mr-4 pr-4">
              {loading ? (
                <div className="flex items-center justify-center h-full">
                  <p className="text-muted-foreground">Loading data sources...</p>
                </div>
              ) : filteredDataSources.length > 0 ? (
                <div className="space-y-2">
                  {filteredDataSources.map((ds) => {
                    const displayName = getDataSourceDisplayName(ds);
                    const namespace = getDataSourceNamespace(ds);
                    const isSelected = isDataSourceSelected(ds);

                    return (
                      <div
                        key={ds.ref}
                        className={`flex items-center justify-between p-3 border rounded-lg group ${
                          isSelected
                            ? 'bg-muted/50 cursor-default'
                            : 'cursor-pointer hover:bg-muted/30'
                        }`}
                        onClick={() => !isSelected && handleAddDataSource(ds)}
                      >
                        <div className="flex items-center gap-3 flex-1 overflow-hidden">
                          <Database className="h-5 w-5 flex-shrink-0 text-violet-500" />
                          <div className="flex-1 overflow-hidden">
                            <p className="font-medium text-sm truncate">
                              {highlightMatch(displayName, searchTerm)}
                            </p>
                            <div className="flex items-center gap-2 mt-1">
                              <Badge variant="secondary" className="text-xs">
                                {highlightMatch(ds.provider, searchTerm)}
                              </Badge>
                              {ds.databricks?.catalog && (
                                <span className="text-xs text-muted-foreground truncate">
                                  {highlightMatch(ds.databricks.catalog, searchTerm)}
                                  {ds.databricks.schema && `.${highlightMatch(ds.databricks.schema, searchTerm)}`}
                                </span>
                              )}
                            </div>
                            {namespace && (
                              <p className="text-xs text-muted-foreground/70 mt-1">
                                namespace: {namespace}
                              </p>
                            )}
                          </div>
                        </div>
                        {!isSelected && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 opacity-0 group-hover:opacity-100 text-green-600 hover:text-green-700"
                          >
                            <PlusCircle className="h-4 w-4" />
                          </Button>
                        )}
                        {isSelected && (
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-destructive hover:text-destructive/80"
                            onClick={(e) => {
                              e.stopPropagation();
                              handleRemoveDataSource(ds);
                            }}
                          >
                            <XCircle className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center h-[200px] text-center p-4 text-muted-foreground">
                  <Database className="h-10 w-10 mb-3 opacity-50" />
                  <p className="font-medium">No data sources available</p>
                  <p className="text-sm">
                    {searchTerm
                      ? "Try adjusting your search."
                      : "Create a data source first to attach it to agents."}
                  </p>
                </div>
              )}
            </ScrollArea>
          </div>

          {/* Right Panel: Selected Data Sources */}
          <div className="w-1/2 flex flex-col p-4 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold">Selected ({localSelectedDataSources.length})</h3>
              <Button
                variant="ghost"
                size="sm"
                onClick={clearAllSelected}
                disabled={localSelectedDataSources.length === 0}
              >
                Clear All
              </Button>
            </div>

            <ScrollArea className="flex-1 -mr-4 pr-4">
              {localSelectedDataSources.length > 0 ? (
                <div className="space-y-2">
                  {localSelectedDataSources.map((ds) => {
                    const displayName = getDataSourceDisplayName(ds);

                    return (
                      <div
                        key={ds.ref}
                        className="flex items-center justify-between p-3 border rounded-md bg-muted/30"
                      >
                        <div className="flex items-center gap-3 flex-1 overflow-hidden">
                          <Database className="h-4 w-4 flex-shrink-0 text-violet-500" />
                          <div className="flex-1 overflow-hidden">
                            <p className="text-sm font-medium truncate">{displayName}</p>
                            <p className="text-xs text-muted-foreground truncate">
                              {ds.provider}
                              {ds.databricks?.catalog && ` - ${ds.databricks.catalog}`}
                            </p>
                          </div>
                        </div>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-6 w-6 ml-2 flex-shrink-0"
                          onClick={() => handleRemoveDataSource(ds)}
                        >
                          <XCircle className="h-4 w-4" />
                        </Button>
                      </div>
                    );
                  })}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-center text-muted-foreground">
                  <PlusCircle className="h-10 w-10 mb-3 opacity-50" />
                  <p className="font-medium">No data sources selected</p>
                  <p className="text-sm">Select data sources from the left panel.</p>
                </div>
              )}
            </ScrollArea>
          </div>
        </div>

        {/* Footer */}
        <DialogFooter className="p-4 border-t mt-auto">
          <div className="flex justify-between w-full items-center">
            <div className="text-sm text-muted-foreground">
              {readyDataSources.length} data source{readyDataSources.length !== 1 ? 's' : ''} available
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={handleCancel}>Cancel</Button>
              <Button
                className="bg-violet-600 hover:bg-violet-700 text-white"
                onClick={handleSave}
              >
                Save Selection ({localSelectedDataSources.length})
              </Button>
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
