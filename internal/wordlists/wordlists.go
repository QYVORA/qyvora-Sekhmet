// Package wordlists provides SecLists integration. It resolves a SecLists
// installation (from QYVORA_SEKHMET_SECLISTS_DIR, a config path, or the
// conventional ~/tools/SecLists locations) and offers listing and targeted
// search without blind-vendoring the ~5GB corpus. Loading only materializes
// the entries needed for a given search so memory stays bounded.
package wordlists

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SecListsDir returns the resolved SecLists directory, or "" if none found.
func SecListsDir(envOverride string) string {
	if envOverride != "" {
		if fi, err := os.Stat(envOverride); err == nil && fi.IsDir() {
			return envOverride
		}
	}
	candidates := []string{
		os.Getenv("QYVORA_SEKHMET_SECLISTS_DIR"),
		os.Getenv("SECLISTS"),
		filepath.Join(os.Getenv("HOME"), "tools", "SecLists"),
		filepath.Join(os.Getenv("HOME"), "SecLists"),
		"/usr/share/seclists",
		"/opt/SecLists",
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return ""
}

// Category is a directory-relative grouping under SecLists root.
type Category struct {
	RelPath string `json:"rel_path"`
	Files   int    `json:"files"`
}

// Categories lists the top-level categories (one level deep) under the
// SecLists installation. Files are counted so the user can gauge the size of
// each category before loading it (no blind vendoring).
func Categories(root string) ([]Category, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []Category
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := e.Name()
		seen[rel] = true
		files := countFiles(filepath.Join(root, rel))
		out = append(out, Category{RelPath: rel, Files: files})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out, nil
}

// SearchResult is a single matching file path with an estimated entry count.
type SearchResult struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Estimate int    `json:"estimate"`
}

// Search looks for wordlist files whose name contains any of the keywords
// (case-insensitive). It limits the depth so the whole tree is not traversed.
func Search(root string, keywords []string, maxDepth int) ([]SearchResult, error) {
	var results []SearchResult
	err := walk(root, root, keywords, maxDepth, &results)
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
}

func walk(base, dir string, keywords []string, depth int, results *[]SearchResult) error {
	if depth < 0 {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			if err := walk(base, filepath.Join(dir, e.Name()), keywords, depth-1, results); err != nil {
				return err
			}
			continue
		}
		name := strings.ToLower(e.Name())
		for _, kw := range keywords {
			if strings.Contains(name, strings.ToLower(kw)) {
				rel, _ := filepath.Rel(base, filepath.Join(dir, e.Name()))
				*results = append(*results, SearchResult{
					Path:     rel,
					Name:     e.Name(),
					Estimate: estimateLines(filepath.Join(dir, e.Name())),
				})
				break
			}
		}
	}
	return nil
}

func countFiles(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}

func estimateLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	buf := make([]byte, 1024*1024)
	total, first := 0, 0
	newlines := 0
	for {
		n, err := f.Read(buf)
		if err != nil {
			break
		}
		total += n
		if first == 0 {
			for _, b := range buf[:n] {
				if b == '\n' {
					newlines++
				}
			}
			first = n
		}
	}
	if first == 0 {
		return 0
	}
	linesPerByte := float64(newlines) / float64(first)
	return int(float64(total) * linesPerByte)
}

// Load reads the entries of a specific wordlist file (resolved via Search or
// a direct path). It returns them as a slice of strings capped at maxEntries.
func Load(root, relOrPath string, maxEntries int) ([]string, error) {
	path := relOrPath
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, relOrPath)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	if maxEntries > 0 && len(lines) > maxEntries {
		lines = lines[:maxEntries]
	}
	return lines, nil
}
