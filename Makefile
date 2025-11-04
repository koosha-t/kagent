# Image configuration
DOCKER_REGISTRY ?= localhost:5001
BASE_IMAGE_REGISTRY ?= cgr.dev
DOCKER_REPO ?= kagent-dev/kagent
HELM_REPO ?= oci://ghcr.io/kagent-dev
HELM_DIST_FOLDER ?= dist

BUILD_DATE := $(shell date -u '+%Y-%m-%d')
GIT_COMMIT := $(shell git rev-parse --short HEAD || echo "unknown")
VERSION ?= $(shell git describe --tags --always 2>/dev/null | grep v || echo "v0.0.0-$(GIT_COMMIT)")

# Local architecture detection to build for the current platform
LOCALARCH ?= $(shell uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')

KUBECONFIG_PERM ?= $(shell \
  if [ "$$(uname -s | tr '[:upper:]' '[:lower:]')" = "darwin" ]; then \
    stat -f "%Lp" ~/.kube/config; \
  else \
    stat -c "%a" ~/.kube/config; \
  fi)


# Docker buildx configuration
BUILDKIT_VERSION = v0.23.0
BUILDX_NO_DEFAULT_ATTESTATIONS=1
BUILDX_BUILDER_NAME ?= kagent-builder-$(BUILDKIT_VERSION)

DOCKER_BUILDER ?= docker buildx
DOCKER_BUILD_ARGS ?= --push --platform linux/$(LOCALARCH)

KIND_CLUSTER_NAME ?= kagent
KIND_IMAGE_VERSION ?= 1.34.0

# Azure Container Registry configuration
ACR_REGISTRY ?= obscr.azurecr.io
ACR_REPO ?= kagent-dev/kagent
ACR_BUILD_ARGS ?= --push --platform linux/amd64,linux/arm64
ACR_USERNAME ?=
ACR_PASSWORD ?=

# AKS deployment configuration
AKS_SERVICE_TYPE ?= ClusterIP
AKS_NAMESPACE ?= kagent
AKS_DEFAULT_MODEL_PROVIDER ?= azureOpenAI

# Azure OpenAI specific configuration
AZUREOPENAI_ENDPOINT ?=
AZUREOPENAI_DEPLOYMENT ?=
AZUREOPENAI_API_VERSION ?= 2024-08-01-preview
AZUREOPENAI_MODEL ?= gpt-4o

# Load environment variables from kinagent/.env if it exists
ifneq (,$(wildcard kinagent/.env))
    include kinagent/.env
    export
endif

CONTROLLER_IMAGE_NAME ?= controller
UI_IMAGE_NAME ?= ui
APP_IMAGE_NAME ?= app
KAGENT_ADK_IMAGE_NAME ?= kagent-adk

CONTROLLER_IMAGE_TAG ?= $(VERSION)
UI_IMAGE_TAG ?= $(VERSION)
APP_IMAGE_TAG ?= $(VERSION)
KAGENT_ADK_IMAGE_TAG ?= $(VERSION)

CONTROLLER_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)/$(CONTROLLER_IMAGE_NAME):$(CONTROLLER_IMAGE_TAG)
UI_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG)
APP_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)/$(APP_IMAGE_NAME):$(APP_IMAGE_TAG)
KAGENT_ADK_IMG ?= $(DOCKER_REGISTRY)/$(DOCKER_REPO)/$(KAGENT_ADK_IMAGE_NAME):$(KAGENT_ADK_IMAGE_TAG)

#take from go/go.mod
AWK ?= $(shell command -v gawk || command -v awk)
TOOLS_GO_VERSION ?= $(shell $(AWK) '/^go / { print $$2 }' go/go.mod)
export GOTOOLCHAIN=go$(TOOLS_GO_VERSION)

# Version information for the build
LDFLAGS := "-X github.com/$(DOCKER_REPO)/go/internal/version.Version=$(VERSION)      \
            -X github.com/$(DOCKER_REPO)/go/internal/version.GitCommit=$(GIT_COMMIT) \
            -X github.com/$(DOCKER_REPO)/go/internal/version.BuildDate=$(BUILD_DATE)"

#tools versions
TOOLS_UV_VERSION ?= 0.8.22
TOOLS_BUN_VERSION ?= 1.2.22
TOOLS_NODE_VERSION ?= 22.19.0
TOOLS_PYTHON_VERSION ?= 3.13

