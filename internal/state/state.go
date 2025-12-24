package state

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

const (
	StateDir      = ".sili"
	StateFile     = "state.json"
	LockFile      = "state.lock"
	SchemaVersion = 2 // Incremented for MigratedDirs field
)

type State struct {
	Schema    int                  `json:"schema"`
	UpdatedAt time.Time            `json:"updated_at"`
	Host      HostInfo             `json:"host"`
	VM        *VMInfo              `json:"vm,omitempty"`
	Ports     PortRegistry         `json:"ports"`
	Envs      map[string]*EnvInfo  `json:"envs"`
	Shims     map[string]*ShimInfo `json:"shims"`
}

type HostInfo struct {
	UID  int    `json:"uid"`
	GID  int    `json:"gid"`
	Arch string `json:"arch"`
	OS   string `json:"os"`
}

type VMInfo struct {
	Name         string    `json:"name"`
	Backend      string    `json:"backend"`
	Profile      string    `json:"profile"`
	CPUs         int       `json:"cpus"`
	Memory       string    `json:"memory"`
	Disk         string    `json:"disk"`
	Status       string    `json:"status"`
	ConfigSHA256 string    `json:"config_sha256"`
	LastActive   time.Time `json:"last_active"`
}

type EnvInfo struct {
	Name          string            `json:"name"`
	Image         string            `json:"image"`
	Runtime       string            `json:"runtime"`
	ProjectPath   string            `json:"project_path"`
	ContainerID   string            `json:"container_id"`
	Volumes       map[string]string `json:"volumes"`
	Mounts        map[string]Mount  `json:"mounts"`
	Ports         map[string]int    `json:"ports"`
	User          UserInfo          `json:"user"`
	Status        string            `json:"status"`
	Persistent    bool              `json:"persistent"`
	LastActive    time.Time         `json:"last_active"`
	ExportedShims []string          `json:"exported_shims"`
	MigratedDirs  map[string]string `json:"migrated_dirs,omitempty"` // Maps dir name to backup path
}

type Mount struct {
	Host  string `json:"host"`
	Guest string `json:"guest"`
	RW    bool   `json:"rw"`
}

type UserInfo struct {
	UID  int    `json:"uid"`
	GID  int    `json:"gid"`
	Name string `json:"name"`
}

type PortRegistry struct {
	NextEphemeral int              `json:"next_ephemeral"`
	Reserved      map[string][]int `json:"reserved"`
}

type ShimInfo struct {
	Env    string `json:"env"`
	Target string `json:"target"`
}

var (
	statePath string
	lockPath  string
	initOnce  sync.Once
)

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}
	statePath = filepath.Join(homeDir, StateDir, StateFile)
	lockPath = filepath.Join(homeDir, StateDir, LockFile)
}

// WithLockedState executes a function with exclusive access to the state
func WithLockedState(fn func(*State) error) error {
	// Ensure state directory exists
	if err := ensureStateDir(); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Acquire file lock
	lock := flock.New(lockPath)
	locked, err := lock.TryLock()
	if err != nil {
		return fmt.Errorf("failed to acquire state lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("state is locked by another process")
	}
	defer lock.Unlock()

	// Load state
	state, err := Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Execute function
	if err := fn(state); err != nil {
		return err
	}

	// Save state
	return SaveAtomic(state)
}

// Load reads and parses the state file
func Load() (*State, error) {
	initOnce.Do(func() {
		// Initialize state directory if it doesn't exist
		ensureStateDir()
	})

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewState(), nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		// Backup corrupted state and create new one
		backupPath := fmt.Sprintf("%s.bak-%d", statePath, time.Now().Unix())
		if err := os.Rename(statePath, backupPath); err != nil {
			return nil, fmt.Errorf("failed to backup corrupted state: %w", err)
		}
		fmt.Printf("Warning: State file was corrupted and has been backed up to %s\n", backupPath)
		return NewState(), nil
	}

	// Run migrations if needed
	if state.Schema < SchemaVersion {
		if err := migrate(&state, state.Schema, SchemaVersion); err != nil {
			return nil, fmt.Errorf("failed to migrate state: %w", err)
		}
	}

	return &state, nil
}

