# Deploy Kagent with Azure OpenAI

## Installation

```bash
# Build Helm charts (required before first installation)
make helm-version

# Create namespace
kubectl create namespace kagent

# Install CRDs
helm install kagent-crds ./helm/kagent-crds/ --namespace kagent

# Set your Azure OpenAI credentials
export AZUREOPENAI_API_KEY=your-azure-api-key

# Install Kagent with Azure OpenAI
helm install kagent ./helm/kagent/ \
  --namespace kagent \
  --set providers.default=azureOpenAI \
  --set providers.azureOpenAI.apiKey=$AZUREOPENAI_API_KEY \
  --set providers.azureOpenAI.config.apiVersion=2023-05-15 \
  --set providers.azureOpenAI.config.azureEndpoint=https://your-resource.openai.azure.com \
  --set providers.azureOpenAI.config.azureDeployment=your-deployment-name
```

## Verify Installation

```bash
# Check pods are running
kubectl get pods -n kagent

# Check the ModelConfig resource
kubectl get modelconfig -n kagent

# Access the UI
kubectl port-forward svc/kagent-ui 8080:8080 -n kagent
# Visit http://localhost:8080
```
