# Release Manager: Automated Release Workflow with Kagent

> **Goal**: Build a self-managing release agent that automates the entire release process

**What you'll build**: A release manager that creates branches, runs tests, builds images, manages Jira tickets, and generates release notes—all automatically.

**Time to read**: 5 minutes

---

## The Workflow

```
Developer: "Start release v1.5.0"

release-manager (Coordinator)
    │
    ├─ Creates Jira Epic + Tasks
    ├─ Delegates to github-agent → create branch
    ├─ Delegates to github-actions-agent → run tests
    ├─ Delegates to github-actions-agent → build images
    ├─ Creates release notes in Confluence
    └─ Delegates to github-agent → tag & publish

If tests fail → Stops release, updates Jira, notifies you
```

**What it does:**
1. Creates Jira epic "Release v1.5.0" with tasks
2. Creates release branch from main
3. Runs tests via GitHub Actions (waits for results)
4. Builds and pushes images to Artifactory
5. Generates release notes in Confluence
6. Tags release and creates GitHub Release
7. Updates all Jira tasks as it progresses

---

## The Manifests

Save this as `release-manager.yaml`:

```yaml
---
# 1. GitHub MCP (for git operations)
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: github-mcp
  namespace: kagent
spec:
  url: "https://api.githubcopilot.com/mcp/repositories"
  protocol: STREAMABLE_HTTP
  timeout: 30s
  headersFrom:
    - name: Authorization
      valueFrom:
        type: Secret
        name: github-token
        key: token
  description: "GitHub repository operations"

---
# 2. GitHub Actions MCP (for CI/CD)
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: github-actions-mcp
  namespace: kagent
spec:
  url: "https://api.githubcopilot.com/mcp/actions"
  protocol: STREAMABLE_HTTP
  timeout: 300s  # Longer timeout for builds
  headersFrom:
    - name: Authorization
      valueFrom:
        type: Secret
        name: github-token
        key: token
  description: "GitHub Actions workflow operations"

---
# 3. Jira MCP (for issue tracking)
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: jira-mcp
  namespace: kagent
spec:
  url: "http://jira-mcp-server.default:8080/mcp"
  protocol: STREAMABLE_HTTP
  timeout: 30s
  headersFrom:
    - name: Authorization
      valueFrom:
        type: Secret
        name: jira-token
        key: token
  description: "Jira issue management"

---
# 4. Confluence MCP (for documentation)
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: confluence-mcp
  namespace: kagent
spec:
  url: "http://confluence-mcp-server.default:8080/mcp"
  protocol: STREAMABLE_HTTP
  timeout: 30s
  headersFrom:
    - name: Authorization
      valueFrom:
        type: Secret
        name: confluence-token
        key: token
  description: "Confluence documentation"

---
# 5. GitHub Specialist Agent
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: github-agent
  namespace: kagent
spec:
  type: Declarative
  description: "GitHub specialist - handles branches, tags, and releases"
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      You are a GitHub specialist. You handle:
      - Creating branches from main
      - Listing PRs and commits between versions
      - Creating annotated tags
      - Publishing GitHub releases

      You execute git operations cleanly and report results with:
      - Branch/tag names
      - Commit SHAs
      - URLs to created resources
    tools:
      - type: McpServer
        mcpServer:
          kind: RemoteMCPServer
          name: github-mcp

---
# 6. GitHub Actions Specialist Agent
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: github-actions-agent
  namespace: kagent
spec:
  type: Declarative
  description: "CI/CD specialist - triggers and monitors workflows"
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      You are a CI/CD specialist. You:
      - Trigger GitHub Actions workflows by name and branch
      - Monitor workflow execution (poll for status)
      - Wait patiently for completion (builds can take 10+ minutes)
      - Report final status clearly: SUCCESS or FAILURE

      If a workflow fails, provide:
      - Error message
      - Failed step name
      - Workflow run URL
    tools:
      - type: McpServer
        mcpServer:
          kind: RemoteMCPServer
          name: github-actions-mcp

---
# 7. Release Manager (Orchestrator)
apiVersion: kagent.dev/v1alpha2
kind: Agent
metadata:
  name: release-manager
  namespace: kagent
spec:
  type: Declarative
  description: "Release manager - orchestrates the entire release process"
  declarative:
    modelConfig: default-model-config
    systemMessage: |
      You are a release manager. You orchestrate software releases following this process:

      ## Release Steps (execute in order):

      1. CREATE JIRA EPIC AND TASKS
         - Create epic: "Release {version}"
         - Create tasks under epic:
           * "Create release branch"
           * "Run test suite"
           * "Build and push images"
           * "Generate release notes"
           * "Tag and publish release"
         - Assign all to bot user
         - Set epic status: "In Progress"

      2. CREATE RELEASE BRANCH
         - Delegate to github-agent: "Create branch release/{version} from main"
         - Update task status: "Done"

      3. RUN TESTS
         - Update task status: "In Progress"
         - Delegate to github-actions-agent: "Run workflow 'test.yml' on release/{version}"
         - Wait for results
         - IF FAILED:
           * Update task: "Blocked" with error details
           * Update epic: "Blocked"
           * STOP and report failure
         - IF PASSED:
           * Update task: "Done"

      4. BUILD AND PUSH IMAGES
         - Update task status: "In Progress"
         - Delegate to github-actions-agent: "Run workflow 'build.yml' on release/{version}"
         - Wait for completion
         - Update task: "Done" with image URLs

      5. GENERATE RELEASE NOTES
         - Update task status: "In Progress"
         - Delegate to github-agent: "List PRs merged between last release and release/{version}"
         - Extract Jira ticket IDs from PR titles (e.g., [PROJ-123])
         - Fetch ticket details from Jira
         - Group by type: Features, Bugs, Improvements
         - Create Confluence page: "Release {version} Notes"
         - Update task: "Done" with Confluence URL

      6. TAG AND PUBLISH
         - Update task status: "In Progress"
         - Delegate to github-agent: "Create tag {version} on release/{version}"
         - Delegate to github-agent: "Create GitHub Release with notes"
         - Update task: "Done"
         - Update epic: "Done"

      7. SUMMARIZE
         - Report success with links to:
           * Jira epic
           * Release branch
           * GitHub release
           * Artifactory images
           * Confluence notes

      ## Important Rules:
      - Execute steps sequentially (don't skip ahead)
      - Update Jira tasks after each step
      - Stop immediately if any step fails
      - Provide clear status updates
      - You are methodical and thorough

    tools:
      # Direct tools for Jira and Confluence
      - type: McpServer
        mcpServer:
          kind: RemoteMCPServer
          name: jira-mcp
      - type: McpServer
        mcpServer:
          kind: RemoteMCPServer
          name: confluence-mcp

      # Delegate technical work to specialists
      - type: Agent
        agent:
          name: github-agent
      - type: Agent
        agent:
          name: github-actions-agent
```

