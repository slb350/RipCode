package store

import (
	"math"
	"sort"
	"time"
)

const frecencyFile = "frecency.json"

// maxFrecencyAge is the age beyond which entries are pruned on load.
const maxFrecencyAge = 30 * 24 * time.Hour

// frecencyHalfLife controls how quickly recency weight decays.
// At 7 days since last use, the recency weight is 0.5.
const frecencyHalfLife = 7.0

// scoreEpsilon is the threshold for treating two frecency scores as equal.
const scoreEpsilon = 1e-9

// FileFrecency tracks file access patterns for frecency-based ranking.
type FileFrecency struct {
	Entries map[string]FrecencyEntry `json:"entries"`
}

// FrecencyEntry records usage count and recency for a single file.
type FrecencyEntry struct {
	Count    int       `json:"count"`
	LastUsed time.Time `json:"last_used"`
}

// LoadFrecency loads from ~/.ripcode/state/frecency.json.
// Auto-prunes entries older than 30 days on load.
func LoadFrecency() (*FileFrecency, error) {
	f, err := loadState[FileFrecency](frecencyFile, "frecency")
	if err != nil {
		return f, err
	}
	if f.Entries == nil {
		f.Entries = make(map[string]FrecencyEntry)
	}
	f.Prune(maxFrecencyAge)
	return f, nil
}

// Save persists to disk.
func (f *FileFrecency) Save() error {
	return saveState(frecencyFile, f)
}

// Record tracks a file access, incrementing count and updating LastUsed.
func (f *FileFrecency) Record(path string) {
	if f.Entries == nil {
		f.Entries = make(map[string]FrecencyEntry)
	}
	e := f.Entries[path]
	e.Count++
	e.LastUsed = time.Now()
	f.Entries[path] = e
}

// Score computes a frecency score for a path.
// Score = count * recencyWeight, where recencyWeight = 1.0 / (1.0 + daysSinceLastUse / halfLife).
// Returns 0 for unknown paths.
func (f *FileFrecency) Score(path string) float64 {
	e, ok := f.Entries[path]
	if !ok {
		return 0
	}
	return f.scoreEntry(e)
}

func (f *FileFrecency) scoreEntry(e FrecencyEntry) float64 {
	days := time.Since(e.LastUsed).Hours() / 24.0
	if days < 0 {
		days = 0
	}
	recencyWeight := 1.0 / (1.0 + days/frecencyHalfLife)
	return float64(e.Count) * recencyWeight
}

// Rank sorts paths by frecency score (highest first).
// Unscored paths are appended in their original order.
func (f *FileFrecency) Rank(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	type scored struct {
		path  string
		score float64
		idx   int // original index for stable sort
	}

	var scored1 []scored
	var unscored []string

	for i, p := range paths {
		s := f.Score(p)
		if s > 0 {
			scored1 = append(scored1, scored{path: p, score: s, idx: i})
		} else {
			unscored = append(unscored, p)
		}
	}

	sort.Slice(scored1, func(i, j int) bool {
		if math.Abs(scored1[i].score-scored1[j].score) > scoreEpsilon {
			return scored1[i].score > scored1[j].score
		}
		return scored1[i].idx < scored1[j].idx
	})

	result := make([]string, 0, len(paths))
	for _, s := range scored1 {
		result = append(result, s.path)
	}
	result = append(result, unscored...)
	return result
}

// Prune removes entries older than maxAge. Returns count of pruned entries.
func (f *FileFrecency) Prune(maxAge time.Duration) int {
	if f.Entries == nil {
		return 0
	}
	cutoff := time.Now().Add(-maxAge)
	pruned := 0
	for path, e := range f.Entries {
		if e.LastUsed.Before(cutoff) {
			delete(f.Entries, path)
			pruned++
		}
	}
	return pruned
}
