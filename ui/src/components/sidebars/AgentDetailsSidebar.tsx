"use client";

import { useEffect, useState, useMemo } from "react";
import { ChevronRight, Edit, Database } from "lucide-react";
import type { AgentResponse, Tool, ToolsResponse, DataSourceResponse } from "@/types";
import { SidebarHeader, Sidebar, SidebarContent, SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuItem, SidebarMenuButton } from "@/components/ui/sidebar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { LoadingState } from "@/components/LoadingState";
import { isAgentTool, isMcpTool, getToolDescription, getToolIdentifier, getToolDisplayName } from "@/lib/toolUtils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { getAgents } from "@/app/actions/agents";
import { k8sRefUtils } from "@/lib/k8sUtils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";

interface AgentDetailsSidebarProps {
  selectedAgentName: string;
  currentAgent: AgentResponse;
  allTools: ToolsResponse[];
  dataSources?: DataSourceResponse[];
}

export function AgentDetailsSidebar({ selectedAgentName, currentAgent, allTools, dataSources = [] }: AgentDetailsSidebarProps) {
  const [toolDescriptions, setToolDescriptions] = useState<Record<string, string>>({});
  const [expandedTools, setExpandedTools] = useState<Record<string, boolean>>({});
  const [availableAgents, setAvailableAgents] = useState<AgentResponse[]>([]);

  const selectedTeam = currentAgent;

  // Identify DataSources linked to this agent via RemoteMCPServer tools
  const linkedDataSources = useMemo(() => {
    if (!selectedTeam?.tools || !dataSources.length) return [];

    return dataSources.filter(ds => {
      if (!ds.generatedMCPServer) return false;
      return selectedTeam.tools?.some(tool =>
        tool.type === "McpServer" &&
        tool.mcpServer?.kind === "RemoteMCPServer" &&
        tool.mcpServer?.name === ds.generatedMCPServer
      );
    });
  }, [selectedTeam?.tools, dataSources]);

  // Fetch agents for looking up agent tool descriptions
  useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = await getAgents();
        if (response.data) {
          setAvailableAgents(response.data);

        } else if (response.error) {
          console.error("AgentDetailsSidebar: Error fetching agents:", response.error);
        }
      } catch (error) {
        console.error("AgentDetailsSidebar: Failed to fetch agents:", error);
      }
    };

    fetchAgents();
  }, []);



  const RenderToolCollapsibleItem = ({
    itemKey,
    displayName,
    providerTooltip,
    description,
    isExpanded,
    onToggleExpansion,
  }: {
    itemKey: string;
    displayName: string;
    providerTooltip: string;
    description: string;
    isExpanded: boolean;
    onToggleExpansion: () => void;
  }) => {
    return (
      <Collapsible
        key={itemKey}
        open={isExpanded}
        onOpenChange={onToggleExpansion}
        className="group/collapsible"
      >
        <SidebarMenuItem>
          <CollapsibleTrigger asChild>
            <SidebarMenuButton tooltip={providerTooltip} className="w-full">
              <div className="flex items-center justify-between w-full">
                <span className="truncate max-w-[200px]">{displayName}</span>
                <ChevronRight
                  className={cn(
                    "h-4 w-4 transition-transform duration-200",
                    isExpanded && "rotate-90"
                  )}
                />
              </div>
            </SidebarMenuButton>
          </CollapsibleTrigger>
          <CollapsibleContent className="px-2 py-1">
            <div className="rounded-md bg-muted/50 p-2">
              <p className="text-sm text-muted-foreground">{description}</p>
            </div>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    );
  };

  useEffect(() => {
    const processToolDescriptions = () => {
      setToolDescriptions({});

      if (!selectedTeam || !allTools) return;

      const descriptions: Record<string, string> = {};
      const toolRefs = selectedTeam.tools;

      if (toolRefs && Array.isArray(toolRefs)) {
        toolRefs.forEach((tool) => {
          if (isMcpTool(tool)) {
            const mcpTool = tool as Tool;
            const baseToolIdentifier = getToolIdentifier(mcpTool);
            const hasExplicitTools = mcpTool.mcpServer?.toolNames && mcpTool.mcpServer.toolNames.length > 0;

            if (hasExplicitTools) {
              // For MCP tools with explicit tool names, each tool name gets its own description
              mcpTool.mcpServer?.toolNames.forEach((mcpToolName) => {
                const subToolIdentifier = `${baseToolIdentifier}::${mcpToolName}`;

                // Find the tool in allTools by matching server ref and tool name
                const toolFromDB = allTools.find(server => {
                  const { name } = k8sRefUtils.fromRef(server.server_name);
                  return name === mcpTool.mcpServer?.name && server.id === mcpToolName;
                });

                if (toolFromDB) {
                  descriptions[subToolIdentifier] = toolFromDB.description;
                } else {
                  descriptions[subToolIdentifier] = "No description available";
                }
              });
            } else {
              // For MCP tools with empty toolNames (DataSource case), look up all discovered tools
              const serverTools = allTools.filter(t => {
                const { name } = k8sRefUtils.fromRef(t.server_name);
                return name === mcpTool.mcpServer?.name;
              });
              serverTools.forEach((serverTool) => {
                const subToolIdentifier = `${baseToolIdentifier}::${serverTool.id}`;
                descriptions[subToolIdentifier] = serverTool.description;
              });
            }
          } else {
            // Handle Agent tools or regular tools using getToolDescription
            const toolIdentifier = getToolIdentifier(tool);
            descriptions[toolIdentifier] = getToolDescription(tool, allTools);
          }
        });
      }

      setToolDescriptions(descriptions);
    };

    processToolDescriptions();
  }, [selectedTeam, allTools, availableAgents]);

  const toggleToolExpansion = (toolIdentifier: string) => {
    setExpandedTools(prev => ({
      ...prev,
      [toolIdentifier]: !prev[toolIdentifier]
    }));
  };

  if (!selectedTeam) {
    return <LoadingState />;
  }

  const renderAgentTools = (tools: Tool[] = []) => {
    if (!tools || tools.length === 0) {
      return (
        <SidebarMenu>
          <div className="text-sm italic">No tools/agents available</div>
        </SidebarMenu>
      );
    }

    return (
      <SidebarMenu>
        {tools.flatMap((tool) => {
          const baseToolIdentifier = getToolIdentifier(tool);

          if (tool.mcpServer) {
            const mcpServerName = tool.mcpServer.name || "mcp_server";
            const mcpProviderParts = mcpServerName.split(".");
            const mcpProviderNameTooltip = mcpProviderParts[mcpProviderParts.length - 1];
            const hasExplicitTools = tool.mcpServer.toolNames && tool.mcpServer.toolNames.length > 0;

            if (hasExplicitTools) {
              // Render explicit tool names
              return tool.mcpServer.toolNames.map((mcpToolName) => {
                const subToolIdentifier = `${baseToolIdentifier}::${mcpToolName}`;
                const description = toolDescriptions[subToolIdentifier] || "Description loading or unavailable";
                const isExpanded = expandedTools[subToolIdentifier] || false;

                return (
                  <RenderToolCollapsibleItem
                    key={subToolIdentifier}
                    itemKey={subToolIdentifier}
                    displayName={mcpToolName}
                    providerTooltip={mcpProviderNameTooltip}
                    description={description}
                    isExpanded={isExpanded}
                    onToggleExpansion={() => toggleToolExpansion(subToolIdentifier)}
                  />
                );
              });
            } else {
              // For empty toolNames (DataSource case), look up all discovered tools from this server
              const serverTools = allTools.filter(t => {
                const { name } = k8sRefUtils.fromRef(t.server_name);
                return name === mcpServerName;
              });

              if (serverTools.length > 0) {
                return serverTools.map((serverTool) => {
                  const subToolIdentifier = `${baseToolIdentifier}::${serverTool.id}`;
                  const description = toolDescriptions[subToolIdentifier] || serverTool.description || "No description available";
                  const isExpanded = expandedTools[subToolIdentifier] || false;

                  return (
                    <RenderToolCollapsibleItem
                      key={subToolIdentifier}
                      itemKey={subToolIdentifier}
                      displayName={serverTool.id}
                      providerTooltip={mcpProviderNameTooltip}
                      description={description}
                      isExpanded={isExpanded}
                      onToggleExpansion={() => toggleToolExpansion(subToolIdentifier)}
                    />
                  );
                });
              } else {
                // Fallback: show server name if no tools discovered
                const toolIdentifier = baseToolIdentifier;
                const isExpanded = expandedTools[toolIdentifier] || false;

                return [(
                  <RenderToolCollapsibleItem
                    key={toolIdentifier}
                    itemKey={toolIdentifier}
                    displayName={mcpServerName}
                    providerTooltip={mcpProviderNameTooltip}
                    description="No tools discovered from this server"
                    isExpanded={isExpanded}
                    onToggleExpansion={() => toggleToolExpansion(toolIdentifier)}
                  />
                )];
              }
            }
          } else if (isAgentTool(tool)) {
            // Handle Agent tools
            const toolIdentifier = baseToolIdentifier;
            const provider = tool.agent?.name || "unknown";
            const displayName = getToolDisplayName(tool);
            const description = toolDescriptions[toolIdentifier] || "Description loading or unavailable";
            const isExpanded = expandedTools[toolIdentifier] || false;

            const providerParts = provider.split(".");
            const providerNameTooltip = providerParts[providerParts.length - 1];

            return [(
              <RenderToolCollapsibleItem
                key={toolIdentifier}
                itemKey={toolIdentifier}
                displayName={displayName}
                providerTooltip={providerNameTooltip}
                description={description}
                isExpanded={isExpanded}
                onToggleExpansion={() => toggleToolExpansion(toolIdentifier)}
              />
            )];
          }

          // Unknown tool type - skip
          return [];
        })}
      </SidebarMenu>
    );
  };

    // Check if agent is BYO type
  const isDeclarativeAgent = selectedTeam?.agent.spec.type === "Declarative";
  
  return (
    <>
      <Sidebar side={"right"} collapsible="offcanvas">
        <SidebarHeader>Agent Details</SidebarHeader>
        <SidebarContent>
          <ScrollArea>
            <SidebarGroup>
              <div className="flex items-center justify-between px-2 mb-1">
                <SidebarGroupLabel className="font-bold mb-0 p-0">
                  {selectedTeam?.agent.metadata.name} {selectedTeam?.model && `(${selectedTeam?.model})`}
                </SidebarGroupLabel>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7"
                  asChild
                  aria-label={`Edit agent ${selectedTeam?.agent.metadata.namespace}/${selectedTeam?.agent.metadata.name}`}
                >
                  <Link href={`/agents/new?edit=true&name=${selectedAgentName}&namespace=${currentAgent.agent.metadata.namespace}`}>
                    <Edit className="h-3.5 w-3.5" />
                  </Link>
                </Button>
              </div>
              <p className="text-sm flex px-2 text-muted-foreground">{selectedTeam?.agent.spec.description}</p>
            </SidebarGroup>
            {isDeclarativeAgent && linkedDataSources.length > 0 && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <div className="flex items-center justify-between px-2 mb-2">
                  <SidebarGroupLabel className="mb-0">Data Sources</SidebarGroupLabel>
                  <Badge variant="secondary" className="h-5">
                    {linkedDataSources.length}
                  </Badge>
                </div>
                <SidebarMenu>
                  {linkedDataSources.map((ds) => {
                    const displayName = ds.ref.split("/").pop() || ds.ref;
                    const catalogInfo = ds.databricks?.catalog
                      ? `${ds.databricks.catalog}${ds.databricks.schema ? `.${ds.databricks.schema}` : ""}`
                      : "";

                    return (
                      <SidebarMenuItem key={ds.ref}>
                        <SidebarMenuButton className="w-full h-auto">
                          <div className="flex items-center gap-2 w-full">
                            <Database className="h-4 w-4 text-violet-500 flex-shrink-0" />
                            <div className="flex flex-col min-w-0">
                              <span className="truncate text-sm font-medium">{displayName}</span>
                              <span className="text-xs text-muted-foreground truncate">
                                {ds.provider}{catalogInfo && ` • ${catalogInfo}`}
                              </span>
                            </div>
                          </div>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </SidebarMenu>
              </SidebarGroup>
            )}

            {isDeclarativeAgent && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <SidebarGroupLabel>Tools & Agents</SidebarGroupLabel>
                {selectedTeam && renderAgentTools(selectedTeam.tools)}
              </SidebarGroup>
            )}

            {isDeclarativeAgent && selectedTeam?.agent.spec?.skills?.refs && selectedTeam.agent.spec.skills.refs.length > 0 && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <div className="flex items-center justify-between px-2 mb-2">
                  <SidebarGroupLabel className="mb-0">Skills</SidebarGroupLabel>
                  <Badge variant="secondary" className="h-5">
                    {selectedTeam.agent.spec.skills.refs.length}
                  </Badge>
                </div>
                <SidebarMenu>
                  <TooltipProvider>
                    {selectedTeam.agent.spec.skills.refs.map((skillRef, index) => (
                      <SidebarMenuItem key={index}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <SidebarMenuButton className="w-full">
                              <div className="flex items-center justify-between w-full">
                                <span className="truncate max-w-[200px] text-sm">{skillRef}</span>
                              </div>
                            </SidebarMenuButton>
                          </TooltipTrigger>
                          <TooltipContent side="left">
                            <p className="max-w-xs break-all">{skillRef}</p>
                          </TooltipContent>
                        </Tooltip>
                      </SidebarMenuItem>
                    ))}
                  </TooltipProvider>
                </SidebarMenu>
              </SidebarGroup>
            )}

          </ScrollArea>
        </SidebarContent>
      </Sidebar>
    </>
  );
}
