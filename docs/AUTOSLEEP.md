# Autosleep & Power Management

Silibox includes an intelligent autosleep system that automatically manages container and VM lifecycle to save system resources when not in use.

## Overview

The autosleep agent monitors container and VM activity and automatically stops resources that have been idle for a configured period. This helps conserve CPU, memory, and battery life on your development machine.

**Key Features:**
- 🌙 Automatic idle detection based on activity timestamps
- 💤 Auto-stop idle containers (default: 15 minutes)
- 🛏️ Auto-stop VM when fully idle (default: 30 minutes)
- ⏰ Auto-wake VM on demand when you run commands
- 🔒 Persistent services that never auto-sleep
- ⚙️ Configurable via config file or CLI flags

## Quick Start

### Run the Autosleep Agent

```bash
# Start with defaults (15m container timeout, 30m VM timeout)
sili agent autosleep

# Custom timeouts
sili agent autosleep --container-timeout 10m --vm-timeout 20m

# Don't stop the VM, only containers
sili agent autosleep --no-stop-vm
```

The agent runs in the foreground and displays activity as it happens:

```
🌙 Autosleep agent starting...
   Container idle timeout: 15m0s
   VM idle timeout: 30m0s
   Poll interval: 30s
   Auto-stop VM: true

💤 Stopping idle container 'dev' (idle for 16 minutes)...
   ✅ Stopped 'dev'
💤 Stopping idle VM (idle for 31 minutes)...
   ✅ VM stopped
```

## How It Works

### Activity Tracking

Silibox tracks activity timestamps for environments and the VM:

1. **Container Activity** - Updated when you:
   - Enter a container (`sili enter`)
   - Run a command (`sili run`)

2. **VM Activity** - Updated when:
   - Any container becomes active
   - VM operations are performed

### Idle Detection

The autosleep agent polls periodically (default: 30 seconds) and checks:

1. **Container Idle Check**:
   - Is the container status "running"?
   - Is `LastActive` timestamp older than `container_timeout`?
   - Is the container marked as `Persistent: false`?
   
   If all conditions are met → Stop the container

2. **VM Idle Check**:
   - Are ALL containers stopped?
   - Is VM `LastActive` timestamp older than `vm_timeout`?
   
   If both conditions are met → Stop the VM

### Auto-Wake on Demand

When you run any command that needs the VM (like `sili enter` or `sili run`), Silibox automatically:
1. Checks if VM is stopped
2. Starts the VM if needed
3. Waits for it to be ready
4. Executes your command

This happens transparently - you don't need to manually start the VM.

```bash
# VM is stopped...
$ sili enter --name dev
⏳ VM is stopped. Starting VM...
✅ VM started successfully
# Now you're in the container
```

## Configuration

### Config File

Create `~/.sili/config.yaml`:

```yaml
autosleep:
  container_timeout: 15m    # How long before stopping idle containers
  vm_timeout: 30m           # How long before stopping idle VM
  poll_interval: 30s        # How often to check for idle resources
  no_stop_vm: false         # Set to true to disable VM auto-stop
```

**Duration Format:**
- `s` - seconds (e.g., `30s`)
- `m` - minutes (e.g., `15m`)
- `h` - hours (e.g., `2h`)
- Combinations: `1h30m`, `90s`

### Command-Line Flags

Flags override config file settings:

```bash
sili agent autosleep \
  --container-timeout 10m \
  --vm-timeout 20m \
  --poll-interval 15s \
  --no-stop-vm
```

**Available Flags:**
- `--container-timeout` - Container idle timeout (default: 15m)
- `--vm-timeout` - VM idle timeout (default: 30m)
- `--poll-interval` - Polling frequency (default: 30s)
- `--no-stop-vm` - Disable VM auto-stop (only stop containers)

## Persistent Services

Mark services that should never auto-sleep with the `--persistent` flag:

```bash
# Create a persistent PostgreSQL service
sili create --name postgres --image postgres:15 --persistent

# Create a persistent Redis service
sili create --name redis --image redis:7 --persistent
```

**Use Cases for Persistent Services:**
- Databases (PostgreSQL, MySQL, Redis)
- Message queues (RabbitMQ, Kafka)
- Long-running services
- Background workers

Persistent environments:
- ✅ Are never stopped by autosleep agent
- ✅ Can still be manually stopped with `sili stop`
- ✅ Show "Yes" in the Persistent column of `sili ls`

## Manual Power Management

In addition to automatic sleep, you can manually control VM power state:

```bash
# Put VM to sleep
sili vm sleep
💤 Putting VM to sleep...
✅ VM is now sleeping

# Wake VM
sili vm wake
⏳ Waking VM...
✅ VM is awake and ready
```

These are user-friendly alternatives to `sili vm stop` and `sili vm up`.

## Best Practices

### Development Workflow

1. **Start autosleep agent in the background**:
   ```bash
   # In a tmux/screen session or as a background service
   sili agent autosleep &
   ```

