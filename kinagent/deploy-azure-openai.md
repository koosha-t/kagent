# Deploy Kagent with Azure OpenAI

## Prerequisites

### 1. Set up environment variables

Copy the template and fill in your values:

```bash
cp kinagent/.env.template kinagent/.env
```

Edit `kinagent/.env` with your actual values:
- **ACR_USERNAME** and **ACR_PASSWORD**: Azure Container Registry credentials
- **AZUREOPENAI_API_KEY**: Your Azure OpenAI API key
- **AZUREOPENAI_ENDPOINT**: Your Azure OpenAI endpoint (e.g., `https://your-resource.openai.azure.com/`)
- **AZUREOPENAI_DEPLOYMENT**: Your deployment name

The Makefile will automatically load these variables from `kinagent/.env`.

## Build and Push Images to ACR

### Option 1: Using the build script (recommended)

```bash
./kinagent/build-and-push-acr.sh
```

### Option 2: Manual steps

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

## Configure ACR Access

The image pull secret will be created automatically when you run `make aks-deploy-all`.

If you need to create it manually:

```bash
# Make sure ACR_USERNAME and ACR_PASSWORD are set in kinagent/.env
make aks-create-acr-secret
```

## Installation

All environment variables are loaded automatically from `kinagent/.env`.

### Option 1: Build and Deploy (First Time)

Builds images, pushes to ACR, and deploys to AKS (~40 minutes):

```bash
make aks-deploy-all
```

### Option 2: Deploy Only (Using Pre-Built Images)

If images are already in ACR, skip building and deploy directly (~2 minutes):

```bash
# Deploy with auto-detected version from git
make aks-deploy-only

# Or specify a version that exists in ACR
make VERSION=v0.0.0-70db161 aks-deploy-only
```

**Find available versions in ACR:**
```bash
az acr repository show-tags --name obscr --repository kagent-dev/kagent/controller
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
