# WARP.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Project Overview

Silibox provides seamless Linux development environments on macOS using Lima (Linux on Mac) and Podman containers. The project creates and manages Ubuntu VMs with containerized development environments, providing native macOS UX while leveraging Linux tooling.

**Target Platform**: macOS (Apple Silicon recommended)
**Language**: Go 1.22+
**VM Backend**: Lima with Apple Virtualization.framework (vz)
**Container Runtime**: Podman (inside Lima VM)

## Common Commands

### Building and Testing
```bash
# Build the binary (outputs to bin/sili)
make build

# Run all tests
make test

# Run only unit tests (fast, no external dependencies)
make test-unit

# Run integration tests (requires Lima VM)
make test-integration

# Run tests with coverage information
make test-coverage

# Run tests with verbose output
make test-verbose

# Lint code (requires golangci-lint: brew install golangci-lint)
make lint

# Install binary to /usr/local/bin
make install

# Clean build artifacts
make clean
```

### Running the CLI
```bash
# After building, the binary is at bin/sili
./bin/sili doctor              # Diagnose environment
./bin/sili vm up               # Start VM (creates if needed)
./bin/sili vm status           # Check VM status
./bin/sili vm status --live    # Get live status from Lima
./bin/sili create --name dev   # Create a container environment
./bin/sili enter --name dev    # Enter interactive shell
./bin/sili run --name dev -- <command>  # Run command in container
```

### Testing Individual Components
```bash
# Test a specific package
go test ./internal/lima
go test ./internal/state
go test ./internal/container

# Run a specific test
go test -run TestConfig ./internal/lima

# Test with verbose output
go test -v ./internal/lima
```

## Architecture

### High-Level Structure

Silibox follows a layered architecture:

1. **CLI Layer** (`internal/cli/`) - Cobra-based command interface that handles user input and orchestrates operations
2. **Business Logic** (`internal/lima/`, `internal/container/`) - Core functionality for VM and container management
3. **State Management** (`internal/state/`) - Persistent state store with atomic operations and file locking
4. **Support Utilities** (`internal/runtime/`, `internal/testutil/`) - Probes and test helpers

### State Management Design

**Critical**: All state modifications MUST use `state.WithLockedState()` to ensure atomicity and prevent corruption:

```go
err := state.WithLockedState(func(s *state.State) error {
    // Read/modify state here
    s.SetVM(vmInfo)
    s.UpsertEnv(envInfo)
    return nil  // State is automatically saved on success
})
```

State is stored at `~/.sili/state.json` and includes:
- VM configuration (CPUs, memory, disk, status)
- Container environments (name, image, mounts, user mapping)
- Port allocations
- Shim registrations
- Host system info

The state uses file locking (`~/.sili/state.lock`) to prevent concurrent modifications. State updates are atomic via write-to-temp-then-rename.

### VM and Container Interaction Flow

1. **VM Management** (`internal/lima/`)
   - Creates Lima VM with Ubuntu 22.04 + Podman
   - Uses `limactl` CLI to interact with Lima
   - VM configuration is templated from `build/lima/templates/`
   - VM status can be read from state (fast) or queried live (slow but accurate)

2. **Container Management** (`internal/container/`)
   - Executes Podman commands inside Lima VM via `limactl shell silibox -- podman ...`
   - Handles UID/GID mapping to maintain file permissions
   - Mounts project directory at `/workspace` and home at `/home/host:ro`
   - Containers run with `sleep infinity` to stay alive

3. **Command Execution**
   - All container/VM operations go through `limactl`
   - Interactive commands (`enter`) use `podman exec -it`
   - Non-interactive commands (`run`) capture stdout/stderr and exit codes

### Package Responsibilities

- `cmd/sili/` - Main entry point (calls cli.Execute())
- `internal/cli/` - Command definitions using Cobra (root, vm, container, doctor, state)
- `internal/lima/` - VM lifecycle (up, stop, status), Lima template generation
- `internal/container/` - Container operations (create, enter, run, stop, remove)
- `internal/state/` - State file I/O, locking, migrations, getters/setters
- `internal/runtime/` - Runtime probes (verify Podman works in VM)
- `internal/testutil/` - Test helpers and mocks

## Code Patterns

### Error Handling
- Use `fmt.Errorf()` with `%w` for error wrapping
- Error strings should be lowercase (per Go conventions)
- Provide actionable context in errors
- Return detailed errors from internal packages, user-friendly messages from CLI layer

### Testing Patterns
- Use table-driven tests for multiple scenarios
- Unit tests should not require Lima or external dependencies
- Integration tests are prefixed with `TestLima` and require actual Lima VM
- Mock external commands when possible
- Use `t.TempDir()` for temporary directories in tests

### Version Information
The Makefile injects version info via ldflags:
- `version` - Git tag or "dev"
- `commit` - Short git hash
- `buildDate` - ISO timestamp

These are set in `internal/cli/root.go` and displayed via `sili version`.

### Lima Template System
VM configuration is generated from Go templates in `build/lima/templates/`. The template includes:
- VM resources (CPUs, memory, disk)
- Ubuntu 22.04 base image
- Apple Virtualization.framework (vz) backend
- Podman installation via cloud-init
- virtiofs for fast file sharing

## Important Implementation Details

### UID/GID Mapping
Containers run with the host user's UID/GID to maintain proper file permissions on mounted directories. This is critical for seamless macOS integration.

### State Schema Migrations
When changing the state schema:
1. Increment `SchemaVersion` in `internal/state/state.go`
2. Add migration logic in the `migrate()` function
3. Test with existing state files
4. Corrupted state files are automatically backed up with timestamp

### Lima Instance Name
The Lima instance is always named `silibox` (constant in `internal/lima/lima.go`). This is used across all `limactl` commands.

### Container Lifecycle
Containers created by `sili create` run `sleep infinity` to stay alive. They are:
- Named (not ephemeral container IDs)
- Detached (`-d` flag)
- Mount project dir at `/workspace` (read-write)
- Mount home dir at `/home/host` (read-only)
- Inherit environment variables (PATH, HOME, USER, SHELL, TERM, LANG, LC_ALL)

## Debugging Tips

### Check State Consistency
```bash
./bin/sili doctor                # Comprehensive health check
./bin/sili state show            # View state file
./bin/sili vm status --live      # Bypass state cache
```

### Direct Lima/Podman Access
```bash
limactl list                     # List Lima instances
limactl shell silibox            # SSH into VM
limactl shell silibox -- podman ps         # List containers
limactl shell silibox -- podman logs <name>  # View container logs
```

### State Recovery
If state becomes corrupted:
```bash
cp ~/.sili/state.json ~/.sili/state.json.bak  # Backup
rm ~/.sili/state.json                         # Remove
./bin/sili vm up                              # Recreate
```

## Development Workflow

1. Make code changes
2. Run `make build` to compile
3. Run `make test-unit` for fast feedback
4. Run `make lint` to check code quality
5. Test manually with `./bin/sili` commands
6. For VM-related changes, test with real Lima VM using integration tests
7. Commit changes (follow existing commit message style)

## Dependencies

Core dependencies:
- `github.com/spf13/cobra` - CLI framework
- `github.com/gofrs/flock` - File locking for state management

External tools required:
- Lima (`brew install lima`) - VM management
- Podman (installed automatically in VM)
- Go 1.22+ for building
