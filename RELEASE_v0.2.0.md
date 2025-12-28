# Release v0.2.0 - Preparation Guide

## Release Date: December 28, 2025

### What's New in v0.2.0

This release adds **multi-provider support** for Mistral AI and Claude (Anthropic), along with comprehensive testing and documentation improvements.

---

## Pre-Release Checklist

### ✅ Code Changes
- [x] Added Mistral provider (`internal/semantic/mistral_provider.go`)
- [x] Added Claude provider (`internal/semantic/claude_provider.go`)
- [x] Created comprehensive unit tests (`internal/semantic/provider_test.go`)
- [x] All tests passing (`go test ./internal/semantic/...`)

### ✅ Documentation
- [x] Updated README.md with:
  - New provider configuration section
  - Environment variable setup for all providers
  - Updated architecture overview
  - Updated roadmap (v0.2.0 marked as released)
- [x] Created CHANGELOG.md with detailed release notes

### 📋 Pre-Release Testing

Run the following commands to ensure everything works:

```bash
# 1. Run all tests
go test ./...

# 2. Build the application
go build -o prompt-cache ./cmd/api

# 3. Verify Docker build
docker build -t prompt-cache:v0.2.0 .

# 4. Run linting (if available)
golangci-lint run ./...
```

---

## Release Steps

### 1. Version Tagging

```bash
# Ensure you're on main branch
git checkout main
git pull origin main

# Create and push version tag
git tag -a v0.2.0 -m "Release v0.2.0: Multi-provider support (Mistral, Claude)"
git push origin v0.2.0
```

### 2. GitHub Release

Create a new release on GitHub with the following:

**Release Title**: `v0.2.0 - Multi-Provider Support`

**Release Description**:
```markdown
# 🚀 PromptCache v0.2.0

## What's New

### 🎯 Multi-Provider Support
We've added native support for **Mistral AI** and **Claude (Anthropic)**! You can now use PromptCache with your preferred AI provider.

**Supported Providers:**
- ✅ OpenAI (existing)
- ✅ Mistral AI (new)
- ✅ Claude/Anthropic (new)

### 🧪 Enhanced Testing
- Comprehensive unit tests for all providers
- Mock HTTP client for reliable testing
- 100% test coverage for provider implementations

### 📚 Better Documentation
- New provider configuration guide
- Environment setup instructions for all providers
- Updated architecture overview

## Quick Start

### OpenAI
```bash
export OPENAI_API_KEY=sk-...
```

### Mistral
```bash
export MISTRAL_API_KEY=your_mistral_key
```

### Claude
```bash
export ANTHROPIC_API_KEY=your_anthropic_key
export VOYAGE_API_KEY=your_voyage_key
```

## Installation

**Docker:**
```bash
docker pull messkan/prompt-cache:v0.2.0
```

**From Source:**
```bash
git clone https://github.com/messkan/prompt-cache.git
cd prompt-cache
git checkout v0.2.0
go build -o prompt-cache ./cmd/api
```

## Full Changelog
See [CHANGELOG.md](CHANGELOG.md) for complete details.

## What's Next?

v0.3.0 will focus on:
- Enhanced configuration options
- Dynamic provider selection via API
- Improved cache management

---

**⭐ If you find PromptCache useful, please give us a star!**
```

### 3. Docker Hub Release

```bash
# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 \
  -t messkan/prompt-cache:v0.2.0 \
  -t messkan/prompt-cache:latest \
  --push .
```

### 4. Update Docker Hub README

Update the DOCKERHUB_README.md with v0.2.0 information:

```bash
# If DOCKERHUB_README.md needs updating, do so and push to Docker Hub
```

### 5. Announcement

Post announcement in:
- GitHub Discussions
- Project README
- Social media (if applicable)

---

## Post-Release Checklist

- [ ] Verify GitHub release is published
- [ ] Verify Docker image is available on Docker Hub
- [ ] Update project website (if applicable)
- [ ] Monitor for issues/bug reports
- [ ] Respond to community feedback

---

## Environment Variables Reference

### OpenAI
- `OPENAI_API_KEY` - Required for OpenAI provider

### Mistral
- `MISTRAL_API_KEY` - Required for Mistral provider

### Claude (Anthropic)
- `ANTHROPIC_API_KEY` - Required for Claude verification
- `VOYAGE_API_KEY` - Required for embeddings (Voyage AI)

---

## Testing the Release

### Manual Testing Steps

1. **Test OpenAI Provider**
```bash
export OPENAI_API_KEY=your_key
./prompt-cache
# Test with OpenAI client
```

2. **Test Mistral Provider**
```bash
export MISTRAL_API_KEY=your_key
# Update config to use Mistral provider
./prompt-cache
```

3. **Test Claude Provider**
```bash
export ANTHROPIC_API_KEY=your_key
export VOYAGE_API_KEY=your_voyage_key
# Update config to use Claude provider
./prompt-cache
```

---

## Rollback Plan

If critical issues are found:

```bash
# Revert to v0.1.0
git checkout v0.1.0

# Remove problematic tag
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0

# Issue hotfix if needed
git checkout -b hotfix/v0.2.1
# Fix issues
git tag -a v0.2.1 -m "Hotfix for v0.2.0"
```

---

## Notes

- All providers implement the same interfaces for consistency
- Claude uses Voyage AI for embeddings (as recommended by Anthropic)
- Unit tests use mocked HTTP clients for reliable testing
- No breaking changes from v0.1.0
