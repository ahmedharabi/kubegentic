<div align="center">

# Kubegentic

**Kubernetes-native AI Agent Runtime Platform**

Treat AI agents as first-class Kubernetes workloads &mdash; declarative, self-healing, observable, and scalable.

```
kubectl apply -f agent.yaml
```

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://golang.org/)

</div>

Kubegentic lets you define and manage AI agents as Kubernetes custom resources. The operator watches Agent CRDs and reconciles the desired state &mdash; creating Deployments, Services, RBAC, and HPA automatically.

> **Status:** Early development. The Kubernetes operator with Agent CRD reconciliation is implemented. The Python agent runtime, tool servers, and Helm chart are planned.

---

## Features

- **Declarative YAML** &mdash; Define agents, tools, and memory in Kubernetes manifests
- **Operator pattern** &mdash; Self-healing reconciliation loop via Kubebuilder
- **Pluggable interfaces** &mdash; ATI (tools), AMI (memory), ALI (LLMs), AWI (agents)
- **Autoscaling** &mdash; Event-driven scaling with KEDA and HPA
- **Observable** &mdash; OpenTelemetry traces, Prometheus metrics, structured logs

---

## How It Works

```yaml
apiVersion: agent.kubegentic.dev/v1
kind: Agent
metadata:
  name: devops-agent
spec:
  model: llama3.2
  provider: ollama
  systemPrompt: |
    You are a senior DevOps engineer.
  tools:
    - name: kubectl
      toolRef: kubectl-tool
  autoscaling:
    minReplicas: 1
    maxReplicas: 5
```

```bash
kubectl apply -f agent.yaml
kubectl get agents
```

---

## Quickstart

**Prerequisites:** Go 1.21+, Docker, [Kind](https://kind.sigs.k8s.io/), kubectl

```bash
git clone https://github.com/ahmedharabi/kubegentic.git
cd kubegentic

make build
kind create cluster --name kubegentic
make install
make docker-build IMG=kubegentic/operator:latest
kind load docker-image kubegentic/operator:latest --name kubegentic
make deploy IMG=kubegentic/operator:latest

kubectl apply -f config/samples/agent_v1_agent.yaml
kubectl get agents
```

---

## Repository Structure

```
kubegentic/
├── api/v1/                  # CRD type definitions
├── cmd/main.go              # Controller manager entry point
├── config/                  # Kustomize manifests (CRD, RBAC, manager)
├── internal/controller/     # Reconciliation logic
├── test/                    # End-to-end tests
├── Dockerfile
├── Makefile
└── go.mod
```

---

## Contributing

```bash
git clone https://github.com/ahmedharabi/kubegentic.git
cd kubegentic
git checkout -b feat/your-feature
make test
make build
```

Please open an issue before starting significant work.

---

## License

Apache 2.0 &mdash; see [LICENSE](./LICENSE).
