# Deploy Kagent to AKS using Makefile

This guide shows how to deploy Kagent to Azure Kubernetes Service (AKS) using the new Makefile targets that handle building, pushing to ACR, and deploying to AKS with Azure OpenAI.

## Prerequisites

1. **Azure CLI** - Install from https://docs.microsoft.com/en-us/cli/azure/install-azure-cli
   ```bash
   az --version  # Verify installation
   az login      # Authenticate to Azure
   ```

2. **kubectl** - Configured to connect to your AKS cluster
   ```bash
   # Get AKS credentials
   az aks get-credentials --resource-group <resource-group> --name <aks-cluster-name>

   # Verify connection
   kubectl get nodes
   ```

3. **Docker** with buildx support
   ```bash
   docker buildx version
   ```

4. **Helm** (v3+)
   ```bash
   helm version
   ```

5. **ACR Access** - Ensure you have push access to the Azure Container Registry

6. **Azure OpenAI** - Have your Azure OpenAI API key ready

## Quick Start

### One-Command Deployment

The simplest way to deploy to AKS with Azure OpenAI:

```bash
# Set your Azure OpenAI API key
export AZUREOPENAI_API_KEY=your-azure-openai-api-key

# Deploy everything with one command
make aks-deploy-all
```

This will:
1. Authenticate to ACR
2. Build all images (controller, UI, app, kagent-adk) for both amd64 and arm64
3. Push images to ACR
4. Deploy kagent to your AKS cluster using Azure OpenAI by default

## Step-by-Step Deployment

If you prefer more control, you can run each step separately:

### Step 1: Build and Push Images to ACR

```bash
make build-acr
```

This target:
- Checks Azure CLI is installed
- Authenticates to ACR using `az acr login`
- Builds multi-platform images (amd64/arm64)
- Pushes them to the registry

### Step 2: Deploy to AKS

```bash
# Set your Azure OpenAI API key
export AZUREOPENAI_API_KEY=your-azure-openai-api-key

# Deploy to AKS (defaults to Azure OpenAI)
make helm-install-aks
```

This will install kagent to your AKS cluster in the `kagent` namespace with Azure OpenAI as the default provider.

## Configuration Options

### Using a Different ACR Registry

Override the default registry (`obscr.azurecr.io`):

```bash
ACR_REGISTRY=myregistry.azurecr.io make aks-deploy-all
```

### Using a Different Repository Path

```bash
ACR_REPO=my-team/kagent make aks-deploy-all
```

### Deploying to a Different Namespace

```bash
AKS_NAMESPACE=my-namespace make helm-install-aks
```

### Using a Different Model Provider

While AKS deployments default to Azure OpenAI, you can override this:

**Using OpenAI:**
```bash
export AKS_DEFAULT_MODEL_PROVIDER=openAI
export OPENAI_API_KEY=your-openai-api-key
make aks-deploy-all
```

**Using Anthropic:**
```bash
export AKS_DEFAULT_MODEL_PROVIDER=anthropic
export ANTHROPIC_API_KEY=your-anthropic-api-key
make aks-deploy-all
```

**Using Gemini:**
```bash
export AKS_DEFAULT_MODEL_PROVIDER=gemini
export GOOGLE_API_KEY=your-gemini-api-key
make aks-deploy-all
```

### Advanced Azure OpenAI Configuration

For more Azure OpenAI configuration options (endpoint, deployment name, API version):

```bash
# Build and push images
make build-acr

# Deploy with custom Azure OpenAI settings
export AZUREOPENAI_API_KEY=your-azure-api-key
make helm-version check-aks-api-key aks-check-context

helm upgrade --install kagent helm/kagent \
  --namespace kagent \
  --create-namespace \
  --set registry=obscr.azurecr.io/kagent-dev/kagent \
  --set tag=$(git describe --tags --always 2>/dev/null | grep v || echo "v0.0.0-$(git rev-parse --short HEAD)") \
  --set providers.default=azureOpenAI \
  --set providers.azureOpenAI.apiKey=$AZUREOPENAI_API_KEY \
  --set providers.azureOpenAI.config.apiVersion=2024-08-01-preview \
  --set providers.azureOpenAI.config.azureEndpoint=https://your-resource.openai.azure.com \
  --set providers.azureOpenAI.config.azureDeployment=your-deployment-name \
  --set providers.azureOpenAI.model=gpt-4o
```

