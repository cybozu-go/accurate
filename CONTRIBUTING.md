# Contributing to Accurate

Thank you for your interest in contributing to Accurate! This document provides guidelines for contributing to the project.

## Code of Conduct

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting Started

### Prerequisites

- Go 1.24.4 or later
- Docker for running tests
- [aqua](https://aquaproj.github.io) for tool management

### Setting up the development environment

1. Fork the repository and clone your fork:
   ```bash
   git clone https://github.com/your-username/accurate.git
   cd accurate
   ```

2. Install development tools using aqua:
   ```bash
   aqua policy allow ./aqua-policy.yaml
   aqua i -l
   ```

3. Build the project:
   ```bash
   make build
   ```

## Development Workflow

### Making Changes

1. Create a new branch for your feature or fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and ensure they follow the project conventions.

3. Run tests to verify your changes:
   ```bash
   make test
   make envtest
   ```

4. Generate any necessary code and manifests:
   ```bash
   make generate
   make manifests
   ```

5. Verify that generated files are up to date:
   ```bash
   make check-generate
   ```

### Testing

The project includes several types of tests:

- **Unit tests**: Run with `make test`
- **Controller tests**: Run with `make envtest`
- **End-to-end tests**: Located in the `e2e/` directory

For e2e tests, you can run them locally using kind:
```bash
cd e2e
make start
# Run your tests
make stop
```

### Code Style

- Follow Go best practices and conventions
- Use `gofmt` for code formatting
- Run `make test` to ensure code passes linting and formatting checks
- Use meaningful commit messages
- Sign off your commits (use `git commit -s` or `git commit --signoff`)

## Submitting Changes

### Pull Request Process

1. Ensure your branch is up to date with the main branch:
   ```bash
   git checkout main
   git pull upstream main
   git checkout your-branch
   git rebase main
   ```

2. Push your changes to your fork:
   ```bash
   git push origin your-branch
   ```

3. Create a pull request with:
   - A clear title describing the change
   - A detailed description of what the PR does
   - Reference to any related issues
   - Appropriate labels (the maintainers will help with this)

### Pull Request Guidelines

- Keep pull requests focused on a single change
- Include tests for new functionality
- Update documentation if needed
- Ensure CI checks pass
- Be responsive to review feedback

### Labels

Release notes are automatically generated based on PR labels. Common labels include:
- `bug`: Something isn't working
- `dependencies`: Dependency updates
- `deprecated`: This feature is deprecated
- `enhancement`: New feature or request
- `removed`: This feature is removed
- `security`: Security-related changes

## Project Structure

- `api/`: Kubernetes API definitions
- `cmd/`: Command-line tools (controller and kubectl plugin)
- `controllers/`: Kubernetes controllers
- `hooks/`: Admission webhooks
- `pkg/`: Shared packages
- `charts/`: Helm chart
- `docs/`: Documentation
- `e2e/`: End-to-end tests

## Release Process

Releases are handled by maintainers following semantic versioning. See [docs/release.md](docs/release.md) for details.

When a new version is released:
- Helm Charts are automatically published
  - Patch versions are automatically incremented
  - To update minor or major versions, modify the corresponding section in [Chart.yaml](charts/accurate/Chart.yaml)
- kubectl plugin is registered with krew for easy installation

## Reporting Issues

- When creating an issue, please use the provided Issue Templates
- For security-related reports, please refer to [SECURITY.md](SECURITY.md)

## Getting Help

- Check the [documentation](https://cybozu-go.github.io/accurate/)
- Open an issue for bugs or feature requests
- Join discussions in existing issues
