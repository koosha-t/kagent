#!/bin/bash

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}Kagent Deployment Script for AKS${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""

# Check if .env file exists
if [ ! -f "$SCRIPT_DIR/.env" ]; then
    echo -e "${RED}ERROR: .env file not found!${NC}"
    echo -e "${YELLOW}Please copy .env.template to .env and fill in your credentials:${NC}"
    echo -e "  cp $SCRIPT_DIR/.env.template $SCRIPT_DIR/.env"
    exit 1
fi

# Load environment variables from .env file
echo -e "${GREEN}Loading environment variables from .env...${NC}"
set -a
source "$SCRIPT_DIR/.env"
set +a

# Validate required environment variables
REQUIRED_VARS=(
    "ACR_NAME"
    "ACR_USERNAME"
    "ACR_PASSWORD"
    "NAMESPACE"
    "AZUREOPENAI_API_KEY"
    "AZUREOPENAI_ENDPOINT"
    "AZUREOPENAI_DEPLOYMENT"
    "AZUREOPENAI_API_VERSION"
)

for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var}" ]; then
        echo -e "${RED}ERROR: $var is not set in .env file${NC}"
        exit 1
    fi
done

echo -e "${GREEN}All required environment variables are set${NC}"
echo ""

# Check prerequisites
echo -e "${GREEN}Checking prerequisites...${NC}"

if ! command -v az &> /dev/null; then
    echo -e "${RED}ERROR: Azure CLI (az) is not installed${NC}"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}ERROR: kubectl is not installed${NC}"
    exit 1
fi

if ! command -v helm &> /dev/null; then
    echo -e "${RED}ERROR: helm is not installed${NC}"
    exit 1
fi

# Check kubectl connection
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}ERROR: kubectl is not configured or cannot connect to cluster${NC}"
    exit 1
fi

echo -e "${GREEN}All prerequisites are met${NC}"
echo ""

# Login to Azure Container Registry
echo -e "${GREEN}Logging into Azure Container Registry ($ACR_NAME)...${NC}"
if az acr login --name "$ACR_NAME"; then
    echo -e "${GREEN}Successfully logged into ACR${NC}"
else
    echo -e "${RED}ERROR: Failed to login to ACR. Please check your Azure credentials${NC}"
    exit 1
fi
echo ""

# Create namespace if it doesn't exist
echo -e "${GREEN}Creating namespace '$NAMESPACE' if it doesn't exist...${NC}"
if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo -e "${YELLOW}Namespace '$NAMESPACE' already exists${NC}"
else
    kubectl create namespace "$NAMESPACE"
    echo -e "${GREEN}Namespace '$NAMESPACE' created${NC}"
fi
echo ""

# Create or update ACR secret
echo -e "${GREEN}Creating ACR pull secret...${NC}"
ACR_SECRET_NAME="acr-secret"

# Delete existing secret if it exists
if kubectl get secret "$ACR_SECRET_NAME" -n "$NAMESPACE" &> /dev/null; then
    echo -e "${YELLOW}Deleting existing ACR secret...${NC}"
    kubectl delete secret "$ACR_SECRET_NAME" -n "$NAMESPACE"
fi

# Create new secret
kubectl create secret docker-registry "$ACR_SECRET_NAME" \
    --namespace "$NAMESPACE" \
    --docker-server="${ACR_NAME}.azurecr.io" \
    --docker-username="$ACR_USERNAME" \
    --docker-password="$ACR_PASSWORD"

echo -e "${GREEN}ACR pull secret created successfully${NC}"
echo ""

# Check if CRDs are already installed
echo -e "${GREEN}Checking if Kagent CRDs are already installed...${NC}"
CRDS_INSTALLED=false

if kubectl get crd agents.kagent.dev &> /dev/null; then
    echo -e "${YELLOW}Kagent CRDs are already installed${NC}"
    CRDS_INSTALLED=true
else
    echo -e "${GREEN}CRDs not found, will install them${NC}"
fi
echo ""

# Build Helm charts
echo -e "${GREEN}Building Helm charts...${NC}"
cd "$PROJECT_ROOT"
if make helm-version; then
    echo -e "${GREEN}Helm charts built successfully${NC}"
else
    echo -e "${RED}ERROR: Failed to build Helm charts${NC}"
    exit 1
fi
echo ""

# Install CRDs if not already installed
if [ "$CRDS_INSTALLED" = false ]; then
    echo -e "${GREEN}Installing Kagent CRDs...${NC}"
    if helm install kagent-crds ./helm/kagent-crds/ --namespace "$NAMESPACE"; then
        echo -e "${GREEN}Kagent CRDs installed successfully${NC}"
    else
        echo -e "${RED}ERROR: Failed to install Kagent CRDs${NC}"
        exit 1
    fi
    echo ""
fi

# Install Kagent
echo -e "${GREEN}Installing Kagent with custom configuration...${NC}"
cd "$PROJECT_ROOT"

helm install kagent ./helm/kagent/ \
    --namespace "$NAMESPACE" \
    --values "$SCRIPT_DIR/kinagent-values.yaml" \
    --set "imagePullSecrets[0].name=$ACR_SECRET_NAME" \
    --set "providers.default=azureOpenAI" \
    --set "providers.azureOpenAI.apiKey=$AZUREOPENAI_API_KEY" \
    --set "providers.azureOpenAI.model=${AZUREOPENAI_MODEL:-gpt-4.1-mini}" \
    --set "providers.azureOpenAI.config.apiVersion=$AZUREOPENAI_API_VERSION" \
    --set "providers.azureOpenAI.config.azureEndpoint=$AZUREOPENAI_ENDPOINT" \
    --set "providers.azureOpenAI.config.azureDeployment=$AZUREOPENAI_DEPLOYMENT"

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}Kagent installed successfully!${NC}"
    echo -e "${GREEN}=====================================${NC}"
    echo ""
else
    echo -e "${RED}ERROR: Failed to install Kagent${NC}"
    exit 1
fi

# Verify installation
echo -e "${GREEN}Verifying installation...${NC}"
echo ""

echo -e "${YELLOW}Waiting for pods to start (this may take a minute)...${NC}"
sleep 10

echo -e "${GREEN}Current pod status:${NC}"
kubectl get pods -n "$NAMESPACE"
echo ""

echo -e "${GREEN}ModelConfig resources:${NC}"
kubectl get modelconfig -n "$NAMESPACE"
echo ""

# Display next steps
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}Next Steps:${NC}"
echo -e "${GREEN}=====================================${NC}"
echo ""
echo -e "1. Check pod status:"
echo -e "   ${YELLOW}kubectl get pods -n $NAMESPACE${NC}"
echo ""
echo -e "2. View logs for controller:"
echo -e "   ${YELLOW}kubectl logs -f deployment/kagent-controller -n $NAMESPACE${NC}"
echo ""
echo -e "3. Access the UI (port-forward):"
echo -e "   ${YELLOW}kubectl port-forward svc/kagent-ui 8080:8080 -n $NAMESPACE${NC}"
echo -e "   Then visit: ${YELLOW}http://localhost:8080${NC}"
echo ""
echo -e "4. Check ModelConfig:"
echo -e "   ${YELLOW}kubectl describe modelconfig -n $NAMESPACE${NC}"
echo ""
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}=====================================${NC}"