### Using LoadBalancer Instead of ClusterIP

By default, services use ClusterIP (no public IP). To use LoadBalancer:

```bash
AKS_SERVICE_TYPE=LoadBalancer make helm-install-aks
```

**Note:** LoadBalancer services will create Azure Load Balancers which may incur additional costs.

## Accessing Kagent

### Access the UI

```bash
make aks-port-forward-ui
```

Then visit http://localhost:8082

### Access the CLI

```bash
make aks-port-forward-cli
```

The CLI will be available at `localhost:8083`

## Verify Installation

```bash
# Check all pods are running
kubectl get pods -n kagent

# Check the ModelConfig resource
kubectl get modelconfig -n kagent

# View ModelConfig details
kubectl describe modelconfig -n kagent

# Check services
kubectl get svc -n kagent

# View logs
kubectl logs -n kagent deployment/kagent-controller
kubectl logs -n kagent deployment/kagent-ui
```

## Updating Kagent

To update an existing installation with new images:

```bash
# Build and push new images
make build-acr

# Upgrade the Helm release
make helm-install-aks
```

## Uninstall

To remove kagent from your AKS cluster:

```bash
make helm-uninstall-aks
```

## Available Make Targets

| Target | Description |
|--------|-------------|
| `check-aks-api-key` | Check if required API key is set based on provider |
| `acr-login` | Authenticate to Azure Container Registry |
| `build-acr` | Build and push all images to ACR (multi-platform) |
| `helm-install-aks` | Deploy kagent to AKS using Helm |
| `helm-uninstall-aks` | Remove kagent from AKS |
| `aks-port-forward-ui` | Port forward UI to localhost:8082 |
| `aks-port-forward-cli` | Port forward CLI to localhost:8083 |
| `aks-check-context` | Verify kubectl context (safety check) |
| `aks-deploy-all` | Complete deployment (build + deploy) |

## Troubleshooting

### Azure CLI Not Found

```bash
# Install Azure CLI
# macOS
brew install azure-cli

# Linux (Debian/Ubuntu)
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Windows
# Download from https://aka.ms/installazurecliwindows
```

### ACR Authentication Failed

```bash
# Ensure you're logged in to Azure
az login

# Manually login to ACR
az acr login --name <registry-name>

# Check ACR access
az acr repository list --name <registry-name>
```

### Missing Azure OpenAI API Key

If you see: `Error: AZUREOPENAI_API_KEY environment variable is not set`

```bash
# Set your Azure OpenAI API key
export AZUREOPENAI_API_KEY=your-azure-openai-api-key

# Retry deployment
make helm-install-aks
```

### Wrong Kubectl Context

If you get an error about deploying to a Kind cluster:

```bash
# List available contexts
kubectl config get-contexts

# Switch to your AKS context
kubectl config use-context <your-aks-context>

# Verify current context
kubectl config current-context
```

### Image Pull Errors

If pods fail to pull images from ACR, you may need to configure image pull secrets or attach ACR to AKS.

**Option 1: Attach ACR to AKS (Recommended)**
```bash
az aks update \
  --name <aks-cluster-name> \
  --resource-group <resource-group> \
  --attach-acr <acr-name>
```

**Option 2: Create Image Pull Secret**
```bash
# Create a service principal (if you don't have one)
az ad sp create-for-rbac --name kagent-acr-sp --skip-assignment

# Grant AcrPull role
az role assignment create \
  --assignee <service-principal-app-id> \
  --role AcrPull \
  --scope /subscriptions/<subscription-id>/resourceGroups/<resource-group>/providers/Microsoft.ContainerRegistry/registries/<acr-name>

# Create image pull secret
kubectl create secret docker-registry acr-secret \
  --docker-server=<your-registry>.azurecr.io \
  --docker-username=<service-principal-app-id> \
  --docker-password=<service-principal-password> \
  --namespace kagent

# Update Helm installation to use the secret
helm upgrade kagent helm/kagent \
  --namespace kagent \
  --set imagePullSecrets[0].name=acr-secret \
  --reuse-values
```

### Build Platform Errors

If you encounter buildx errors:

```bash
# Recreate the builder
docker buildx rm kagent-builder-v0.23.0
docker buildx create --name kagent-builder-v0.23.0 \
  --platform linux/amd64,linux/arm64 \
  --driver docker-container \
  --use \
  --driver-opt network=host
```

