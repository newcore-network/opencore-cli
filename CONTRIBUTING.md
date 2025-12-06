# Contributing to OpenCore CLI

Thank you for your interest in contributing to OpenCore CLI! This document provides guidelines and instructions for contributing.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Node.js (for testing NPM wrapper)

### Development Setup

1. Fork the repository
2. Clone your fork:

   ```bash
   git clone https://github.com/YOUR_USERNAME/opencore-cli
   cd opencore-cli
   ```

3. Install dependencies:

   ```bash
   go mod download
   ```

4. Build the project:
   ```bash
   make build
   ```

## Development Workflow

### Running Locally

```bash
go run ./cmd/opencore [command]
```

### Running Tests

```bash
go test -v ./...
```

### Building

```bash
make build        # Build for current platform
make build-all    # Build for all platforms
```

## Code Style

- Follow standard Go conventions
- Use `gofmt` to format your code
- Run `golangci-lint run` before committing

## Commit Messages

- Use clear and meaningful commit messages
- Follow conventional commits format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation changes
  - `refactor:` for code refactoring
  - `test:` for adding tests
  - `chore:` for maintenance tasks

Example:

```
feat: add support for custom templates
fix: resolve path issue on Windows
docs: update installation instructions
```

## Pull Request Process

1. Create a new branch for your feature/fix:

   ```bash
   git checkout -b feat/my-new-feature
   ```

2. Make your changes and commit them

3. Push to your fork:

   ```bash
   git push origin feat/my-new-feature
   ```

4. Open a Pull Request on GitHub

5. Ensure all CI checks pass

6. Wait for review and address any feedback

## Reporting Issues

When reporting issues, please include:

- OpenCore CLI version (`opencore --version`)
- Operating system and version
- Go version (if building from source)
- Steps to reproduce the issue
- Expected vs actual behavior
- Error messages and logs

## Feature Requests

We welcome feature requests! Please open an issue with:

- Clear description of the feature
- Use cases and benefits
- Any relevant examples or mockups

## License

By contributing to OpenCore CLI, you agree that your contributions will be licensed under the MPL-2.0 license.

## Questions?

Feel free to open a discussion on GitHub or reach out to the maintainers.

---

**Thank you for contributing to OpenCore CLI!** 🚀
