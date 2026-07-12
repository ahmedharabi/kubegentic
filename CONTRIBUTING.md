# Contributing

Thanks for taking the time to contribute.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch

```bash
git clone https://github.com/YOUR_USERNAME/kubegentic.git
cd kubegentic
git checkout -b feat/your-feature
```

## Development

### Prerequisites

- Go 1.21+
- Docker
- kubectl
- [Kind](https://kind.sigs.k8s.io/) (for local testing)

### Make targets

| Target       | Description                            |
| ------------ | -------------------------------------- |
| `make build` | Build the operator binary              |
| `make test`  | Run unit tests                         |
| `make lint`  | Run linter (golangci-lint)             |
| `make fmt`   | Format Go source code                  |
| `make vet`   | Run `go vet`                           |
| `make docker-build` | Build the Docker image          |
| `make deploy` | Deploy operator to the current cluster |
| `make install` | Install CRDs into the cluster        |

### Run tests

```bash
make test
make lint
```

## Pull Request Guidelines

- Open an issue first for significant changes so we can discuss the approach
- Keep PRs focused — one feature or fix per PR
- Write clear commit messages
- Update or add tests where applicable
- Make sure `make test` and `make lint` pass

## Code Style

This project follows [Go style guide](https://go.dev/doc/effective_go) conventions. Run `make fmt` before committing.

## Project Structure

```
api/v1/                  # CRD type definitions
internal/controller/     # Reconciliation logic
config/                  # Kustomize manifests
test/                    # E2E tests
```

## Questions?

Open a [discussion](https://github.com/ahmedharabi/kubegentic/discussions) or an issue.