---

## Understanding the Structure

### 1️⃣ **MCP Servers** (The Tools)
```yaml
kind: RemoteMCPServer
spec:
  url: "https://api.githubcopilot.com/mcp/repositories"
  headersFrom:
    - name: Authorization
      valueFrom:
        type: Secret
        name: github-token
```
- Connect to external services (GitHub, Jira, Confluence)
- Use secrets for authentication
- Provide tools to agents

### 2️⃣ **Specialist Agents**
```yaml
# github-agent: Git operations only
# github-actions-agent: CI/CD operations only
```
- Each focuses on ONE domain
- Reusable by other workflows
- Clear responsibility

### 3️⃣ **Release Manager** (The Orchestrator)
```yaml
tools:
  - type: McpServer
    mcpServer:
      name: jira-mcp          # Direct: Owns release state
  - type: Agent
    agent:
      name: github-agent      # Delegates: Technical work
```
- Uses MCPs directly for Jira/Confluence (core release tracking)
- Delegates to specialists for git/CI operations
- Follows strict sequential workflow

---

## Security

### How Communication Works

**Agent-to-Agent Communication**:
- Uses **A2A (Agent-to-Agent) protocol** over HTTP
- Traffic stays within Kubernetes cluster (private network)
- Agents discover each other via Kubernetes DNS: `http://agent-name.namespace:8080`
- Each agent has its own ServiceAccount with unique identity token
- Network isolation by namespace (agents can only call agents in same namespace)