// SaveAtomic saves state atomically
func SaveAtomic(state *State) error {
	state.UpdatedAt = time.Now()
	state.Schema = SchemaVersion

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file
	tmpPath := statePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Sync file to disk
	file, err := os.OpenFile(tmpPath, os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open temporary file for sync: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}
	file.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, statePath); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	// Sync parent directory
	dir, err := os.Open(filepath.Dir(statePath))
	if err != nil {
		return fmt.Errorf("failed to open state directory: %w", err)
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return fmt.Errorf("failed to sync state directory: %w", err)
	}

	return nil
}

// NewState creates a new empty state
func NewState() *State {
	uid, gid := getCurrentUserIDs()
	return &State{
		Schema:    SchemaVersion,
		UpdatedAt: time.Now(),
		Host: HostInfo{
			UID:  uid,
			GID:  gid,
			Arch: runtime.GOARCH,
			OS:   runtime.GOOS,
		},
		Ports: PortRegistry{
			NextEphemeral: 51000,
			Reserved:      make(map[string][]int),
		},
		Envs:  make(map[string]*EnvInfo),
		Shims: make(map[string]*ShimInfo),
	}
}

// VM helpers
func (s *State) GetVM() *VMInfo {
	return s.VM
}

func (s *State) SetVM(vm *VMInfo) {
	s.VM = vm
}

func (s *State) UpdateVMStatus(status string) {
	if s.VM != nil {
		s.VM.Status = status
	}
}

func (s *State) TouchVMActivity() {
	if s.VM != nil {
		s.VM.LastActive = time.Now()
	}
}

// Environment helpers
func (s *State) UpsertEnv(env *EnvInfo) {
	s.Envs[env.Name] = env
}

func (s *State) GetEnv(name string) *EnvInfo {
	return s.Envs[name]
}

func (s *State) ListEnvs() []*EnvInfo {
	envs := make([]*EnvInfo, 0, len(s.Envs))
	for _, env := range s.Envs {
		envs = append(envs, env)
	}
	return envs
}

func (s *State) RemoveEnv(name string) {
	delete(s.Envs, name)
	// Release ports for this environment
	s.ReleasePorts(name)
}

func (s *State) UpdateEnvStatus(name string, status string) {
	if env := s.Envs[name]; env != nil {
		env.Status = status
	}
}

func (s *State) TouchEnvActivity(name string) {
	if env := s.Envs[name]; env != nil {
		env.LastActive = time.Now()
	}
}

func (s *State) FindEnvByProject(path string) *EnvInfo {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil
	}

	for _, env := range s.Envs {
		if env.ProjectPath == absPath {
			return env
		}
	}
	return nil
}

// Port management
func (s *State) ReservePort(name string, suggested int) (int, error) {
	// Check if suggested port is available
	for _, ports := range s.Ports.Reserved {
		for _, port := range ports {
			if port == suggested {
				// Port is taken, allocate next available
				suggested = s.Ports.NextEphemeral
				s.Ports.NextEphemeral++
			}
		}
	}

	// Reserve the port
	if s.Ports.Reserved[name] == nil {
		s.Ports.Reserved[name] = make([]int, 0)
	}
	s.Ports.Reserved[name] = append(s.Ports.Reserved[name], suggested)

	return suggested, nil
}

func (s *State) ReleasePorts(name string) {
	delete(s.Ports.Reserved, name)
}

// Shim management
func (s *State) RegisterShim(alias, env, targetPath string) {
	s.Shims[alias] = &ShimInfo{
		Env:    env,
		Target: targetPath,
	}
}

func (s *State) UnregisterShim(alias string) {
	delete(s.Shims, alias)
}

func (s *State) ListShims() map[string]*ShimInfo {
	return s.Shims
}

// Utility functions
func ensureStateDir() error {
	dir := filepath.Dir(statePath)
	return os.MkdirAll(dir, 0700)
}

func getCurrentUserIDs() (int, int) {
	uid := os.Getuid()
	gid := os.Getgid()
	return uid, gid
}

func migrate(state *State, from, to int) error {
	// Migrate from v1 to v2: add MigratedDirs field to all environments
	if from < 2 && to >= 2 {
		for _, env := range state.Envs {
			if env.MigratedDirs == nil {
				env.MigratedDirs = make(map[string]string)
			}
		}
	}
	
	state.Schema = to
	return nil
}

// ComputeConfigSHA256 computes SHA256 of Lima config
func ComputeConfigSHA256(config []byte) string {
	hash := sha256.Sum256(config)
	return fmt.Sprintf("%x", hash)
}
