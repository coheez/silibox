# Silibox Roadmap

This document outlines the development roadmap for Silibox v0.1, organized into sprints.

## Overview

**Current Status:** Sprint 1 Complete ✅
- CLI scaffold with Cobra
- Lima VM lifecycle management (`vm up`, `vm status`, `vm stop`, `vm probe`)
- `sili doctor` for diagnostics
- Basic state store skeleton (`~/.sili/state.json`)

**Goal:** After Sprint 4, Silibox is a shippable MVP with Distrobox-level ergonomics + auto-sleep magic.

---

## Sprint Progression

| Sprint | Outcome | Status |
|--------|---------|--------|
| Sprint 0 | VM + CLI foundation | ✅ Complete |
| Sprint 1 | (Same as Sprint 0) | ✅ Complete |
| Sprint 2 | Real Linux environments (core MVP) | 📋 Next |
| Sprint 3 | Performance + Distrobox-level ergonomics | 🔜 Planned |
| Sprint 4 | Power efficiency + "it disappears when idle" magic | 🔜 Planned |

---

## Sprint 0/1 — Foundation ✅

**Status:** Complete

### Completed Features
- CLI scaffold and command structure
- Lima VM lifecycle (`vm up`, `vm status`, `vm stop`, `vm probe`)
- State store foundation (`~/.sili/state.json`)
- `sili doctor` for basic health checks
- `sili version` command

**Git History:**
- COH-5: `sili vm up`
- COH-6: `sili vm status`
- COH-7: `sili vm stop`
- COH-8: `sili create` (initial)
- COH-9: `sili enter` (initial)
- COH-10: `sili run` (initial)
- COH-11: State store implementation
- COH-12: `sili doctor`

---

## Sprint 2 — Core Environments (Distrobox-like MVP)

**Goal:** You can create, enter, and run Linux environments reliably. This is the real MVP.

**Success Criteria:**
```bash
sili create -i ubuntu:22.04 -n myenv
sili enter myenv
sili run myenv -- python3 --version
sili ls
```

### Stories (9 total)

1. **Implement `sili create` (core)** [High/runtime]
   - Create a named Podman container inside the Lima VM
   - Pull image if missing
   - Mount project dir to /work
   - Record container_id in state.json

2. **UID/GID user mapping in container** [High/runtime]
   - Inside container, create a user matching host UID/GID
   - Exec as that user by default
   - Prevent permission issues on mounted files

3. **Implement `sili enter`** [High/cli]
   - Interactive shell into a running container using podman exec via limactl shell
   - Default cwd=/work

4. **Implement `sili run`** [High/cli]
   - Run non-interactive commands inside container
   - Capture exit code and forward stdout/stderr

5. **Implement `sili ls`** [Medium/cli]
   - List all known environments from state.json with status (running/stopped/image)

6. **Implement `sili stop <env>`** [Medium/runtime]
   - Stop a running container without deleting it
   - Update env status in state

7. **Implement `sili rm <env>`** [Medium/runtime]
   - Remove container and associated volumes
   - Clean state.json entries safely

8. **Persist env metadata in state store** [High/state]
   - Store image, container_id, mounts, volumes, ports, status, last_active per env

9. **Basic error handling + recovery** [Medium/infra]
   - Detect missing containers or desynced state
   - Provide actionable error messages

---

## Sprint 3 — Performance & Ergonomics

**Goal:** Make Silibox feel fast and invisible. This sprint differentiates from Docker Desktop.

**Success Criteria:**
- `node_modules`, `.venv`, `target/` don't destroy performance
- Users don't think about file I/O
- Commands feel host-native

### Stories (9 total)

1. **Auto-detect project language stack** [High/perf]
   - Detect Node, Python, Rust, Go projects based on files (package.json, pyproject.toml, Cargo.toml)

2. **Volume mapping for hot directories** [High/perf]
   - Automatically move node_modules, .venv, target, .cargo, .pnpm-store into Podman volumes

3. **One-time migration for existing dirs** [Medium/perf]
   - If hot dirs already exist on host, rsync them into volumes
   - Clean host copy after confirmation

4. **Watcher command detection** [Medium/perf]
   - Detect commands like pnpm dev, vite, pytest -f, cargo watch
   - Tag as watcher workloads