# build args
TOOLS_IMAGE_BUILD_ARGS =  --build-arg VERSION=$(VERSION)
TOOLS_IMAGE_BUILD_ARGS += --build-arg LDFLAGS=$(LDFLAGS)
TOOLS_IMAGE_BUILD_ARGS += --build-arg DOCKER_REPO=$(DOCKER_REPO)
TOOLS_IMAGE_BUILD_ARGS += --build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY)
TOOLS_IMAGE_BUILD_ARGS += --build-arg BASE_IMAGE_REGISTRY=$(BASE_IMAGE_REGISTRY)
TOOLS_IMAGE_BUILD_ARGS += --build-arg TOOLS_GO_VERSION=$(TOOLS_GO_VERSION)
TOOLS_IMAGE_BUILD_ARGS += --build-arg TOOLS_UV_VERSION=$(TOOLS_UV_VERSION)
TOOLS_IMAGE_BUILD_ARGS += --build-arg TOOLS_BUN_VERSION=$(TOOLS_BUN_VERSION)
TOOLS_IMAGE_BUILD_ARGS += --build-arg TOOLS_PYTHON_VERSION=$(TOOLS_PYTHON_VERSION)
TOOLS_IMAGE_BUILD_ARGS += --build-arg TOOLS_NODE_VERSION=$(TOOLS_NODE_VERSION)

# KMCP 
KMCP_ENABLED ?= true
KMCP_VERSION ?= $(shell $(AWK) '/github\.com\/kagent-dev\/kmcp/ { print substr($$2, 2) }' go/go.mod) # KMCP version defaults to what's referenced in go.mod

HELM_ACTION=upgrade --install

# Helm chart variables
KAGENT_DEFAULT_MODEL_PROVIDER ?= openAI

# Print tools versions
print-tools-versions:
	@echo "VERSION      : $(VERSION)"
	@echo "Tools Go     : $(TOOLS_GO_VERSION)"
	@echo "Tools UV     : $(TOOLS_UV_VERSION)"
	@echo "Tools Node   : $(TOOLS_NODE_VERSION)"
	@echo "Tools Istio  : $(TOOLS_ISTIO_VERSION)"
	@echo "Tools Argo CD: $(TOOLS_ARGO_CD_VERSION)"

# Check if the appropriate API key is set based on the model provider
check-api-key:
	@if [ "$(KAGENT_DEFAULT_MODEL_PROVIDER)" = "openAI" ]; then \
		if [ -z "$(OPENAI_API_KEY)" ]; then \
			echo "Error: OPENAI_API_KEY environment variable is not set for OpenAI provider"; \
			echo "Please set it with: export OPENAI_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(KAGENT_DEFAULT_MODEL_PROVIDER)" = "anthropic" ]; then \
		if [ -z "$(ANTHROPIC_API_KEY)" ]; then \
			echo "Error: ANTHROPIC_API_KEY environment variable is not set for Anthropic provider"; \
			echo "Please set it with: export ANTHROPIC_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(KAGENT_DEFAULT_MODEL_PROVIDER)" = "azureOpenAI" ]; then \
		if [ -z "$(AZUREOPENAI_API_KEY)" ]; then \
			echo "Error: AZUREOPENAI_API_KEY environment variable is not set for Azure OpenAI provider"; \
			echo "Please set it with: export AZUREOPENAI_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(KAGENT_DEFAULT_MODEL_PROVIDER)" = "gemini" ]; then \
		if [ -z "$(GOOGLE_API_KEY)" ]; then \
			echo "Error: GOOGLE_API_KEY environment variable is not set for Gemini provider"; \
			echo "Please set it with: export GOOGLE_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(KAGENT_DEFAULT_MODEL_PROVIDER)" = "ollama" ]; then \
		echo "Note: Ollama provider does not require an API key"; \
	else \
		echo "Warning: Unknown model provider '$(KAGENT_DEFAULT_MODEL_PROVIDER)'. Skipping API key check."; \
	fi

.PHONY: buildx-create
buildx-create:
	docker buildx inspect $(BUILDX_BUILDER_NAME) 2>&1 > /dev/null || \
	docker buildx create --name $(BUILDX_BUILDER_NAME) --platform linux/amd64,linux/arm64 --driver docker-container --use --driver-opt network=host || true

.PHONY: build-all  # for test purpose build all but output to /dev/null
build-all: BUILD_ARGS ?= --progress=plain --builder $(BUILDX_BUILDER_NAME) --platform linux/amd64,linux/arm64 --output type=tar,dest=/dev/null
build-all: buildx-create
	$(DOCKER_BUILDER) build $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -f go/Dockerfile     ./go
	$(DOCKER_BUILDER) build $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -f ui/Dockerfile     ./ui
	$(DOCKER_BUILDER) build $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -f python/Dockerfile ./python