**Agent-to-MCP Communication**:
- Uses **MCP (Model Context Protocol)** over HTTP/HTTPS
- Credentials passed via headers from Kubernetes Secrets (encrypted at rest)
```yaml
headersFrom:
  - name: Authorization
    valueFrom:
      type: Secret      # Stored encrypted in etcd
      name: github-token
      key: token
```
- External MCPs (GitHub, Jira): HTTPS with token authentication
- Internal MCPs: Private cluster network (ClusterIP services)

### Built-in Security

✅ **Network Isolation**: All services use ClusterIP (not exposed outside cluster)

✅ **Secret Management**: Tokens mounted securely to pods, never exposed in manifests

✅ **Service Accounts**: Each agent gets unique identity for Kubernetes RBAC

✅ **Namespace Boundaries**: Agents cannot access agents in other namespaces

---

## Prerequisites

Before deploying, create secrets:

```bash
# GitHub token (needs: repo, workflow permissions)
kubectl create secret generic github-token \
  --from-literal=token="ghp_your_github_token" \
  -n kagent

# Jira token (Basic auth: email:api_token base64 encoded)
kubectl create secret generic jira-token \
  --from-literal=token="Basic $(echo -n 'email@example.com:jira_api_token' | base64)" \
  -n kagent

# Confluence token (same as Jira if using same Atlassian account)
kubectl create secret generic confluence-token \
  --from-literal=token="Basic $(echo -n 'email@example.com:confluence_token' | base64)" \
  -n kagent
```

**Note**: You'll need to deploy Jira and Confluence MCP servers separately (see below).

---

## Deploy

```bash
# Apply the release manager workflow
kubectl apply -f release-manager.yaml

# Verify all components
kubectl get agents,remotemcpservers -n kagent | grep -E '(release-manager|github-agent|github-actions|jira|confluence)'

# Check agent status
kubectl get agents release-manager -n kagent -o wide
```

---

## Deploy Jira & Confluence MCPs

You'll need MCP servers for Jira and Confluence. Here's a simple approach using docker containers:

```yaml
# jira-mcp-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jira-mcp-server
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jira-mcp
  template:
    metadata:
      labels:
        app: jira-mcp
    spec:
      containers:
      - name: jira-mcp
        image: your-jira-mcp-server:latest  # Deploy your Jira MCP server
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: jira-mcp-server
  namespace: default
spec:
  selector:
    app: jira-mcp
  ports:
  - port: 8080
    targetPort: 8080
```

Apply the same pattern for `confluence-mcp-server`.

---

## Test It

Access the Kagent UI:

```bash
kubectl port-forward svc/kagent-ui 8080:8080 -n kagent
# Open http://localhost:8080
```

**Chat with `release-manager`:**

**Prompt**: *"Start release v1.5.0"*

**Expected behavior:**
1. Creates Jira epic and 5 tasks
2. Creates branch `release/v1.5.0`
3. Triggers test workflow and waits
4. If tests pass: proceeds to build
5. Builds images, creates notes, tags release
6. Updates all Jira tasks to "Done"
7. Provides summary with all links

**If tests fail:**
- Marks test task as "Blocked"
- Marks epic as "Blocked"
- Stops the release
- Reports error details

---

## What Happens Under the Hood

### Step-by-Step Execution

```
1. release-manager receives: "Start release v1.5.0"
   ├─ Uses jira-mcp: Creates epic + 5 tasks
   └─ Updates epic status: "In Progress"

2. release-manager delegates: "Create branch"
   ├─ Calls github-agent
   ├─ github-agent uses github-mcp: Creates branch
   └─ Updates Jira task: "Done"

3. release-manager delegates: "Run tests"
   ├─ Calls github-actions-agent
   ├─ github-actions-agent uses github-actions-mcp:
   │   ├─ Triggers workflow
   │   ├─ Polls for status every 30s
   │   └─ Returns: "Tests passed ✅"
   └─ Updates Jira task: "Done"

4. (continues for build, notes, tag...)

5. release-manager summarizes:
   "Release v1.5.0 complete! 🎉
    ├─ Epic: https://jira.company.com/browse/REL-123
    ├─ Branch: release/v1.5.0
    ├─ Images: https://artifactory.company.com/v1.5.0
    ├─ Notes: https://confluence.company.com/release-v1.5.0
    └─ GitHub: https://github.com/org/repo/releases/v1.5.0"
```

