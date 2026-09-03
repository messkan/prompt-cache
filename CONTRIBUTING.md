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

For suspected security vulnerabilities, follow [`SECURITY.md`](SECURITY.md) instead of publishing exploit details in a public issue.

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
   # Run tests
   go test ./...
   
   # Run specific package tests
   go test ./internal/semantic/
   
   # Run with coverage
   go test -cover ./...
   ```

3. **Update documentation**
   - Update README.md if needed
   - Add/update code comments
   - Update docs/ if user-facing changes

4. **Run benchmarks** (for performance-related changes)
   ```bash
   make benchmark
   ```

#### Code Style

- Use `gofmt` to format your code
- Run `go vet` to catch common issues
- Keep lines under 120 characters when reasonable
- Write clear commit messages

#### Commit Messages

Follow conventional commits format:

```
type(scope): brief description

Longer explanation if needed.

Fixes #123
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Maintenance tasks

Examples:
```
feat(semantic): add support for Gemini provider
fix(cache): resolve race condition in concurrent access
docs(api): update provider configuration examples
test(semantic): add benchmark for FindSimilar
```

#### Submitting Your PR

1. **Push your branch** to your fork
2. **Create a pull request** against `main`
3. **Fill out the PR template** completely
4. **Link related issues** using "Fixes #123" or "Relates to #456"

Your PR should:
- Pass all tests
- Include tests for new functionality
- Update relevant documentation
- Have a clear description of changes
- Address reviewer feedback promptly

## Development Guidelines

### Project Structure

```
prompt-cache/
├── cmd/api/          # Main application entry point
├── internal/
│   ├── cache/        # Cache logic
│   ├── semantic/     # Semantic similarity & providers
│   └── storage/      # Storage backends
├── docs/             # Documentation
└── scripts/          # Utility scripts
```

### Adding a New Provider

To add a new embedding/LLM provider:

1. **Create provider file** in `internal/semantic/`:
   ```go
   // newprovider_provider.go
   package semantic
   
   type NewProviderProvider struct {
       apiKey string
   }
   
   func (p *NewProviderProvider) Embed(text string) ([]float64, error) {
       // Implementation
   }
   
   func (p *NewProviderProvider) CheckSimilarity(prompt1, prompt2 string) (bool, error) {
       // Implementation
   }
   ```

2. **Update provider factory** in `semantic.go`:
   ```go
   case "newprovider":
       return &NewProviderProvider{apiKey: apiKey}, nil
   ```

3. **Add tests** in `provider_test.go`

4. **Update documentation**:
   - docs/providers.md
   - README.md
   - docker-compose.yml

### Testing Guidelines

- Write unit tests for all new functions
- Use table-driven tests for multiple scenarios
- Mock external API calls in tests
- Aim for >80% code coverage
- Test edge cases and error conditions

Example test structure:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "expected", false},
        {"invalid input", "", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NewFeature(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewFeature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("NewFeature() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Performance Considerations

- Benchmark performance-critical code
- Avoid unnecessary allocations
- Use proper concurrency patterns
- Consider cache implications
- Profile before optimizing

## Documentation

### Updating Docs

Documentation lives in `docs/` directory:

```bash
# Test docs locally
cd docs
bundle install
bundle exec jekyll serve

# Visit http://localhost:4000/prompt-cache
```

### Documentation Style

- Use clear, concise language
- Include code examples
- Add comments for complex logic
- Keep README.md up to date
- Update CHANGELOG.md for releases

## Release Process

Maintainers handle releases, but contributors should:

1. Update CHANGELOG.md with changes
2. Update version numbers if applicable
3. Ensure all tests pass
4. Update documentation

## Getting Help

- **Questions?** Open a discussion or issue
- **Stuck?** Ask for help in your PR
- **Ideas?** We love to hear them!

## License

By contributing, you agree that your contributions will be licensed under the MIT License. Do not submit material you do not have the right to contribute under those terms.

## Thank You!

Every contribution helps make PromptCache better. We appreciate your time and effort! 🚀
