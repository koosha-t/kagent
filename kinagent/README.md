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
