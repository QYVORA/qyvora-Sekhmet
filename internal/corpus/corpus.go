// Package corpus manages the seed corpus: import, export, dedup, prioritize
// and archive seeds, each carrying rich metadata for scheduling. The corpus
// stores seed content on disk under controlled directories and keeps metadata
// in memory for fast scheduling on the hot path.
package corpus

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QYVORA/qyvora-sekhmet/pkg/models"
)

// Store holds the in-memory seed metadata and a content directory on disk.
type Store struct {
	mu     sync.RWMutex
	seeds  map[string]*models.Seed
	order  []string // insertion order for stable iteration
	dir    string
	weight float64
}

// New returns an empty corpus store backed by dir.
func New(dir string) *Store {
	return &Store{seeds: make(map[string]*models.Seed), dir: dir, weight: 1.0}
}

// SetWeight sets the novelty weight used by prioritization (default 1.0).
func (s *Store) SetWeight(w float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weight = w
}

// Dir returns the content directory.
func (s *Store) Dir() string { return s.dir }

// Add stores a seed, deduplicating by content hash: an identical payload is
// not stored twice, but existing coverage/novelty metadata is preserved.
func (s *Store) Add(seed *models.Seed) (*models.Seed, bool) {
	if seed == nil || len(seed.Data) == 0 {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	hash := hashBytes(seed.Data)
	if existing, ok := s.seeds[hash]; ok {
		// Already present; bump execution metadata only.
		return existing, false
	}
	if seed.ID == "" {
		seed.ID = models.NewSeedID()
	}
	seed.CreatedAt = time.Now().UTC()
	seed.Size = len(seed.Data)
	s.seeds[hash] = seed
	s.order = append(s.order, hash)
	return seed, true
}

// AddBytes is a convenience to add raw bytes with a creation source.
func (s *Store) AddBytes(data []byte, source string) (*models.Seed, bool) {
	return s.Add(&models.Seed{Source: source, Data: data})
}

// Remove deletes a seed by id, returning true if it existed.
func (s *Store) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for h, seed := range s.seeds {
		if seed.ID == id {
			delete(s.seeds, h)
			for i, o := range s.order {
				if o == h {
					s.order = append(s.order[:i], s.order[i+1:]...)
					break
				}
			}
			return true
		}
	}
	return false
}

// Get returns the seed whose id matches.
func (s *Store) Get(id string) (*models.Seed, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, seed := range s.seeds {
		if seed.ID == id {
			return seed, true
		}
	}
	return nil, false
}

// Size returns the number of unique seeds.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.seeds)
}

// List returns all seeds in insertion order.
func (s *Store) List() []*models.Seed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Seed, 0, len(s.order))
	for _, h := range s.order {
		if seed, ok := s.seeds[h]; ok {
			out = append(out, seed)
		}
	}
	return out
}

// Prioritized returns seeds sorted by descending scheduling priority. The
// ordering favors novelty, interestingness and small size so the hot path
// executes cheap, high-value seeds first.
func (s *Store) Prioritized() []*models.Seed {
	s.mu.RLock()
	sorted := s.ListUnderLock()
	w := s.weight
	s.mu.RUnlock()
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority(w) > sorted[j].Priority(w)
	})
	return sorted
}

// ListUnderLock returns a copy of the seed slice; caller must hold the lock.
func (s *Store) ListUnderLock() []*models.Seed {
	out := make([]*models.Seed, 0, len(s.seeds))
	for _, seed := range s.seeds {
		out = append(out, seed)
	}
	return out
}

// MarkExecuted updates bookkeeping for a seed after an execution.
func (s *Store) MarkExecuted(id string, novelty, coverageNovelty int, interesting float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, seed := range s.seeds {
		if seed.ID == id {
			seed.Executions++
			seed.Novelty += novelty
			seed.CoverageNovelty += coverageNovelty
			seed.Interestingness = interesting
			return
		}
	}
}

// TrimTo keeps at most n seeds (by priority), removing the lowest-value ones.
// Used to bound corpus growth during long campaigns.
func (s *Store) TrimTo(n int) []*models.Seed {
	if n <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.seeds) <= n {
		return nil
	}
	sorted := make([]*models.Seed, 0, len(s.seeds))
	for _, seed := range s.seeds {
		sorted = append(sorted, seed)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		pi := float64(sorted[i].Novelty)*s.weight + sorted[i].Interestingness
		pj := float64(sorted[j].Novelty)*s.weight + sorted[j].Interestingness
		return pi > pj
	})
	removed := sorted[n:]
	keep := make(map[*models.Seed]bool, n)
	for _, k := range sorted[:n] {
		keep[k] = true
	}
	newSeeds := make(map[string]*models.Seed, n)
	newOrder := make([]string, 0, n)
	for _, h := range s.order {
		if seed, ok := s.seeds[h]; ok && keep[seed] {
			newSeeds[h] = seed
			newOrder = append(newOrder, h)
		}
	}
	s.seeds = newSeeds
	s.order = newOrder
	return removed
}

// Persist writes all seed contents to the content directory for archival.
func (s *Store) Persist() error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for h, seed := range s.seeds {
		path := filepath.Join(s.dir, h+".seed")
		if err := os.WriteFile(path, seed.Data, 0o600); err != nil {
			return fmt.Errorf("persisting seed %s: %w", seed.ID, err)
		}
	}
	return nil
}

// ImportDir loads every file under dir as a seed with the given source label.
func (s *Store) ImportDir(dir, source string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if _, added := s.AddBytes(data, source); added {
			count++
		}
	}
	return count, nil
}

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// FilterByIter iterates seeds whose data contains the given marker.
func (s *Store) FilterByIter(fn func(*models.Seed) bool) []*models.Seed {
	var out []*models.Seed
	for _, seed := range s.List() {
		if fn(seed) {
			out = append(out, seed)
		}
	}
	return out
}

var _ = strings.TrimSpace // keep strings import if unused paths change
