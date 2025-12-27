# Silibox

**Linux environments, native macOS UX**

Silibox provides seamless Linux development environments on macOS using Lima (Linux on Mac) and Podman containers. Get the power of Linux tooling with the convenience of native macOS integration.

## 🚀 Quick Start

### Prerequisites

- **macOS** (Apple Silicon recommended)
- **Go 1.22+** for building from source
- **Lima** for Linux VM management

### Installation

**One-line install:**
```bash
curl -fsSL https://raw.githubusercontent.com/coheez/silibox/main/scripts/install.sh | bash
```

This will download and install the latest release. Then:

```bash
brew install lima  # Install Lima VM manager
sili doctor        # Verify installation
```

**Or build from source:**
```bash
git clone https://github.com/coheez/silibox.git
cd silibox
make build
make install  # Optional: install globally
```

## 📖 Usage

### 1. Check Your Environment

```bash
./bin/sili doctor
```

This will verify:
- ✅ Lima installation
- ✅ VM status
- ✅ Podman availability in VM
- ✅ State consistency

### 2. Start the VM

```bash
./bin/sili vm up
```

This creates and starts a Lima VM with:
- Ubuntu 22.04 LTS
- Apple Virtualization.framework (vz)
- Podman pre-installed
- 4 vCPUs, 8GB RAM, 60GB disk (configurable)

### 3. Create a Development Environment

```bash
# Basic usage (uses current directory)
./bin/sili create --name my-project

# Custom configuration
./bin/sili create \
  --name my-app \
  --image ubuntu:22.04 \
  --dir /path/to/project \
  --workdir /workspace

# Create persistent service (won't auto-sleep)
./bin/sili create --name postgres --image postgres:15 --persistent
```

This creates a Podman container with:
- Your project directory mounted at `/workspace`
- Your home directory mounted read-only at `/home/host`
- UID/GID mapping for seamless file permissions
- Environment variables from your host
- Optional `--persistent` flag to prevent auto-sleep

### 4. Enter Your Environment

```bash
# Enter interactive shell
./bin/sili enter --name my-project

# Use different shell
./bin/sili enter --name my-project --shell zsh
```

### 5. Run Commands

```bash
# Run single commands (non-interactive)
./bin/sili run --name my-project -- ls -la
./bin/sili run --name my-project -- make build
./bin/sili run --name my-project -- python script.py
```

## 🛠️ Commands Reference

### VM Management

```bash
# Start/restart VM
./bin/sili vm up

# Check VM status
./bin/sili vm status
./bin/sili vm status --live    # Get live status from lima

# Stop VM
./bin/sili vm stop

# Power management (user-friendly aliases)
./bin/sili vm sleep           # Put VM to sleep
./bin/sili vm wake            # Wake VM

# Test runtime (hello-world container)
./bin/sili vm probe
```

### Container Management

```bash
# Create environment
./bin/sili create --name my-env --image ubuntu:22.04

# Create persistent service (won't auto-sleep)
./bin/sili create --name postgres --persistent

# List environments
./bin/sili ls

# Enter interactive shell
./bin/sili enter --name my-env

# Run commands
./bin/sili run --name my-env -- command args

# Stop/remove environment
./bin/sili stop --name my-env
./bin/sili rm --name my-env

# View state
./bin/sili state show
```

### Autosleep Agent

```bash
# Run autosleep agent (auto-stops idle containers and VM)
./bin/sili agent autosleep

# With custom timeouts
./bin/sili agent autosleep --container-timeout 10m --vm-timeout 20m

# Don't stop VM, only containers
./bin/sili agent autosleep --no-stop-vm
```

📚 **See [docs/AUTOSLEEP.md](docs/AUTOSLEEP.md) for comprehensive autosleep documentation**

### Diagnostics

```bash
# Comprehensive health check
./bin/sili doctor

# Auto-repair common issues
./bin/sili doctor --fix

# Show version info
./bin/sili version
```

## 🔧 Configuration

### Config File

Create `~/.sili/config.yaml` to configure autosleep behavior:

```yaml
autosleep:
  container_timeout: 15m    # Idle timeout for containers
  vm_timeout: 30m           # Idle timeout for VM
  poll_interval: 30s        # How often to check
  no_stop_vm: false         # Disable VM auto-stop
```

Command-line flags override config file settings.

### VM Resources

```bash
./bin/sili vm up --cpus 8 --memory 16GiB --disk 100GiB
```

### Container Settings

```bash
./bin/sili create \
  --name my-env \
  --image ubuntu:22.04 \
  --dir /path/to/project \
  --workdir /workspace \
  --user myuser \
  --persistent              # Opt out of autosleep
```

## 📁 Project Structure

```
silibox/
├── cmd/sili/main.go              # CLI entry point
├── internal/
│   ├── agent/                    # Autosleep agent & idle detection
│   ├── cli/                      # Cobra commands
│   ├── config/                   # Config file management
│   ├── container/                # Container operations
│   ├── lima/                     # VM management
│   ├── runtime/                  # Runtime probes
│   ├── shim/                     # Binary shim generation
│   ├── stack/                    # Stack management
│   ├── state/                    # State management
│   └── vm/                       # VM utility functions
├── build/lima/templates/         # Lima VM templates
├── scripts/dev/                  # Development scripts
└── Makefile                      # Build system
```

## 🏗️ Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Lint code
make lint

# Install globally
make install
```

### State Management

Silibox maintains state in `~/.sili/state.json`:
- VM configuration and status
- Created environments
- Port allocations
- Shim registrations

### Debugging

```bash
# View current state
./bin/sili state show

# Check live VM status
./bin/sili vm status --live

# Run diagnostics
./bin/sili doctor
```

## 🐛 Troubleshooting

### Common Issues

**VM won't start:**
```bash
./bin/sili doctor
./bin/sili vm stop
./bin/sili vm up
```

**Container not found:**
```bash
./bin/sili create --name my-env
./bin/sili enter --name my-env
```

**Permission issues:**
- Silibox automatically maps your UID/GID
- Check file ownership in mounted directories

**State inconsistencies:**
```bash
./bin/sili doctor --fix        # Auto-repair state issues
./bin/sili vm status --live
./bin/sili state show
```

### Getting Help

1. Run `./bin/sili doctor` for diagnostics
2. Check `~/.sili/state.json` for state issues
3. Use `--live` flags to bypass state cache
4. View logs with `limactl show-ssh silibox`

## 🚧 Alpha Status

This is an internal alpha release. Features may change and bugs are expected.

**Known Limitations:**
- Only supports macOS (Apple Silicon preferred)
- Requires Lima for VM management
- Container networking is basic
- No automatic port forwarding yet

**Recent Features (Sprint 4):**
- ✅ Autosleep agent with idle detection
- ✅ Auto-wake VM on demand
- ✅ Persistent services flag
- ✅ Power management commands (`vm sleep`, `vm wake`)
- ✅ Config file support (`~/.sili/config.yaml`)
- ✅ `doctor --fix` auto-repair

**Planned Features:**
- Port forwarding and service exposure
- Volume management
- Enhanced shim generation
- Stack management improvements

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE) for details.

---

**Happy coding! 🎉**

For questions or issues, please check the troubleshooting section or run `./bin/sili doctor`.