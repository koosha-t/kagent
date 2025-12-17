import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Plus, Database, X } from "lucide-react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useState, useEffect } from "react";
import { SelectDataSourcesDialog } from "./SelectDataSourcesDialog";
import type { DataSourceResponse } from "@/types";
import { getDataSources } from "@/app/actions/datasources";
import { Badge } from "@/components/ui/badge";

interface DataSourcesSectionProps {
  selectedDataSources: DataSourceResponse[];
  setSelectedDataSources: (dataSources: DataSourceResponse[]) => void;
  isSubmitting: boolean;
  onBlur?: () => void;
}

// Helper to extract display name from ref (e.g., "kagent/predicthq" -> "predicthq")
const getDataSourceDisplayName = (ds: DataSourceResponse): string => {
  const parts = ds.ref.split("/");
  return parts.length > 1 ? parts[1] : ds.ref;
};

export const DataSourcesSection = ({
  selectedDataSources,
  setSelectedDataSources,
  isSubmitting,
  onBlur,
}: DataSourcesSectionProps) => {
  const [showDataSourceSelector, setShowDataSourceSelector] = useState(false);
  const [availableDataSources, setAvailableDataSources] = useState<DataSourceResponse[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchDataSources = async () => {
      setLoading(true);
      try {
        const response = await getDataSources();
        if (!response.error && response.data) {
          setAvailableDataSources(response.data);
        } else {
          console.error("Failed to fetch data sources:", response.error);
        }
      } catch (error) {
        console.error("Failed to fetch data sources:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchDataSources();
  }, []);

  const handleDataSourceSelect = (newSelectedDataSources: DataSourceResponse[]) => {
    setSelectedDataSources(newSelectedDataSources);
    setShowDataSourceSelector(false);

    if (onBlur) {
      onBlur();
    }
  };

  const handleRemoveDataSource = (dsRef: string) => {
    const updatedDataSources = selectedDataSources.filter(ds => ds.ref !== dsRef);
    setSelectedDataSources(updatedDataSources);
  };

  const renderSelectedDataSources = () => (
    <div className="space-y-2">
      {selectedDataSources.map((ds) => {
        const displayName = getDataSourceDisplayName(ds);

        return (
          <Card key={ds.ref}>
            <CardContent className="p-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center text-xs">
                  <div className="inline-flex space-x-2 items-center">
                    <Database className="h-4 w-4 text-violet-500" />
                    <div className="inline-flex flex-col space-y-1">
                      <span className="font-medium">{displayName}</span>
                      <div className="flex items-center gap-2">
                        <Badge variant="secondary" className="text-xs">
                          {ds.provider}
                        </Badge>
                        {ds.databricks?.catalog && (
                          <span className="text-muted-foreground">
                            {ds.databricks.catalog}
                            {ds.databricks.schema && `.${ds.databricks.schema}`}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleRemoveDataSource(ds.ref)}
                    disabled={isSubmitting}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        );
      })}
    </div>
  );

  // Count available ready data sources
  const readyDataSourcesCount = availableDataSources.filter(
    ds => ds.ready && ds.generatedMCPServer
  ).length;

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <div>
          <h3 className="text-sm font-medium">Data Sources</h3>
          <p className="text-xs text-muted-foreground mt-1">
            Attach data sources to give your agent access to external data
          </p>
        </div>
        {selectedDataSources.length > 0 && (
          <Button
            onClick={() => setShowDataSourceSelector(true)}
            disabled={isSubmitting || readyDataSourcesCount === 0}
            variant="outline"
            className="border bg-transparent"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Data Source
          </Button>
        )}
      </div>

      <ScrollArea>
        {selectedDataSources.length === 0 ? (
          <Card>
            <CardContent className="p-6 flex flex-col items-center justify-center text-center">
              <Database className="h-10 w-10 mb-3 text-muted-foreground/50" />
              <h4 className="text-sm font-medium mb-1">No data sources attached</h4>
              <p className="text-muted-foreground text-xs mb-4">
                {readyDataSourcesCount > 0
                  ? "Add data sources to give your agent access to external data"
                  : "No data sources available. Create a data source first."}
              </p>
              {readyDataSourcesCount > 0 && (
                <Button
                  onClick={() => setShowDataSourceSelector(true)}
                  disabled={isSubmitting}
                  variant="outline"
                  size="sm"
                  className="flex items-center"
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add Data Source
                </Button>
              )}
            </CardContent>
          </Card>
        ) : (
          renderSelectedDataSources()
        )}
      </ScrollArea>

      <SelectDataSourcesDialog
        open={showDataSourceSelector}
        onOpenChange={setShowDataSourceSelector}
        availableDataSources={availableDataSources}
        selectedDataSources={selectedDataSources}
        onDataSourcesSelected={handleDataSourceSelect}
        loading={loading}
      />
    </div>
  );
};
