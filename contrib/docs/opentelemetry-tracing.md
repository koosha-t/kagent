# OpenTelemetry Tracing in Kagent

This guide explains how to collect and view traces from Kagent ADK agents using the in-cluster OpenTelemetry Collector.

## Overview

Kagent agents built with Google ADK automatically generate OpenTelemetry traces. The OTel Collector receives these traces and exports them to **Jaeger** for visual analysis and console logs for debugging. This provides out-of-the-box trace visualization with zero agent configuration required.

## Architecture

```
ADK Agent Pods
    ↓ (OTLP gRPC/HTTP)
OTel Collector Service (kagent-otel-collector.kagent:4317)
    ↓ (processing pipeline: memory_limiter → batch → attributes)
    ├─→ Console/Stdout (logs with sampling)
    └─→ Jaeger (kagent-jaeger.kagent:4317 via OTLP)
             ↓
        Jaeger UI :16686 (trace visualization)
```

**Components:**
- **ADK Agents**: Automatically emit traces via OpenTelemetry SDK
- **OTel Collector**: Receives, processes, and exports traces
- **Jaeger**: Stores and visualizes traces with a web UI
- **Exporters**: Dual export to both console (debugging) and Jaeger (visual analysis)

## Quick Start

### 1. Deploy Kagent with OTel Collector

The OTel Collector is now **integrated into the Helm chart** and deploys automatically:

```bash
# Deploy kagent with OTel Collector enabled (default)
helm upgrade --install kagent ./helm/kagent --namespace kagent

# Or using the kinagent Makefile
cd kinagent
make aks-deploy-all
```

Verify deployment:
```bash
kubectl get deployment -n kagent
kubectl get service -n kagent
kubectl logs -n kagent deployment/kagent-otel-collector -f
```

### 2. Deploy Agents

Deploy agents as usual - they automatically connect to the OTel Collector:

```bash
# Deploy the example supply-chain-planner agent
kubectl apply -f kinagent/supply-chain-planner.yaml

# Or restart existing agents to pick up the configuration
kubectl rollout restart deployment -n kagent -l app.kagent.dev/type=agent
```

### 3. Disable OTel Collector (Optional)

To deploy kagent without the OTel Collector:

```bash
helm upgrade --install kagent ./helm/kagent \
  --namespace kagent \
  --set otelCollector.enabled=false \
  --set otel.tracing.enabled=false
```

## Viewing Traces

### Console Output

View traces in the OTel Collector logs:

```bash
# Follow collector logs (note: service name includes release name prefix)
kubectl logs -n kagent deployment/kagent-otel-collector -f

# Filter for trace data
kubectl logs -n kagent deployment/kagent-otel-collector | grep -A 20 "Trace ID"
```

### Jaeger UI (Recommended)

Kagent includes Jaeger for visual trace analysis. Jaeger is **enabled by default** and provides a rich UI for exploring traces.

#### Accessing Jaeger

```bash
# Port-forward to the Jaeger UI
kubectl port-forward -n kagent svc/kagent-jaeger 16686:16686

# Open in your browser
open http://localhost:16686
```

#### Using Jaeger UI

Once connected, you can:

1. **Search Traces**:
   - Select service (e.g., `supply-chain-planner`)
   - Choose operation (e.g., `agent.execute`, `tool.call`)
   - Set time range and click "Find Traces"

2. **View Trace Details**:
   - Click on any trace to see the complete span timeline
   - Expand spans to see attributes (LLM prompts, tool parameters, timing)
   - View service dependencies and call graphs

3. **Analyze Performance**:
   - Compare trace durations
   - Identify slow operations
   - View LLM token usage across traces

4. **Service Map**:
   - Navigate to "Dependencies" tab
   - Visualize agent → tool → LLM relationships
   - See request volumes and error rates

#### Example Trace in Jaeger

When you interact with an agent, you'll see traces like:

```
Trace ID: abc123...
Duration: 2.4s
Spans: 8

├─ agent.execute (2.4s)
│  ├─ tool.search_events (0.8s)
│  │  └─ http.request POST /mcp (0.7s)
│  └─ llm.openai.chat.completions (1.2s)
│     └─ http.request POST /v1/chat/completions (1.1s)
```

Each span includes attributes:
- `gen_ai.agent.name`: Agent identifier
- `gen_ai.tool.name`: Tool called
- `gen_ai.request.model`: LLM model used
- `gen_ai.usage.input_tokens`: Tokens consumed
- `http.status_code`: Response status

#### Disabling Jaeger

To run Kagent without Jaeger (lighter deployment):

```bash
helm upgrade --install kagent ./helm/kagent \
  --namespace kagent \
  --set jaeger.enabled=false
```

Traces will still be collected and visible in OTel Collector console logs.

### What Traces Include

- **Agent execution spans**: Overall agent invocation timing
- **Tool calls**: MCP tool executions (e.g., `search_events`)
- **LLM API calls**: OpenAI/Anthropic/Google API requests with timing and token usage
- **HTTP requests**: Outbound HTTP calls from agents
- **Framework-specific spans**: CrewAI tasks, LangGraph state transitions