---

## Workflow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ Developer                                                    │
│ "Start release v1.5.0"                                      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ release-manager (Orchestrator)                              │
│                                                              │
│ Creates Epic in Jira ─────────────────► jira-mcp            │
│ Creates 5 tasks      ─────────────────► jira-mcp            │
│                                                              │
│ "Create branch" ──────────────────────► github-agent        │
│                                           └─► github-mcp     │
│ Updates task ─────────────────────────► jira-mcp            │
│                                                              │
│ "Run tests" ──────────────────────────► github-actions-agent│
│                                           └─► github-actions-mcp
│ Waits for result...                                         │
│ Updates task ─────────────────────────► jira-mcp            │
│                                                              │
│ "Build images" ───────────────────────► github-actions-agent│
│ Updates task ─────────────────────────► jira-mcp            │
│                                                              │
│ "Create notes" ───────────────────────► confluence-mcp      │
│ Updates task ─────────────────────────► jira-mcp            │
│                                                              │
│ "Tag release" ────────────────────────► github-agent        │
│ Updates epic to Done ──────────────────► jira-mcp           │
│                                                              │
│ Summarizes ────────────────────────────► Developer          │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Benefits

✅ **Automated Release Process**: No manual steps, fully orchestrated

✅ **Failure Handling**: Stops on test failures, updates Jira automatically

✅ **Audit Trail**: All actions tracked in Jira tasks

✅ **Reusable Specialists**: github-agent and github-actions-agent can be used by other workflows

✅ **Sequential Safety**: Steps run in order, no skipping ahead

✅ **Clear Status**: Always know where the release is via Jira epic

---

## Extend This Example

### Add Slack Notifications

```yaml
# Add slack-mcp
apiVersion: kagent.dev/v1alpha2
kind: RemoteMCPServer
metadata:
  name: slack-mcp
spec:
  url: "http://slack-mcp-server:8080/mcp"
```

Update release-manager tools:
```yaml
tools:
  - type: McpServer
    mcpServer:
      name: slack-mcp  # ← Add this
```

### Add Approval Step

Modify release-manager system message:
```yaml
systemMessage: |
  ...
  3.5. WAIT FOR APPROVAL
     - After tests pass, create Jira approval task
     - Assign to release lead
     - Wait for task status: "Approved"
     - If rejected, stop release
  ...
```

### Support Multiple Environments

```yaml
# Deploy to staging first
4.5. DEPLOY TO STAGING
   - Delegate to k8s-agent: "Deploy v1.5.0 to staging"
   - Wait for health checks
   - Update task: "Done"
```

---

## Troubleshooting

### Tests keep failing
- Check GitHub Actions workflow logs
- Ensure release branch is created correctly
- Verify test dependencies are available

### Jira tasks not updating
- Check jira-mcp server logs
- Verify Jira token has correct permissions (create issues, edit issues)
- Ensure bot user exists in Jira

### Agent times out waiting for builds
- Increase timeout in github-actions-mcp spec (default: 300s)
- Check if workflow is actually running in GitHub

---

## Clean Up

```bash
kubectl delete -f release-manager.yaml
kubectl delete secret github-token jira-token confluence-token -n kagent
```

---

## What's Next?

- **Add more gates**: Security scans, performance tests
- **Multi-repo releases**: Coordinate releases across microservices
- **Rollback capability**: Automated rollback on production issues
- **Release calendar**: Scheduled releases via CronJobs

---

**Remember**: The release-manager owns the process and state (Jira), but delegates execution to specialists. This keeps it modular and maintainable! 🚀
