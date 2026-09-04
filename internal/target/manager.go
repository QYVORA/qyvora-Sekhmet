// Package target manages the fuzzing target abstraction: target definition,
// selection, validation and auto-detection of the target class. The target is
// always explicit and always authorization-gated; sekhmet never guesses a
// remote target silently.
package target

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Manager tracks the current fuzz target and the set of defined targets.
type Manager struct {
	mu      sync.Mutex
	dir     string
	current *models.Target
	all     map[string]*models.Target
}

const stateFile = "targets.json"

// NewManager returns an empty target manager. If dir is non-empty it is used
// as the persistence directory; targets are persisted there so the selection
// survives across invocations.
func NewManager(dir string) *Manager {
	m := &Manager{dir: dir, all: make(map[string]*models.Target)}
	if dir != "" {
		_ = m.load()
	}
	return m
}

// Set records a target as both current and stored, persisted when a dir is
// configured.
func (m *Manager) Set(t *models.Target) error {
	m.mu.Lock()
	if t.ID == "" {
		t.ID = models.NewSessionID()
	}
	m.all[t.ID] = t
	m.current = t
	m.mu.Unlock()
	return m.Persist()
}

// Current returns the current target, or nil.
func (m *Manager) Current() *models.Target {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current
}

// Get returns a target by id.
func (m *Manager) Get(id string) (*models.Target, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.all[id]
	return t, ok
}

// List returns all defined targets in insertion order.
func (m *Manager) List() []*models.Target {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*models.Target, 0, len(m.all))
	for _, t := range m.all {
		out = append(out, t)
	}
	return out
}

// ChangeTarget updates the current target id, persisting the selection.
func (m *Manager) ChangeTarget(id string) bool {
	m.mu.Lock()
	t, ok := m.all[id]
	if !ok {
		m.mu.Unlock()
		return false
	}
	m.current = t
	m.mu.Unlock()
	_ = m.Persist()
	return true
}

// Persist writes targets and the current selection to the configured dir.
func (m *Manager) Persist() error {
	if m.dir == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	state := struct {
		Current string                    `json:"current"`
		Targets map[string]*models.Target `json:"targets"`
	}{
		Current: currentID(m.current),
		Targets: m.all,
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dir, stateFile), data, 0o600)
}

// load restores targets and the current selection from the configured dir.
func (m *Manager) load() error {
	data, err := os.ReadFile(filepath.Join(m.dir, stateFile))
	if err != nil {
		return err
	}
	state := struct {
		Current string                    `json:"current"`
		Targets map[string]*models.Target `json:"targets"`
	}{Targets: map[string]*models.Target{}}
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	m.all = state.Targets
	if state.Current != "" {
		m.current = m.all[state.Current]
	}
	return nil
}

func currentID(t *models.Target) string {
	if t != nil {
		return t.ID
	}
	return ""
}
