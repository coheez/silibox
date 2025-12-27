# Development Guide

This document provides detailed information for developers working on Silibox.

## 🏗️ Architecture

### Core Components

- **CLI Layer** (`internal/cli/`) - Cobra-based command interface
- **Agent** (`internal/agent/`) - Autosleep agent and idle detection
- **Config** (`internal/config/`) - Configuration file management
- **Lima Integration** (`internal/lima/`) - VM lifecycle management
- **Container Management** (`internal/container/`) - Podman operations
- **State Store** (`internal/state/`) - Persistent state management
- **VM Utilities** (`internal/vm/`) - VM helper functions (auto-wake)
- **Runtime Probes** (`internal/runtime/`) - Environment validation
- **Shim** (`internal/shim/`) - Binary shim generation
- **Stack** (`internal/stack/`) - Stack management

### State Management

Silibox uses a JSON-based state store (`~/.sili/state.json`) to track:
- VM configuration and status
- Created environments and their settings
- Port allocations
- Shim registrations
- Host system information

State operations are atomic with file locking to prevent corruption.

## 🔧 Development Setup

### Prerequisites

```bash
# Install dependencies
brew install lima go

# Verify Go version
go version  # Should be 1.22+
```

### Building

```bash
# Clone repository
git clone https://github.com/coheez/silibox.git
cd silibox

# Install dependencies
go mod download

# Build binary
make build

# Run tests
make test

# Lint code
make lint
```

### Testing

```bash
# Unit tests
make test-unit

# Integration tests (requires Lima)
make test-integration

# All tests with coverage
make test-coverage
```

## 📁 Code Organization

### Command Structure

Commands are organized by functionality:

- **Root commands**: `version`, `doctor`, `state`
- **VM commands**: `vm up`, `vm status`, `vm stop`, `vm sleep`, `vm wake`, `vm probe`
- **Container commands**: `create`, `enter`, `run`, `stop`, `rm`, `ls`
- **Agent commands**: `agent autosleep`
- **Export commands**: `export-bin`
- **Stack commands**: `stack create`, `stack up`, `stack down`, `stack rm`

### State Operations

All state modifications use `WithLockedState()` for atomicity:

```go
err := state.WithLockedState(func(s *state.State) error {
    // Modify state here
    s.SetVM(vmInfo)
    return nil
})
```

### Error Handling

- Use `fmt.Errorf()` for error wrapping
- Error strings should be lowercase (ST1005)
- Provide actionable error messages
- Include context in error messages

## 🧪 Testing Strategy

### Unit Tests

- Test individual functions in isolation
- Mock external dependencies (lima, podman)
- Use table-driven tests for multiple scenarios

### Integration Tests

- Test complete workflows
- Require actual Lima VM
- Use `TestLima` prefix for integration tests

### Test Utilities

Located in `internal/testutil/`:
- Mock implementations
- Test data generators
- Common test helpers

## 🔍 Debugging

### State Inspection

```bash
# View current state
./bin/sili state show

# Check state consistency
./bin/sili doctor
```

### Lima Debugging

```bash
# View Lima instances
limactl list

# SSH into VM
limactl shell silibox

# View Lima logs
limactl show-ssh silibox
```

### Container Debugging

```bash
# List containers in VM
limactl shell silibox -- podman ps

# Check container logs
limactl shell silibox -- podman logs container-name
```

## 🚀 Adding Features

### New Commands

1. Add command to appropriate CLI file
2. Implement business logic in internal package
3. Add tests for new functionality
4. Update documentation

### State Schema Changes

1. Increment `SchemaVersion` in `state.go`
2. Add migration logic in `migrate()` function
3. Update JSON schema documentation
4. Test migration with existing state files

### New Dependencies

1. Add to `go.mod` with `go get`
2. Update `go.sum` with `go mod tidy`
3. Document in README if user-facing

## 📋 Code Standards

### Go Conventions

- Use `gofmt` for formatting
- Follow `golangci-lint` rules
- Error strings should be lowercase
- Use meaningful variable names
- Add comments for exported functions

### File Organization

- One package per directory
- Group related functionality
- Keep files under 500 lines when possible
- Use descriptive file names

### Error Messages

- Be specific about what went wrong
- Include suggested fixes
- Use consistent formatting
- Avoid technical jargon for user-facing errors

## 🔄 Release Process

### Version Management

Versions are set via ldflags in Makefile:
- `version` - Git tag or "dev"
- `commit` - Short git hash
- `buildDate` - ISO timestamp

### Pre-release Checklist

- [ ] All tests pass
- [ ] Linting passes
- [ ] Documentation updated
- [ ] State schema stable
- [ ] No breaking changes

## 🐛 Common Issues

### State Corruption

If state becomes corrupted:
1. Backup: `cp ~/.sili/state.json ~/.sili/state.json.bak`
2. Reset: `rm ~/.sili/state.json`
3. Recreate: `./bin/sili vm up`

### Lima Issues

If Lima VM is broken:
1. Stop: `limactl stop silibox`
2. Remove: `limactl delete silibox`
3. Recreate: `./bin/sili vm up`

### Container Issues

If containers are stuck:
1. List: `limactl shell silibox -- podman ps`
2. Stop: `limactl shell silibox -- podman stop container-name`
3. Remove: `limactl shell silibox -- podman rm container-name`

## 📚 Resources

- [Lima Documentation](https://lima-vm.io/)
- [Podman Documentation](https://podman.io/docs/)
- [Cobra CLI Library](https://cobra.dev/)
- [Go Best Practices](https://golang.org/doc/effective_go.html)

---

For questions about development, please refer to the main README or create an issue.
