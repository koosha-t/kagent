#!/usr/bin/env bash

set -e  # Exit on error
set -u  # Exit on undefined variable

# Configuration
ACR_NAME="${ACR_NAME:-obscr}"
ACR_REGISTRY="${ACR_REGISTRY:-obscr.azurecr.io}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-localhost:5001}"
DOCKER_REPO="${DOCKER_REPO:-kagent-dev/kagent}"

# Get version from git
VERSION=$(git rev-parse --short HEAD)
IMAGE_TAG="v0.0.0-$VERSION"

echo "========================================"
echo "Building and Pushing Kagent Images to ACR"
echo "========================================"
echo "ACR Registry: $ACR_REGISTRY"
echo "Image Tag: $IMAGE_TAG"
echo "========================================"
echo ""

# Step 1: Build images
echo "Step 1: Building images locally..."
echo ""

echo "Building controller manifests..."
make controller-manifests

echo "Creating buildx builder..."
make buildx-create

echo "Building controller image..."
make build-controller DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="$DOCKER_REGISTRY"

echo "Building UI image..."
make build-ui DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="$DOCKER_REGISTRY"

echo "Building kagent-adk image..."
make build-kagent-adk DOCKER_BUILD_ARGS="--load --platform linux/arm64" DOCKER_REGISTRY="$DOCKER_REGISTRY"

echo "Building app image with regular docker..."
docker build --platform linux/arm64 \
  --build-arg VERSION=$IMAGE_TAG \
  --build-arg KAGENT_ADK_VERSION=$IMAGE_TAG \
  --build-arg DOCKER_REGISTRY=$DOCKER_REGISTRY \
  --build-arg DOCKER_REPO=$DOCKER_REPO \
  -t $DOCKER_REGISTRY/$DOCKER_REPO/app:$IMAGE_TAG \
  -f python/Dockerfile.app ./python

echo ""
echo "✓ All images built successfully"
echo ""

# Step 2: Login to ACR
echo "Step 2: Logging into Azure Container Registry..."
az acr login --name $ACR_NAME

echo ""
echo "✓ Logged in to ACR"
echo ""

# Step 3: Tag images for ACR
echo "Step 3: Tagging images for ACR..."
echo ""

docker tag $DOCKER_REGISTRY/$DOCKER_REPO/controller:$IMAGE_TAG $ACR_REGISTRY/$DOCKER_REPO/controller:$IMAGE_TAG
echo "✓ Tagged controller"

docker tag $DOCKER_REGISTRY/$DOCKER_REPO/ui:$IMAGE_TAG $ACR_REGISTRY/$DOCKER_REPO/ui:$IMAGE_TAG
echo "✓ Tagged ui"

docker tag $DOCKER_REGISTRY/$DOCKER_REPO/app:$IMAGE_TAG $ACR_REGISTRY/$DOCKER_REPO/app:$IMAGE_TAG
echo "✓ Tagged app"

docker tag $DOCKER_REGISTRY/$DOCKER_REPO/kagent-adk:$IMAGE_TAG $ACR_REGISTRY/$DOCKER_REPO/kagent-adk:$IMAGE_TAG
echo "✓ Tagged kagent-adk"

echo ""
echo "✓ All images tagged"
echo ""

# Step 4: Push images to ACR
echo "Step 4: Pushing images to ACR..."
echo ""

docker push $ACR_REGISTRY/$DOCKER_REPO/controller:$IMAGE_TAG
echo "✓ Pushed controller"

docker push $ACR_REGISTRY/$DOCKER_REPO/ui:$IMAGE_TAG
echo "✓ Pushed ui"

docker push $ACR_REGISTRY/$DOCKER_REPO/app:$IMAGE_TAG
echo "✓ Pushed app"

docker push $ACR_REGISTRY/$DOCKER_REPO/kagent-adk:$IMAGE_TAG
echo "✓ Pushed kagent-adk"

echo ""
echo "========================================"
echo "✓ All images successfully pushed to ACR!"
echo "========================================"
echo ""
echo "Images pushed:"
echo "  - $ACR_REGISTRY/$DOCKER_REPO/controller:$IMAGE_TAG"
echo "  - $ACR_REGISTRY/$DOCKER_REPO/ui:$IMAGE_TAG"
echo "  - $ACR_REGISTRY/$DOCKER_REPO/app:$IMAGE_TAG"
echo "  - $ACR_REGISTRY/$DOCKER_REPO/kagent-adk:$IMAGE_TAG"
echo ""
echo "To install with Helm, run:"
echo "  export VERSION=$VERSION"
echo "  helm install kagent ./helm/kagent/ \\"
echo "    --namespace kagent \\"
echo "    -f kinagent/kinagent-values.yaml \\"
echo "    --set registry=$ACR_REGISTRY \\"
echo "    --set tag=$IMAGE_TAG \\"
echo "    --set providers.default=azureOpenAI \\"
echo "    --set providers.azureOpenAI.apiKey=\$AZUREOPENAI_API_KEY \\"
echo "    --set providers.azureOpenAI.config.azureEndpoint=https://your-resource.openai.azure.com \\"
echo "    --set providers.azureOpenAI.config.azureDeployment=your-deployment-name"
echo ""
