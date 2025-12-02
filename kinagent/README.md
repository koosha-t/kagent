# Deploy Kagent with Azure OpenAI

## Setup

1. Configure environment variables:

```bash
cp kinagent/.env.template kinagent/.env
```

Edit `kinagent/.env` with your credentials:
```bash
ACR_USERNAME=your-acr-username
ACR_PASSWORD=your-acr-password
AZUREOPENAI_API_KEY=your-azure-openai-api-key
AZUREOPENAI_ENDPOINT=https://your-resource.openai.azure.com/
AZUREOPENAI_DEPLOYMENT=your-deployment-name
```

2. Connect to AKS:

Make sure you're connected to the AKS you want to deploy in (i.e. the kubectl works).

## Deploy

**Build images and deploy (~40 minutes):**
```bash
make aks-deploy-all
```

**Deploy with existing images (~2 minutes):**
```bash
# Auto-detect version from git
make aks-deploy-only

# Or specify a version
make VERSION=v0.0.0-70db161 aks-deploy-only
```

## Access

```bash
make aks-port-forward-ui
# Visit http://localhost:8082
```

## Tracing with Jaeger

Kagent supports distributed tracing via OpenTelemetry. Traces are automatically sent to Jaeger when you deploy with `make helm-install-aks` or `make aks-deploy-all`.

### Setup Tracing

1. **Deploy Jaeger** (one-time setup):
```bash
make aks-deploy-jaeger
```

This deploys Jaeger all-in-one to your AKS cluster with:
- OTLP receiver (gRPC on port 4317, HTTP on port 4318)
- In-memory storage
- Query UI on port 16686

2. **Deploy or update kagent**:
```bash
# For new deployments (includes building images)
make aks-deploy-all

# For updating existing deployments (uses existing images)
make helm-install-aks
```

**Note:** Tracing is automatically enabled by the Makefile (see `helm-install-aks` target, lines 576-579). No manual configuration needed.

### Access Jaeger UI

```bash
make aks-port-forward-jaeger
# Visit http://localhost:16686
```

### What Gets Traced

- Agent lifecycle events (creation, execution, completion)
- Tool invocations (Kubernetes, Helm, Istio, etc.)
- LLM API calls (OpenAI, Anthropic, Azure OpenAI, etc.)
- Agent-to-agent communication (A2A protocol)
- HTTP requests and responses

**Note:** Only tracing is enabled by default. LLM request/response logging is disabled to minimize overhead.

### Viewing Traces

1. Open Jaeger UI at http://localhost:16686
2. Select a service from the dropdown (e.g., `kagent`)
3. Click "Find Traces" to see recent traces
4. Click on a trace to see the full span tree with timing information

### Troubleshooting

**No traces appearing:**
- Verify Jaeger is running: `kubectl get pods -n kagent -l app.kubernetes.io/name=jaeger`
- Check controller logs: `kubectl logs -n kagent -l app.kubernetes.io/component=controller`
- Verify tracing is enabled in values: `otel.tracing.enabled: true`

**Jaeger pod not starting:**
- Check pod status: `kubectl describe pod -n kagent -l app.kubernetes.io/name=jaeger`
- View pod logs: `kubectl logs -n kagent -l app.kubernetes.io/name=jaeger`

## Uninstall

```bash
make helm-uninstall-aks
```