Example trace output:
```
Trace ID: 1234567890abcdef
Span ID: abcdef1234567890
Name: agent.execute
Attributes:
  gen_ai.agent.name: supply-chain-planner
  gen_ai.conversation.id: session-xyz
  gen_ai.tool.name: search_events
  http.method: POST
  http.status_code: 200
```

## Configuration

### Default Settings

Tracing and the OTel Collector are enabled by default via `helm/kagent/values.yaml`:

```yaml
otel:
  tracing:
    enabled: true
    exporter:
      otlp:
        endpoint: ""  # Auto-configured to: http://{{ release-name }}-otel-collector.{{ namespace }}:4317
        timeout: 15
        insecure: true

otelCollector:
  enabled: true
  replicas: 1
  service:
    ports:
      otlpGrpc: 4317
      otlpHttp: 4318
```

The endpoint is automatically constructed by the Helm chart as:
`http://{{ release-name }}-otel-collector.{{ namespace }}:4317`

For a release named "kagent" in namespace "kagent", this becomes:
`http://kagent-otel-collector.kagent:4317`

### Per-Agent Configuration

To disable tracing for a specific agent, add environment variables to the Agent spec:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: my-agent
spec:
  declarative:
    env:
      - name: OTEL_TRACING_ENABLED
        value: "false"
```

### OTel Collector Pipeline

The collector (deployed via Helm templates in `helm/kagent/templates/otel-collector-*.yaml`) is configured with:

- **Receivers**: OTLP (gRPC port 4317, HTTP port 4318)
- **Processors**:
  - `memory_limiter`: Prevents OOM (512 MiB limit)
  - `batch`: Batches spans for efficiency (10s timeout)
  - `attributes`: Adds Kubernetes context
- **Exporters**:
  - `logging`: Console output with sampling

## How It Works

### Agent-Side

1. ADK agents have OpenTelemetry instrumentation built-in
2. The kagent controller injects OTel environment variables into agent pods
3. Agents automatically send traces via OTLP to the collector endpoint

### Controller-Side

The controller (`go/internal/controller/translator/agent/adk_api_translator.go:313`) collects OTEL_* environment variables and injects them into agent deployments via `collectOtelEnvFromProcess()`.

### Environment Variables Injected

```bash
OTEL_TRACING_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector.kagent:4317
OTEL_EXPORTER_OTLP_TRACES_INSECURE=true
OTEL_LOGGING_ENABLED=true
```

## Troubleshooting

### Collector Not Receiving Traces

1. Check collector is running:
   ```bash
   kubectl get pods -n kagent -l app.kubernetes.io/component=otel-collector
   ```

2. Check service DNS resolution from agent pod:
   ```bash
   kubectl exec -n kagent <agent-pod> -- nslookup kagent-otel-collector.kagent
   ```

3. Check collector logs for errors:
   ```bash
   kubectl logs -n kagent deployment/kagent-otel-collector
   ```

### Agent Not Sending Traces

1. Check agent pod environment variables:
   ```bash
   kubectl exec -n kagent <agent-pod> -- env | grep OTEL
   ```

   Expected output:
   ```
   OTEL_TRACING_ENABLED=true
   OTEL_EXPORTER_OTLP_ENDPOINT=http://kagent-otel-collector.kagent:4317
   OTEL_EXPORTER_OTLP_TRACES_INSECURE=true
   OTEL_LOGGING_ENABLED=true
   ```

2. Check agent logs for OTel initialization:
   ```bash
   kubectl logs -n kagent <agent-pod> | grep -i otel
   ```

3. Verify agent is ADK-based (tracing currently only works with ADK/LangGraph/CrewAI agents)

### No Traces in Collector Logs

- Ensure you've triggered an agent execution (traces are only generated during agent runs)
- Check sampling settings in values.yaml under `otelCollector.config.exporters.logging`
- Increase log verbosity by setting `otelCollector.config.exporters.logging.loglevel: debug` in values.yaml

## Future Enhancements

### Add Persistent Storage (Tempo)

To store and query traces persistently:

1. Deploy Tempo backend
2. Update the collector configuration in values.yaml to add Tempo exporter:

```yaml
exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, attributes]
      exporters: [logging, otlp/tempo]
```

### Add Grafana Dashboard

Create a Grafana dashboard for trace visualization:
- Trace search by agent name, session ID
- Latency histograms for tool calls and LLM requests
- Error traces with detailed context
- Agent-to-agent communication flows

### Add Metrics

Export OTel metrics to Prometheus:
- Agent execution duration
- Tool call success rates
- LLM token usage
- Error rates by agent type

## Resources

- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OTLP Protocol Specification](https://opentelemetry.io/docs/specs/otlp/)
- [Google ADK Observability](https://cloud.google.com/vertex-ai/generative-ai/docs/agent-builder/observability)
- [Kagent Python Core Tracing](../python/packages/kagent-core/src/kagent/core/tracing/)