5. **Auto-enable polling for watchers** [Medium/perf]
   - Set env vars or flags (CHOKIDAR_USEPOLLING=1, --watch-poll)
   - Apply when watchers run on mounted dirs

6. **Implement `sili export-bin`** [High/cli]
   - Create host shims that proxy commands into containers (node, python, psql)
   - Register in state.json

7. **Add `~/.sili/bin` PATH helper** [Low/cli]
   - Detect if ~/.sili/bin is on PATH
   - Prompt user to add it if missing

8. **Port mapping support** [High/networking]
   - Allow declaring ports in env creation
   - Track and avoid port collisions via state.json

9. **Implement `sili ports`** [Medium/cli]
   - List active port forwards with friendly localhost URLs

---

## Sprint 4 — Autosleep, Power & UX Polish

**Goal:** Silibox uses zero resources when idle and wakes instantly when needed.

**Success Criteria:**
- Zero CPU/memory usage when idle
- Instant wake-up on demand
- Better battery life than native Linux laptops

### Stories (9 total)

1. **Implement idle detection agent** [High/agent]
   - Monitor container CPU, disk I/O, active shells, and port activity
   - Determine idleness threshold

2. **Track last_active timestamps** [High/state]
   - Update env and VM last_active fields on enter/run/port access

3. **Auto-stop containers on idle** [High/agent]
   - Stop non-persistent containers when idle threshold exceeded

4. **Auto-stop VM on full idle** [High/agent]
   - Stop Lima VM when no active containers and idle threshold reached

5. **Auto-wake VM on demand** [High/vm]
   - Ensure any sili command automatically starts the VM if stopped

6. **Persistent services flag** [Medium/agent]
   - Allow envs or services (e.g. Postgres) to opt out of autosleep

7. **Implement `sili vm sleep` and `sili vm wake`** [Medium/cli]
   - Manual override commands for power management

8. **Autosleep configuration** [Low/config]
   - Allow idle timeout configuration via config file or flags

9. **Improve `sili doctor` with fixes** [Medium/cli]
   - Detect orphaned containers, missing volumes, stale state
   - Offer --fix option

---

## Important Development Notes

### Build Order Philosophy

**DO NOT skip ahead.** Each sprint builds on the previous:

1. **Sprint 2 first** — If you can't reliably `create`/`enter`/`run`, nothing else matters
2. **Sprint 3 second** — Performance issues will kill adoption faster than missing features
3. **Sprint 4 third** — Auto-sleep is the "magic" differentiator but requires stable foundation

### Critical Rule

**Do not start VS Code integration or GUI stuff until Sprint 3 is solid.**

If `sili enter` + `sili run` don't feel instant and reliable, everything else is noise.

### Performance is a Feature

- Sprint 3's volume mapping is critical for real-world usage
- File I/O performance makes or breaks the developer experience
- Watcher support is non-negotiable for modern dev workflows

---

## Future Considerations

Potential Sprint 5 topics (not yet scoped):
- Beta polish + comprehensive docs
- Homebrew tap for distribution
- VS Code Remote integration
- Multi-environment orchestration
- GUI/TUI for environment management
- Config profiles and templates

---

## CSV Files for Linear Import

Sprint backlogs are available as CSV files in this directory:

- `silibox-sprint0-foundation.csv` — Sprint 0/1 (✅ complete)
- `silibox-sprint2-envs.csv` — Sprint 2 stories
- `silibox-sprint3-perf.csv` — Sprint 3 stories  
- `silibox-sprint4-autosleep.csv` — Sprint 4 stories

### Import Instructions

1. Open Linear project "Silibox v0.1"
2. Go to Settings → Import
3. Upload CSV file
4. Map columns: Title → Title, Description → Description, etc.
5. All stories import as Status: Todo

---

## Labels Reference

- `cli` — Command-line interface changes
- `runtime` — Container runtime and execution
- `state` — State store modifications
- `perf` — Performance optimizations
- `networking` — Network and port management
- `agent` — Background agent/daemon work
- `vm` — Lima VM management
- `config` — Configuration system
- `infra` — Infrastructure and tooling
- `docs` — Documentation
- `release` — Release engineering

---

**Last Updated:** December 24, 2024  
**Current Sprint:** Sprint 2 (starting)  
**Source:** Planning agent output
