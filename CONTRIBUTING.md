# Contributing to PromptCache

Thank you for your interest in contributing to PromptCache! This guide will help you get started.

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check existing issues to avoid duplicates
2. Use the latest version to verify the bug still exists

Include in your bug report:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, provider used)
- Relevant logs or error messages

For suspected security vulnerabilities, follow [`SECURITY.md`](SECURITY.md) instead of posting exploit details publicly.

### Suggesting Features

Feature requests are welcome! Please:
1. Check existing issues/roadmap first
2. Clearly describe the use case
3. Explain why it benefits the project
4. Be open to discussion and feedback

### Pull Requests

#### Before You Start

1. **Fork and clone** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

#### Development Setup

```bash
# Install dependencies
go mod download

# Run tests
make test

# Run with your changes
make run
```

#### Making Changes

1. **Write clear, idiomatic Go code**
   - Follow Go conventions and best practices
   - Keep functions small and focused
   - Use meaningful variable names

2. **Add tests for new features**
   ```bash
   go test ./...
   go test ./internal/semantic/
   go test -cover ./...
   ```

3. **Update documentation**
   - Update README.md if needed
   - Add/update code comments
   - Update docs/ if user-facing changes

4. **Run benchmarks** for performance-related changes
   ```bash
   make benchmark
   ```

#### Code Style

- Use `gofmt` to format Go code
- Run `go vet` to catch common issues
- Keep lines under 120 characters when reasonable
- Write clear commit messages

#### Commit Messages

Conventional commits are encouraged:

```text
type(scope): brief description
```

Examples:

```text
feat(semantic): add support for a provider
fix(cache): resolve race condition in concurrent access
docs(api): update provider configuration examples
test(semantic): add benchmark for FindSimilar
```

#### Submitting Your PR

1. Push your branch to your fork
2. Create a pull request against `main`
3. Explain what changed and why
4. Link related issues when applicable

Your PR should pass relevant tests, include tests for new behavior, and update user-facing documentation where needed.

## Development Guidelines

### Project Structure

```text
prompt-cache/
├── cmd/api/          # Main application entry point
├── internal/
│   ├── cache/        # Cache logic
│   ├── semantic/     # Semantic similarity & providers
│   └── storage/      # Storage backends
├── docs/             # Documentation
└── scripts/          # Utility scripts
```

### Performance Considerations

- Benchmark performance-critical code
- Avoid unnecessary allocations
- Use proper concurrency patterns
- Consider cache implications
- Profile before optimizing

## Documentation

Documentation lives in `docs/`. Keep README.md and relevant docs in sync with user-facing behavior.

## Release Process

Maintainers handle releases. Before a release, ensure relevant tests pass and documentation/changelog entries are current.

## Getting Help

- Open a discussion or issue for general questions
- Ask for help in your pull request when blocked
- Use `SECURITY.md` for vulnerability reporting

## License of Contributions

By submitting a contribution to this repository, you agree that the contribution is provided under the repository's MIT License. Do not submit code, documentation, assets, or other material that you do not have the right to contribute under those terms.

## Thank You!

Every contribution helps make PromptCache better. We appreciate your time and effort! 🚀
