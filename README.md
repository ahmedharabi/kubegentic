<div align="center">

<img src="../docs/logo.png" alt="Kubegentic Logo" width="200">

# Kubegentic

**Kubernetes-native AI Agent Runtime Platform**

Treat AI agents as first-class Kubernetes workloads -- declarative, self-healing, observable, and scalable.

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

Kubegentic lets you define and manage AI agents as Kubernetes custom resources. The operator watches Agent CRDs and reconciles the desired state, creating Deployments, Services, RBAC, and HPA automatically.

> **Status:** Early development. The Kubernetes operator with Agent CRD reconciliation is implemented. Python agent runtime, tool servers, and Helm chart are on the roadmap.

---

## Features

- **Declarative YAML** : Define agents, tools, and memory in standard Kubernetes manifests
- **Operator Pattern** : Self-healing reconciliation loop built with Kubebuilder
- **Pluggable Interfaces** : ATI (tools), AMI (memory), ALI (LLMs), AWI (agents)
- **Autoscaling** : Event-driven scaling via KEDA and HPA
- **Observable** : OpenTelemetry traces, Prometheus metrics, structured logging

---

## Why Kubegentic?

Running AI agents in production today requires wiring together infrastructure, scaling, and monitoring manually. Kubegentic brings AI agents into the Kubernetes ecosystem so they behave like any other workload:

1. Declare your agent in YAML
2. The operator handles provisioning, networking, and RBAC
3. Platform tooling (Prometheus, Grafana, KEDA) works out of the box

---

## How It Works

Define an agent in a manifest:

```yaml
apiVersion: agent.kubegentic.dev/v1
kind: Agent
metadata:
  name: devops-agent
  namespace: default
spec:
  model: deepseek-v4-flash
  provider: deepseek
  systemPrompt: |
    You are a senior DevOps engineer. Be concise and security-focused.
  apiKeySecretRef:
    name: deepseek-credentials
    key: api-key
```

Apply it and watch it reconcile:

```bash
kubectl apply -f agent.yaml
kubectl get agents
```

The operator creates the underlying Kubernetes resources (Deployments, Services, RBAC) and keeps them in sync with the declared state.

---

## Quickstart

### Prerequisites

- Go 1.21+
- Docker
- [Kind](https://kind.sigs.k8s.io/)
- kubectl

### Run it locally

```bash
# Clone and build
git clone https://github.com/ahmedharabi/kubegentic.git
cd kubegentic

make build

# Bootstrap a cluster and deploy
kind create cluster --name kubegentic
make install
make docker-build IMG=kubegentic/operator:latest
kind load docker-image kubegentic/operator:latest --name kubegentic
make deploy IMG=kubegentic/operator:latest

# Apply a sample agent
kubectl apply -f config/samples/agent_v1_agent.yaml
kubectl get agents
```

---

## Repository Structure

```
kubegentic/
├── api/v1/                  # CRD type definitions
├── cmd/main.go              # Controller manager entry point
├── config/                  # Kustomize manifests (CRD, RBAC, manager deployment)
├── internal/controller/     # Reconciliation loop logic
├── test/                    # End-to-end tests
├── Dockerfile               # Container image
├── Makefile                 # Build, test, deploy targets
└── go.mod                   # Go module
```

---

## Contributing

Contributions are welcome.

```bash
git clone https://github.com/ahmedharabi/kubegentic.git
cd kubegentic
git checkout -b feat/your-feature

# Make your changes, then:
make test
make build
```

Before starting significant work, please open an issue to discuss your approach. This helps avoid duplicated effort and ensures alignment with the project direction.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for detailed guidelines.

---

## License

Apache 2.0 -- see [LICENSE](./LICENSE).
