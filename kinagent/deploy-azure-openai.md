# Deploy Kagent with Azure OpenAI

## Build and Push Images to ACR

```bash
# Build images locally
make controller-manifests
make buildx-create
make build-controller DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="localhost:5001"
make build-ui DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="localhost:5001"
make build-kagent-adk DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="localhost:5001"

# Build app image with regular docker
export VERSION=$(git rev-parse --short HEAD)
docker build --platform linux/arm64 \
  --build-arg VERSION=v0.0.0-$VERSION \
  --build-arg KAGENT_ADK_VERSION=v0.0.0-$VERSION \
  --build-arg DOCKER_REGISTRY=localhost:5001 \
  --build-arg DOCKER_REPO=kagent-dev/kagent \
  -t localhost:5001/kagent-dev/kagent/app:v0.0.0-$VERSION \
  -f python/Dockerfile.app ./python

# Login and push to ACR
az acr login --name obscr
export ACR_REGISTRY="obscr.azurecr.io"

docker tag localhost:5001/kagent-dev/kagent/controller:v0.0.0-$VERSION $ACR_REGISTRY/kagent-dev/kagent/controller:v0.0.0-$VERSION
docker tag localhost:5001/kagent-dev/kagent/ui:v0.0.0-$VERSION $ACR_REGISTRY/kagent-dev/kagent/ui:v0.0.0-$VERSION
docker tag localhost:5001/kagent-dev/kagent/app:v0.0.0-$VERSION $ACR_REGISTRY/kagent-dev/kagent/app:v0.0.0-$VERSION
docker tag localhost:5001/kagent-dev/kagent/kagent-adk:v0.0.0-$VERSION $ACR_REGISTRY/kagent-dev/kagent/kagent-adk:v0.0.0-$VERSION

docker push $ACR_REGISTRY/kagent-dev/kagent/controller:v0.0.0-$VERSION
docker push $ACR_REGISTRY/kagent-dev/kagent/ui:v0.0.0-$VERSION
docker push $ACR_REGISTRY/kagent-dev/kagent/app:v0.0.0-$VERSION
docker push $ACR_REGISTRY/kagent-dev/kagent/kagent-adk:v0.0.0-$VERSION
```

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
  -f kinagent/kinagent-values.yaml \
  --set registry=$ACR_REGISTRY \
  --set tag=v0.0.0-$VERSION \
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
