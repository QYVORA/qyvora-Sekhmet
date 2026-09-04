// Package session manages fuzzing campaign sessions: creation, persistence to
// disk, and loading. A session tracks the target, findings, crashes and
// campaign statistics so an operator (or the future orchestrator) can
// understand exactly what happened during a fuzz run.
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

var DefaultDir = filepath.Join(".", "sessions")

// Store persists sessions to disk as JSON with restrictive permissions.
type Store struct {
	dir string
}

// NewStore returns a store rooted at dir (or the default if empty).
func NewStore(dir string) *Store {
	if dir == "" {
		dir = DefaultDir
	}
	return &Store{dir: dir}
}

// Dir returns the store root directory.
func (s *Store) Dir() string { return s.dir }

// Save persists a session and returns the path written.
func (s *Store) Save(sess *models.Session) (string, error) {
	if sess == nil {
		return "", fmt.Errorf("session is nil")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("creating session dir: %w", err)
	}
	path := filepath.Join(s.dir, sess.ID+".session.json")
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads a session by id (auto-suffixed) or explicit path.
func (s *Store) Load(id string) (*models.Session, error) {
	path := id
	if !strings.HasSuffix(path, ".session.json") {
		path = filepath.Join(s.dir, id+".session.json")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sess models.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// List returns sorted session ids.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".session.json") {
			ids = append(ids, strings.TrimSuffix(name, ".session.json"))
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// Begin creates a new session bound to an optional target id.
func Begin(targetID string) *models.Session {
	return &models.Session{
		ID:       models.NewSessionID(),
		TargetID: targetID,
		Start:    time.Now().UTC(),
		State:    models.SessionRunning,
		Findings: []*models.Finding{},
		Crashes:  []*models.Finding{},
		Errors:   []string{},
	}
}
