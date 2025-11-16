# Silibox

**Linux environments, native macOS UX.**
CLI: `sili`

Silibox provides seamless Linux development environments on macOS using Lima VMs and Podman containers. Run Ubuntu (or any Linux distro) containers with proper UID/GID mapping, project directory mounting, and native-feeling command execution.

## Prerequisites

- **macOS** (Apple Silicon recommended for best performance)
- **Homebrew** package manager
- **Go 1.25+** (for building from source)

## Installation

### 1. Install Lima

Lima (Linux Machines) provides the VM infrastructure:

```bash
brew install lima
```

Verify installation:
```bash
limactl --version
```

### 2. Build Silibox

Clone the repository and build:

```bash
git clone https://github.com/coheez/silibox.git
cd silibox
make build
```

The binary will be available at `./bin/sili`.

Optional: Install globally
```bash
make install
# Or manually:
sudo cp bin/sili /usr/local/bin/
```

### 3. Verify Installation

Run the doctor command to check your environment:

```bash
./bin/sili doctor
```

This will check for Lima installation and report any issues.

## Quick Start Guide

### Step 1: Start the VM

Create and start a Silibox VM with custom resources:

```bash
# Default: 4 CPUs, 8GiB RAM, 60GiB disk
./bin/sili vm up

# Custom resources:
./bin/sili vm up --cpus 6 --memory 12GiB --disk 100GiB
```

The VM will:
- Download Ubuntu LTS image (first time only)
- Install Podman inside the VM
- Configure networking and mounts
- Start in ~1-2 minutes

Check VM status:
```bash
./bin/sili vm status

# Live status (queries Lima directly):
./bin/sili vm status --live

# JSON output:
./bin/sili vm status --json
```

### Step 2: Create a Development Environment

Create a container with your project mounted:

```bash
# In your project directory:
cd ~/myproject

# Create container with project mounted at /workspace:
./bin/sili create \
  --name myapp \
  --image ubuntu:22.04 \
  --dir . \
  --workdir /workspace
```

Options:
- `--name` - Container name (default: silibox-dev)
- `--image` - Docker/OCI image (default: ubuntu:22.04)
- `--dir` - Local directory to mount (default: current directory)
- `--workdir` - Working directory inside container (default: /workspace)
- `--user` - User to run as (default: current user with UID/GID mapping)

The container will:
- Pull the specified image
- Mount your project directory with read/write access
- Mount your home directory at `/home/host` (read-only)
- Map your host UID/GID for correct file permissions
- Run in the background

### Step 3: Enter the Environment

Start an interactive shell in your container:

```bash
./bin/sili enter --name myapp

# Use a different shell:
./bin/sili enter --name myapp --shell zsh
```

You're now inside the container at `/workspace` with:
- Your project files mounted and writable
- Your host home directory accessible at `/home/host`
- Correct file ownership (no permission issues!)
- Full Linux environment (apt, git, build tools, etc.)

### Step 4: Run Commands

Execute commands non-interactively:

```bash
# Run a command in the container:
./bin/sili run --name myapp ls -la

# Run build commands:
./bin/sili run --name myapp npm install
./bin/sili run --name myapp make test
```

The exit code is preserved, making it perfect for scripts and CI.

## Container Management

### List Containers

```bash
./bin/sili ps
```

### Stop a Container

```bash
./bin/sili stop myapp
```

### Start a Stopped Container

```bash
./bin/sili start myapp
```

### View Container Logs

```bash
# Show last 50 lines:
./bin/sili logs myapp

# Follow logs:
./bin/sili logs myapp --follow

# Show last 100 lines:
./bin/sili logs myapp --tail 100
```

### Remove a Container

```bash
# Remove stopped container:
./bin/sili rm myapp

# Force remove running container:
./bin/sili rm myapp --force
```

## VM Management

### Stop the VM

```bash
./bin/sili vm stop
```

This stops the VM and all containers inside it.

### Restart the VM

```bash
./bin/sili vm stop
./bin/sili vm up
```

### Check Podman in VM

```bash
./bin/sili vm probe
```

Runs `podman run hello-world` inside the VM to verify Podman is working.

## State Management

Silibox tracks all VMs, containers, and configuration in `~/.sili/state.json`.

View current state:
```bash
./bin/sili state show
```

This shows:
- VM configuration and status
- All containers and their metadata
- Port mappings
- Mount points
- Timestamps

## Troubleshooting

### Doctor Command

The doctor command checks your entire setup:

```bash
./bin/sili doctor
```

It will report:
- ✓ Lima installation
- ✓ VM status
- ✓ Podman availability in VM
- ✓ State consistency
- ⚠️ Any warnings or issues

### Common Issues

**VM won't start:**
```bash
# Check Lima status:
limactl list

# Stop and recreate:
./bin/sili vm stop
./bin/sili vm up
```

**Container permission issues:**
- Silibox automatically maps your UID/GID, so files created in the container will have the correct owner on the host

**State out of sync:**
```bash
# Check live status:
./bin/sili vm status --live

# View state:
./bin/sili state show
```

## Development Workflow Example

```bash
# Start VM (once)
./bin/sili vm up

# Create a Node.js dev environment
cd ~/my-node-project
./bin/sili create --name nodedev --image node:20

# Enter and set up
./bin/sili enter --name nodedev
# Inside container:
$ npm install
$ npm run dev
$ exit

# Run commands from host
./bin/sili run --name nodedev npm test
./bin/sili run --name nodedev npm run build

# View logs
./bin/sili logs nodedev --follow

# Clean up when done
./bin/sili stop nodedev
./bin/sili rm nodedev
```

## Architecture

- **Host (macOS)** → runs `sili` CLI
- **Lima VM (Linux)** → lightweight Ubuntu VM with Podman
- **Podman Containers** → your development environments

Benefits:
- Native Linux tooling and packages
- Proper file permissions via UID/GID mapping
- Fast, lightweight (compared to Docker Desktop)
- Apple Silicon optimized (uses Virtualization.framework)
- Multiple isolated environments

## For Internal Alpha Testers

This is early-stage software. Please report issues with:
- Platform: macOS version, chip (Intel/Apple Silicon)
- Command that failed
- Full error output
- Output of `./bin/sili doctor`

## Commands Reference

| Command | Description |
|---------|-------------|
| `sili version` | Show version info |
| `sili doctor` | Check environment and dependencies |
| `sili vm up` | Create/start VM |
| `sili vm status` | Show VM status |
| `sili vm stop` | Stop VM |
| `sili vm probe` | Test Podman in VM |
| `sili create` | Create a container |
| `sili enter` | Enter container shell |
| `sili run` | Run command in container |
| `sili ps` | List containers |
| `sili start` | Start stopped container |
| `sili stop` | Stop container |
| `sili rm` | Remove container |
| `sili logs` | View container logs |
| `sili state show` | View state file |

## What's Next?

Coming soon:
- Port forwarding
- Volume management
- Command shims (run container commands directly from host)
- Environment templates
- Multi-container orchestration

## License

[Add license information]