2. **Create your environments**:
   ```bash
   # Regular dev environment
   sili create --name my-app
   
   # Persistent database
   sili create --name postgres --persistent
   ```

3. **Work normally**:
   ```bash
   sili enter --name my-app
   # VM auto-wakes if needed
   # Activity is tracked automatically
   ```

4. **Let autosleep manage resources**:
   - When you're not using containers, they'll auto-stop after 15 minutes
   - When all containers are stopped and VM is idle, it'll stop after 30 minutes
   - When you need to work again, VM auto-wakes on first command

### Recommended Timeouts

**Fast Laptop (Good Battery):**
```yaml
autosleep:
  container_timeout: 10m
  vm_timeout: 15m
```

**Balanced (Default):**
```yaml
autosleep:
  container_timeout: 15m
  vm_timeout: 30m
```

**Conservative (Slow Startup):**
```yaml
autosleep:
  container_timeout: 30m
  vm_timeout: 1h
```

### Running as a Service

You can run the autosleep agent as a background service using your preferred method:

**Using tmux:**
```bash
tmux new -d -s sili-autosleep 'sili agent autosleep'
```

**Using screen:**
```bash
screen -dmS sili-autosleep sili agent autosleep
```

**Using nohup:**
```bash
nohup sili agent autosleep > ~/.sili/autosleep.log 2>&1 &
```

## Troubleshooting

### Agent Not Stopping Containers

**Check configuration:**
```bash
sili agent autosleep --container-timeout 15m --vm-timeout 30m
# Watch the startup output to verify settings
```

**Verify timestamps:**
```bash
sili state show | grep -A5 "LastActive"
```

### VM Not Auto-Waking

**Check VM status:**
```bash
sili vm status --live
```

**Try manual wake:**
```bash
sili vm wake
```

**Check for errors:**
```bash
sili doctor
```

### Persistent Flag Not Working

**Verify environment is marked persistent:**
```bash
sili ls
# Check the "Persistent" column
```

**Check state:**
```bash
sili state show | grep -B5 -A5 "my-service-name"
```

### Agent Polling Too Often/Slow

**Adjust poll interval:**
```bash
sili agent autosleep --poll-interval 10s  # Check every 10 seconds
sili agent autosleep --poll-interval 60s  # Check every minute
```

Or in config file:
```yaml
autosleep:
  poll_interval: 15s
```

## Advanced Usage

### Disable VM Auto-Stop

If you want containers to auto-sleep but keep the VM running:

```bash
sili agent autosleep --no-stop-vm
```

Or in config:
```yaml
autosleep:
  no_stop_vm: true
```

### Multiple Autosleep Profiles

You can create different config files for different scenarios:

```bash
# Work hours (aggressive)
cp ~/.sili/config-work.yaml ~/.sili/config.yaml
sili agent autosleep

# After hours (conservative)
cp ~/.sili/config-afterhours.yaml ~/.sili/config.yaml
sili agent autosleep
```

### Monitoring Activity

Watch what the agent is doing:

```bash
sili agent autosleep | tee ~/.sili/autosleep.log
```

## Implementation Details

### Idle Detection Logic

The agent uses a simple time-based approach:

```
IsIdle = (CurrentTime - LastActive) > Timeout
```

**Container Idle:**
- Status == "running"
- Persistent == false
- (Now - LastActive) > container_timeout

**VM Idle:**
- All containers stopped
- (Now - VM.LastActive) > vm_timeout

### Activity Tracking

Activity is tracked at two levels:

1. **Environment Level** - Each environment has:
   - `LastActive` timestamp
   - `Persistent` flag

2. **VM Level** - The VM has:
   - `LastActive` timestamp (synced with any container activity)

### State Updates

Activity tracking uses non-blocking state updates:
- If state update fails, a warning is logged
- Command execution continues normally
- This prevents state issues from blocking your workflow

## FAQ

**Q: What happens if I close the agent?**  
A: Auto-sleep stops. Containers and VM won't be automatically stopped. You can restart the agent anytime.

**Q: Can I run multiple agents?**  
A: No, only run one agent at a time. Multiple agents would conflict.

**Q: Does autosleep affect running commands?**  
A: No. Active containers are never stopped. Only idle containers (no recent `enter` or `run` activity) are affected.

**Q: What if VM is starting when I run a command?**  
A: The command waits for the VM to be ready before proceeding. You'll see "⏳ VM is stopped. Starting VM..." and then "✅ VM started successfully".

**Q: Can I change timeouts while agent is running?**  
A: No, you need to stop (Ctrl+C) and restart the agent with new settings.

**Q: Does autosleep work with stacks?**  
A: Yes! Each environment in a stack is tracked independently. Mark services as persistent if needed.

## See Also

- [Main README](../README.md) - General Silibox documentation
- [Development Guide](DEVELOPMENT.md) - Development and contribution guide
- [WARP.md](../WARP.md) - Detailed architecture and patterns