### Azure OpenAI Connection Issues

If kagent can't connect to Azure OpenAI:

```bash
# Check ModelConfig status
kubectl get modelconfig -n kagent -o yaml

# View controller logs for errors
kubectl logs -n kagent deployment/kagent-controller --tail=100

# Verify Azure OpenAI endpoint and deployment name are correct
# Update if needed:
helm upgrade kagent helm/kagent \
  --namespace kagent \
  --set providers.azureOpenAI.config.azureEndpoint=https://correct-endpoint.openai.azure.com \
  --set providers.azureOpenAI.config.azureDeployment=correct-deployment-name \
  --reuse-values
```

## Advanced Configuration

### Custom Helm Values

Create a custom values file for AKS-specific settings:

```yaml
# aks-values.yaml
controller:
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "1Gi"
      cpu: "1000m"

ui:
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
  service:
    type: ClusterIP

providers:
  azureOpenAI:
    config:
      apiVersion: "2024-08-01-preview"
      azureEndpoint: "https://your-resource.openai.azure.com"
      azureDeployment: "gpt-4o"
    model: "gpt-4o"
```

Deploy with custom values:
```bash
make build-acr

helm upgrade --install kagent helm/kagent \
  --namespace kagent \
  --values aks-values.yaml \
  --set registry=obscr.azurecr.io/kagent-dev/kagent \
  --set tag=$(git describe --tags --always) \
  --set providers.azureOpenAI.apiKey=$AZUREOPENAI_API_KEY
```

### Using a Specific Image Tag

```bash
# Build with a specific tag
VERSION=v1.0.0 make build-acr

# Deploy with that tag
VERSION=v1.0.0 make helm-install-aks
```

### Enable KMCP (Kubernetes MCP Server)

```bash
KMCP_ENABLED=true make helm-install-aks
```

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `ACR_REGISTRY` | `obscr.azurecr.io` | Azure Container Registry URL |
| `ACR_REPO` | `kagent-dev/kagent` | Repository path in ACR |
| `AKS_SERVICE_TYPE` | `ClusterIP` | Kubernetes service type (ClusterIP or LoadBalancer) |
| `AKS_NAMESPACE` | `kagent` | Kubernetes namespace for deployment |
| `AKS_DEFAULT_MODEL_PROVIDER` | `azureOpenAI` | Default model provider for AKS |
| `AZUREOPENAI_API_KEY` | (required) | Azure OpenAI API key |
| `VERSION` | auto-detected from git | Image tag version |
| `KMCP_ENABLED` | `true` | Enable/disable KMCP |

## Comparison with Other Deployment Methods

This Makefile-based approach differs from the existing `deploy-azure-openai.md` guide:

| Feature | Makefile Approach | Script/Manual Approach |
|---------|-------------------|------------------------|
| Build platform | Multi-platform (amd64/arm64) | Single platform (arm64) |
| Steps | Automated (1-2 commands) | Manual (multiple steps) |
| ACR authentication | Automatic | Manual |
| Default provider | Azure OpenAI (for AKS) | Manual selection |
| Helm installation | Integrated | Separate |
| Safety checks | Built-in (context check, API key check) | Manual verification |
| Repeatability | High (idempotent) | Medium |

**Use the Makefile approach when:**
- You want a streamlined, repeatable deployment process
- You need multi-platform images
- You're deploying to AKS with Azure OpenAI (default use case)
- You're deploying frequently during development

**Use the manual approach when:**
- You need fine-grained control over each step
- You're troubleshooting build issues
- You have custom build requirements
- You're deploying to non-AKS Kubernetes clusters

## Next Steps

After deploying kagent to AKS:

1. **Test the UI**: Access via `make aks-port-forward-ui` and create a test agent
2. **Monitor resources**: Check pod status and logs regularly
3. **Set up monitoring**: Consider integrating with Azure Monitor or Prometheus
4. **Configure autoscaling**: Set up HPA for controller and UI deployments
5. **Secure access**: Consider using Azure Application Gateway or Ingress controllers
6. **Backup**: Regularly backup your agent configurations and CRDs

## Getting Help

- Check the [main README](../README.md) for general Kagent documentation
- See [DEVELOPMENT.md](../DEVELOPMENT.md) for local development setup
- Review [deploy-azure-openai.md](./deploy-azure-openai.md) for manual deployment steps
- File issues at https://github.com/kagent-dev/kagent/issues