.PHONY: push-test-agent
push-test-agent: buildx-create build-kagent-adk
	echo "Building FROM DOCKER_REGISTRY=$(DOCKER_REGISTRY)/$(DOCKER_REPO)/kagent-adk:$(VERSION)"
	$(DOCKER_BUILDER) build --push $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(DOCKER_REGISTRY)/kebab:latest -f go/test/e2e/agents/kebab/Dockerfile ./go/test/e2e/agents/kebab
	kubectl apply --namespace kagent --context kind-$(KIND_CLUSTER_NAME) -f go/test/e2e/agents/kebab/agent.yaml
	$(DOCKER_BUILDER) build --push $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(DOCKER_REGISTRY)/poem-flow:latest -f python/samples/crewai/poem_flow/Dockerfile ./python

.PHONY: push-test-skill
push-test-skill: buildx-create
	echo "Building FROM DOCKER_REGISTRY=$(DOCKER_REGISTRY)/$(DOCKER_REPO)/kebab-maker:$(VERSION)"
	$(DOCKER_BUILDER) build --push $(BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(DOCKER_REGISTRY)/kebab-maker:latest -f go/test/e2e/testdata/skills/kebab/Dockerfile ./go/test/e2e/testdata/skills/kebab

.PHONY: create-kind-cluster
create-kind-cluster:
	bash ./scripts/kind/setup-kind.sh
	bash ./scripts/kind/setup-metallb.sh

.PHONY: use-kind-cluster
use-kind-cluster:
	kind get kubeconfig --name $(KIND_CLUSTER_NAME) > /tmp/kind-config
	KUBECONFIG=~/.kube/config:/tmp/kind-config kubectl config view --merge --flatten > ~/.kube/config.tmp && mv ~/.kube/config.tmp ~/.kube/config && chmod $(KUBECONFIG_PERM) ~/.kube/config
	kubectl create namespace kagent || true
	kubectl config set-context --current --namespace kagent || true

.PHONY: delete-kind-cluster
delete-kind-cluster:
	kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: clean
clean: prune-kind-cluster
clean: prune-docker-images
	docker buildx rm $(BUILDX_BUILDER_NAME)  -f || true
	rm -rf ./go/bin

.PHONY: prune-kind-cluster
prune-kind-cluster:
	echo "Pruning dangling docker images from kind  ..."
	docker exec $(KIND_CLUSTER_NAME)-control-plane crictl images --no-trunc --quiet | \
	grep '<none>' | awk '{print $3}' | xargs -r -n1 docker exec $(KIND_CLUSTER_NAME)-control-plane crictl rmi || :

.PHONY: prune-docker-images
prune-docker-images:
	echo "Pruning dangling docker images ..."
	docker images --format '{{.Repository}}:{{.Tag}} {{.ID}}' | \
	grep -v ":$(VERSION) " | grep kagent | grep -v '<none>' | awk '{print $2}' | xargs -r docker rmi || :
	docker images --filter dangling=true -q | xargs -r docker rmi || :

.PHONY: build
build: buildx-create build-controller build-ui build-app
	@echo "Build completed successfully."
	@echo "Controller Image: $(CONTROLLER_IMG)"
	@echo "UI Image: $(UI_IMG)"
	@echo "App Image: $(APP_IMG)"
	@echo "Kagent ADK Image: $(KAGENT_ADK_IMG)"
	@echo "Tools Image: $(TOOLS_IMG)"

.PHONY: build-monitor
build-monitor: buildx-create
	watch docker exec -t  buildx_buildkit_$(BUILDX_BUILDER_NAME)0  ps

.PHONY: build-cli
build-cli:
	make -C go build

.PHONY: build-cli-local
build-cli-local:
	make -C go clean
	make -C go bin/kagent-local

.PHONY: build-img-versions
build-img-versions:
	@echo controller=$(CONTROLLER_IMG)
	@echo ui=$(UI_IMG)
	@echo app=$(APP_IMG)
	@echo kagent-adk=$(KAGENT_ADK_IMG)

.PHONY: lint
lint:
	make -C go lint
	make -C python lint

.PHONY: push
push: push-controller push-ui push-app push-kagent-adk

.PHONY: controller-manifests
controller-manifests:
	make -C go manifests
	cp go/config/crd/bases/* helm/kagent-crds/templates/

.PHONY: build-controller
build-controller: buildx-create controller-manifests
	$(DOCKER_BUILDER) build $(DOCKER_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(CONTROLLER_IMG) -f go/Dockerfile ./go

.PHONY: build-ui
build-ui: buildx-create
	$(DOCKER_BUILDER) build $(DOCKER_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(UI_IMG) -f ui/Dockerfile ./ui

.PHONY: build-kagent-adk
build-kagent-adk: buildx-create
		$(DOCKER_BUILDER) build $(DOCKER_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(KAGENT_ADK_IMG) -f python/Dockerfile ./python

.PHONY: build-app
build-app: buildx-create build-kagent-adk
	$(DOCKER_BUILDER) build $(DOCKER_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) --build-arg KAGENT_ADK_VERSION=$(KAGENT_ADK_IMAGE_TAG) --build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) -t $(APP_IMG) -f python/Dockerfile.app ./python

.PHONY: helm-cleanup
helm-cleanup:
	rm -f ./$(HELM_DIST_FOLDER)/*.tgz

.PHONY: helm-test
helm-test: helm-version
	mkdir -p tmp
	echo $$(helm template kagent ./helm/kagent/ --namespace kagent --set providers.default=ollama																	| tee tmp/ollama.yaml 		| grep ^kind: | wc -l)
	echo $$(helm template kagent ./helm/kagent/ --namespace kagent --set providers.default=openAI       --set providers.openAI.apiKey=your-openai-api-key 			| tee tmp/openAI.yaml 		| grep ^kind: | wc -l)
	echo $$(helm template kagent ./helm/kagent/ --namespace kagent --set providers.default=anthropic    --set providers.anthropic.apiKey=your-anthropic-api-key 	| tee tmp/anthropic.yaml 	| grep ^kind: | wc -l)
	echo $$(helm template kagent ./helm/kagent/ --namespace kagent --set providers.default=azureOpenAI  --set providers.azureOpenAI.apiKey=your-openai-api-key		| tee tmp/azureOpenAI.yaml	| grep ^kind: | wc -l)
	echo $$(helm template kagent ./helm/kagent/ --namespace kagent --set providers.default=gemini       --set providers.gemini.apiKey=your-gemini-api-key 			| tee tmp/gemini.yaml 		| grep ^kind: | wc -l)
	helm plugin ls | grep unittest || helm plugin install https://github.com/helm-unittest/helm-unittest.git
	helm unittest helm/kagent

.PHONY: helm-agents
helm-agents:
	VERSION=$(VERSION) envsubst < helm/agents/k8s/Chart-template.yaml > helm/agents/k8s/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/k8s
	VERSION=$(VERSION) envsubst < helm/agents/kgateway/Chart-template.yaml > helm/agents/kgateway/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/kgateway
	VERSION=$(VERSION) envsubst < helm/agents/istio/Chart-template.yaml > helm/agents/istio/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/istio
	VERSION=$(VERSION) envsubst < helm/agents/promql/Chart-template.yaml > helm/agents/promql/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/promql
	VERSION=$(VERSION) envsubst < helm/agents/observability/Chart-template.yaml > helm/agents/observability/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/observability
	VERSION=$(VERSION) envsubst < helm/agents/helm/Chart-template.yaml > helm/agents/helm/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/helm
	VERSION=$(VERSION) envsubst < helm/agents/argo-rollouts/Chart-template.yaml > helm/agents/argo-rollouts/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/argo-rollouts
	VERSION=$(VERSION) envsubst < helm/agents/cilium-policy/Chart-template.yaml > helm/agents/cilium-policy/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/cilium-policy
	VERSION=$(VERSION) envsubst < helm/agents/cilium-debug/Chart-template.yaml > helm/agents/cilium-debug/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/cilium-debug
	VERSION=$(VERSION) envsubst < helm/agents/cilium-manager/Chart-template.yaml > helm/agents/cilium-manager/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/agents/cilium-manager

.PHONY: helm-tools
helm-tools:
	VERSION=$(VERSION) envsubst < helm/tools/grafana-mcp/Chart-template.yaml > helm/tools/grafana-mcp/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/tools/grafana-mcp
	VERSION=$(VERSION) envsubst < helm/tools/querydoc/Chart-template.yaml > helm/tools/querydoc/Chart.yaml
	helm package -d $(HELM_DIST_FOLDER) helm/tools/querydoc

.PHONY: helm-version
helm-version: helm-cleanup helm-agents helm-tools
	VERSION=$(VERSION) KMCP_VERSION=$(KMCP_VERSION) envsubst < helm/kagent-crds/Chart-template.yaml > helm/kagent-crds/Chart.yaml
	VERSION=$(VERSION) KMCP_VERSION=$(KMCP_VERSION) envsubst < helm/kagent/Chart-template.yaml > helm/kagent/Chart.yaml
	helm dependency update helm/kagent
	helm dependency update helm/kagent-crds
	helm package -d $(HELM_DIST_FOLDER) helm/kagent-crds
	helm package -d $(HELM_DIST_FOLDER) helm/kagent

.PHONY: helm-install-provider
helm-install-provider: helm-version check-api-key
	helm $(HELM_ACTION) kagent-crds helm/kagent-crds \
		--namespace kagent \
		--create-namespace \
		--history-max 2    \
		--timeout 5m 			\
		--kube-context kind-$(KIND_CLUSTER_NAME) \
		--wait \
		--set kmcp.enabled=$(KMCP_ENABLED)
	helm $(HELM_ACTION) kagent helm/kagent \
		--namespace kagent \
		--create-namespace \
		--history-max 2    \
		--timeout 5m       \
		--kube-context kind-$(KIND_CLUSTER_NAME) \
		--wait \
		--set ui.service.type=LoadBalancer \
		--set registry=$(DOCKER_REGISTRY) \
		--set imagePullPolicy=Always \
		--set tag=$(VERSION) \
		--set controller.image.pullPolicy=Always \
		--set ui.image.pullPolicy=Always \
		--set controller.service.type=LoadBalancer \
		--set providers.openAI.apiKey=$(OPENAI_API_KEY) \
		--set providers.azureOpenAI.apiKey=$(AZUREOPENAI_API_KEY) \
		--set providers.anthropic.apiKey=$(ANTHROPIC_API_KEY) \
		--set providers.gemini.apiKey=$(GOOGLE_API_KEY) \
		--set providers.default=$(KAGENT_DEFAULT_MODEL_PROVIDER) \
		--set kmcp.enabled=$(KMCP_ENABLED) \
		--set kmcp.image.tag=$(KMCP_VERSION) \
		--set querydoc.openai.apiKey=$(OPENAI_API_KEY) \
		$(KAGENT_HELM_EXTRA_ARGS)

.PHONY: helm-install
helm-install: build
helm-install: helm-install-provider

.PHONY: helm-test-install
helm-test-install: HELM_ACTION+="--dry-run"
helm-test-install: helm-install-provider
# Test install with dry-run
# Example: `make helm-test-install | tee helm-test-install.log`

.PHONY: helm-uninstall
helm-uninstall:
	helm uninstall kagent --namespace kagent --kube-context kind-$(KIND_CLUSTER_NAME) --wait
	helm uninstall kagent-crds --namespace kagent --kube-context kind-$(KIND_CLUSTER_NAME) --wait

.PHONY: helm-publish
helm-publish: helm-version
	helm push ./$(HELM_DIST_FOLDER)/kagent-crds-$(VERSION).tgz $(HELM_REPO)/kagent/helm
	helm push ./$(HELM_DIST_FOLDER)/kagent-$(VERSION).tgz $(HELM_REPO)/kagent/helm
	helm push ./$(HELM_DIST_FOLDER)/helm-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/istio-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/promql-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/observability-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/argo-rollouts-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/cilium-policy-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/cilium-manager-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/cilium-debug-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents
	helm push ./$(HELM_DIST_FOLDER)/kgateway-agent-$(VERSION).tgz $(HELM_REPO)/kagent/agents

.PHONY: kagent-cli-install
kagent-cli-install: use-kind-cluster build-cli-local helm-version helm-install-provider
	KAGENT_HELM_REPO=./helm/ ./go/bin/kagent-local dashboard

.PHONY: kagent-cli-port-forward
kagent-cli-port-forward: use-kind-cluster
	@echo "Port forwarding to kagent CLI..."
	kubectl port-forward -n kagent service/kagent-controller 8083:8083

.PHONY: kagent-ui-port-forward
kagent-ui-port-forward: use-kind-cluster
	open http://localhost:8082/
	kubectl port-forward -n kagent service/kagent-ui 8082:8080

.PHONY: kagent-addon-install
kagent-addon-install: use-kind-cluster
	#to test the kagent addons - installing istio, grafana, prometheus, metrics-server
	istioctl install --set profile=demo -y
	kubectl apply -f contrib/addons/grafana.yaml
	kubectl apply -f contrib/addons/prometheus.yaml
	kubectl apply -f contrib/addons/metrics-server.yaml
	#wait for pods to be ready
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=grafana 	-n kagent --timeout=60s
	kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=prometheus -n kagent --timeout=60s
	#port forward grafana service
	kubectl port-forward svc/grafana 3000:3000 -n kagent

.PHONY: open-dev-container
open-dev-container:
	@echo "Opening dev container..."
	devcontainer build .
	@devcontainer open .

.PHONY: otel-local
otel-local:
	docker rm -f jaeger-desktop || true
	docker run -d --name jaeger-desktop --restart=always -p 16686:16686 -p 4317:4317 -p 4318:4318 jaegertracing/jaeger:2.7.0
	open http://localhost:16686/

.PHONY: kind-debug
kind-debug:
	@echo "Debugging the kind cluster..."
	@echo "Enter the kind cluster control plane container..."
	docker exec -it $(KIND_CLUSTER_NAME)-control-plane bash -c 'apt-get update && apt-get install -y btop htop'
	docker exec -it $(KIND_CLUSTER_NAME)-control-plane bash -c 'btop --utf-force'

.PHONY: audit
audit:
	echo "Running CVE audit GO"
	make -C go govulncheck
	echo "Running CVE audit UI"
	make -C ui audit
	echo "Running CVE audit PYTHON"
	make -C python audit

.PHONY: report/image-cve
report/image-cve: audit build
	echo "Running CVE scan :: CVE -> CSV ... reports/$(SEMVER)/"
	grype docker:$(CONTROLLER_IMG) -o template -t reports/cve-report.tmpl --file reports/$(SEMVER)/controller-cve.csv
	grype docker:$(APP_IMG)        -o template -t reports/cve-report.tmpl --file reports/$(SEMVER)/app-cve.csv
	grype docker:$(UI_IMG)         -o template -t reports/cve-report.tmpl --file reports/$(SEMVER)/ui-cve.csv

##############################################################################
# Azure Container Registry (ACR) and Azure Kubernetes Service (AKS) Targets
##############################################################################

# Check if the appropriate API key is set for AKS deployment
check-aks-api-key:
	@if [ "$(AKS_DEFAULT_MODEL_PROVIDER)" = "openAI" ]; then \
		if [ -z "$(OPENAI_API_KEY)" ]; then \
			echo "Error: OPENAI_API_KEY environment variable is not set for OpenAI provider"; \
			echo "Please set it with: export OPENAI_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(AKS_DEFAULT_MODEL_PROVIDER)" = "anthropic" ]; then \
		if [ -z "$(ANTHROPIC_API_KEY)" ]; then \
			echo "Error: ANTHROPIC_API_KEY environment variable is not set for Anthropic provider"; \
			echo "Please set it with: export ANTHROPIC_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(AKS_DEFAULT_MODEL_PROVIDER)" = "azureOpenAI" ]; then \
		if [ -z "$(AZUREOPENAI_API_KEY)" ]; then \
			echo "Error: AZUREOPENAI_API_KEY environment variable is not set for Azure OpenAI provider"; \
			echo "Please set it with: export AZUREOPENAI_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
		if [ -z "$(AZUREOPENAI_ENDPOINT)" ]; then \
			echo "Error: AZUREOPENAI_ENDPOINT environment variable is not set for Azure OpenAI provider"; \
			echo "Please set it with: export AZUREOPENAI_ENDPOINT=https://your-resource.openai.azure.com"; \
			exit 1; \
		fi; \
	elif [ "$(AKS_DEFAULT_MODEL_PROVIDER)" = "gemini" ]; then \
		if [ -z "$(GOOGLE_API_KEY)" ]; then \
			echo "Error: GOOGLE_API_KEY environment variable is not set for Gemini provider"; \
			echo "Please set it with: export GOOGLE_API_KEY=your-api-key"; \
			exit 1; \
		fi; \
	elif [ "$(AKS_DEFAULT_MODEL_PROVIDER)" = "ollama" ]; then \
		echo "Note: Ollama provider does not require an API key"; \
	else \
		echo "Warning: Unknown model provider '$(AKS_DEFAULT_MODEL_PROVIDER)'. Skipping API key check."; \
	fi

.PHONY: acr-login
acr-login:
	@echo "Checking Azure CLI availability..."
	@which az > /dev/null || (echo "Error: Azure CLI (az) is not installed. Please install it from https://docs.microsoft.com/en-us/cli/azure/install-azure-cli" && exit 1)
	@echo "Authenticating to Azure Container Registry: $(ACR_REGISTRY)..."
	@ACR_NAME=$$(echo $(ACR_REGISTRY) | cut -d. -f1); \
	az acr login --name $$ACR_NAME || (echo "Error: Failed to authenticate to ACR. Please run 'az login' first." && exit 1)
	@echo "Successfully authenticated to $(ACR_REGISTRY)"

.PHONY: build-acr
build-acr: buildx-create acr-login controller-manifests
	@echo "Building and pushing images to Azure Container Registry: $(ACR_REGISTRY)"
	$(DOCKER_BUILDER) build $(ACR_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(ACR_REGISTRY)/$(ACR_REPO)/$(CONTROLLER_IMAGE_NAME):$(CONTROLLER_IMAGE_TAG) -f go/Dockerfile ./go
	$(DOCKER_BUILDER) build $(ACR_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(ACR_REGISTRY)/$(ACR_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG) -f ui/Dockerfile ./ui
	$(DOCKER_BUILDER) build $(ACR_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(ACR_REGISTRY)/$(ACR_REPO)/$(KAGENT_ADK_IMAGE_NAME):$(KAGENT_ADK_IMAGE_TAG) -f python/Dockerfile ./python
	$(DOCKER_BUILDER) build $(ACR_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) --build-arg KAGENT_ADK_VERSION=$(KAGENT_ADK_IMAGE_TAG) --build-arg DOCKER_REGISTRY=$(ACR_REGISTRY) -t $(ACR_REGISTRY)/$(ACR_REPO)/$(APP_IMAGE_NAME):$(APP_IMAGE_TAG) -f python/Dockerfile.app ./python
	@echo "Build completed successfully."
	@echo "Controller Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(CONTROLLER_IMAGE_NAME):$(CONTROLLER_IMAGE_TAG)"
	@echo "UI Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG)"
	@echo "App Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(APP_IMAGE_NAME):$(APP_IMAGE_TAG)"
	@echo "Kagent ADK Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(KAGENT_ADK_IMAGE_NAME):$(KAGENT_ADK_IMAGE_TAG)"

.PHONY: build-ui-acr
build-ui-acr: buildx-create acr-login
	@echo "Building and pushing UI image to Azure Container Registry: $(ACR_REGISTRY)"
	$(DOCKER_BUILDER) build $(ACR_BUILD_ARGS) $(TOOLS_IMAGE_BUILD_ARGS) -t $(ACR_REGISTRY)/$(ACR_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG) -f ui/Dockerfile ./ui
	@echo "UI build completed successfully."
	@echo "UI Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG)"

.PHONY: aks-check-context
aks-check-context:
	@echo "Checking current kubectl context..."
	@CURRENT_CONTEXT=$$(kubectl config current-context); \
	echo "Current context: $$CURRENT_CONTEXT"; \
	if echo "$$CURRENT_CONTEXT" | grep -q "^kind-"; then \
		echo "Warning: Current context appears to be a Kind cluster ($$CURRENT_CONTEXT)."; \
		echo "Are you sure you want to deploy to this cluster? If not, switch context first:"; \
		echo "  kubectl config use-context <your-aks-context>"; \
		exit 1; \
	fi

.PHONY: aks-create-acr-secret
aks-create-acr-secret:
	@echo "Creating ACR image pull secret..."
	@if [ -z "$(ACR_USERNAME)" ] || [ -z "$(ACR_PASSWORD)" ]; then \
		echo "Error: ACR_USERNAME and ACR_PASSWORD environment variables must be set"; \
		echo "Run: export ACR_USERNAME=<username> ACR_PASSWORD=<password>"; \
		exit 1; \
	fi
	@kubectl create namespace $(AKS_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@kubectl create secret docker-registry acr-secret \
		--namespace $(AKS_NAMESPACE) \
		--docker-server=$(ACR_REGISTRY) \
		--docker-username=$(ACR_USERNAME) \
		--docker-password=$(ACR_PASSWORD) \
		--dry-run=client -o yaml | kubectl apply -f -
	@echo "ACR secret created successfully"

.PHONY: helm-install-aks
helm-install-aks: helm-version check-aks-api-key aks-check-context aks-create-acr-secret
	@echo "Installing kagent to AKS cluster using ACR images..."
	helm $(HELM_ACTION) kagent-crds helm/kagent-crds \
		--namespace $(AKS_NAMESPACE) \
		--create-namespace \
		--history-max 2    \
		--timeout 5m 			\
		--wait \
		--set kmcp.enabled=$(KMCP_ENABLED)
	helm $(HELM_ACTION) kagent helm/kagent \
		--namespace $(AKS_NAMESPACE) \
		--create-namespace \
		--history-max 2    \
		--timeout 5m       \
		--wait \
		--set ui.service.type=$(AKS_SERVICE_TYPE) \
		--set registry=$(ACR_REGISTRY) \
		--set imagePullPolicy=IfNotPresent \
		--set tag=$(VERSION) \
		--set imagePullSecrets[0].name=acr-secret \
		--set controller.agentImage.pullSecret=acr-secret \
		--set controller.image.pullPolicy=IfNotPresent \
		--set ui.image.pullPolicy=IfNotPresent \
		--set controller.service.type=$(AKS_SERVICE_TYPE) \
		--set providers.openAI.apiKey=$(OPENAI_API_KEY) \
		--set providers.azureOpenAI.apiKey=$(AZUREOPENAI_API_KEY) \
		--set providers.azureOpenAI.config.apiVersion=$(AZUREOPENAI_API_VERSION) \
		--set providers.azureOpenAI.config.azureEndpoint=$(AZUREOPENAI_ENDPOINT) \
		--set providers.azureOpenAI.config.azureDeployment=$(AZUREOPENAI_DEPLOYMENT) \
		--set providers.azureOpenAI.model=$(AZUREOPENAI_MODEL) \
		--set providers.anthropic.apiKey=$(ANTHROPIC_API_KEY) \
		--set providers.gemini.apiKey=$(GOOGLE_API_KEY) \
		--set providers.default=$(AKS_DEFAULT_MODEL_PROVIDER) \
		--set kmcp.enabled=$(KMCP_ENABLED) \
		--set kmcp.image.tag=$(KMCP_VERSION) \
		--set querydoc.openai.apiKey=$(OPENAI_API_KEY) \
		$(KAGENT_HELM_EXTRA_ARGS)
	@echo ""
	@echo "Kagent successfully installed to AKS!"
	@echo "To access the UI, run: make aks-port-forward-ui"
	@echo "To access the CLI, run: make aks-port-forward-cli"

.PHONY: helm-uninstall-aks
helm-uninstall-aks:
	@echo "Uninstalling kagent from AKS cluster..."
	helm uninstall kagent --namespace $(AKS_NAMESPACE) --wait || true
	helm uninstall kagent-crds --namespace $(AKS_NAMESPACE) --wait || true
	@echo "Kagent uninstalled from AKS."

.PHONY: aks-port-forward-ui
aks-port-forward-ui:
	@echo "Port forwarding kagent UI to http://localhost:8082/"
	@echo "Press Ctrl+C to stop port forwarding."
	kubectl port-forward -n $(AKS_NAMESPACE) service/kagent-ui 8082:8080

.PHONY: aks-port-forward-cli
aks-port-forward-cli:
	@echo "Port forwarding kagent CLI to localhost:8083"
	@echo "Press Ctrl+C to stop port forwarding."
	kubectl port-forward -n $(AKS_NAMESPACE) service/kagent-controller 8083:8083

.PHONY: aks-update-ui
aks-update-ui:
	@echo "Redeploying kagent UI on AKS..."
	@echo "Setting imagePullPolicy to Always to force image pull..."
	kubectl patch deployment kagent-ui -n $(AKS_NAMESPACE) -p '{"spec":{"template":{"spec":{"containers":[{"name":"ui","imagePullPolicy":"Always"}]}}}}'
	@echo "Triggering rollout restart..."
	kubectl rollout restart deployment/kagent-ui -n $(AKS_NAMESPACE)
	@echo "Waiting for rollout to complete..."
	kubectl rollout status deployment/kagent-ui -n $(AKS_NAMESPACE)
	@echo ""
	@echo "UI successfully redeployed!"
	@echo "Access UI: make aks-port-forward-ui"

.PHONY: aks-update-ui-all
aks-update-ui-all: build-ui-acr aks-update-ui
	@echo ""
	@echo "=========================================="
	@echo "UI update to AKS complete!"
	@echo "=========================================="
	@echo ""
	@echo "UI Image: $(ACR_REGISTRY)/$(ACR_REPO)/$(UI_IMAGE_NAME):$(UI_IMAGE_TAG)"
	@echo "Namespace: $(AKS_NAMESPACE)"
	@echo ""
	@echo "Access UI: make aks-port-forward-ui"
	@echo ""

.PHONY: aks-deploy-all
aks-deploy-all: build-acr helm-install-aks
	@echo ""
	@echo "=========================================="
	@echo "Kagent deployment to AKS complete!"
	@echo "=========================================="
	@echo ""
	@echo "Images pushed to: $(ACR_REGISTRY)/$(ACR_REPO)"
	@echo "Deployed to namespace: $(AKS_NAMESPACE)"
	@echo ""
	@echo "Next steps:"
	@echo "  - Access UI: make aks-port-forward-ui"
	@echo "  - Access CLI: make aks-port-forward-cli"
	@echo ""

.PHONY: aks-deploy-only
aks-deploy-only: helm-install-aks
	@echo ""
	@echo "=========================================="
	@echo "Deployed using pre-built images"
	@echo "=========================================="
	@echo ""
	@echo "Registry: $(ACR_REGISTRY)/$(ACR_REPO)"
	@echo "Version: $(VERSION)"
	@echo "Namespace: $(AKS_NAMESPACE)"
	@echo ""
	@echo "Next steps:"
	@echo "  - Access UI: make aks-port-forward-ui"
	@echo "  - Access CLI: make aks-port-forward-cli"
	@echo ""
