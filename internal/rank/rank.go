package rank

import (
	"math"
	"sort"
	"strings"
	"time"

	"idk/internal/history"
)

// Item is a deduplicated, scored candidate command.
type Item struct {
	Command          string
	GlobalCount      int
	GlobalAgeSeconds float64
	CwdCount         int
	CwdAgeSeconds    float64
	ContextMatches   int
	score            float64
}

const (
	wFreq    = 1.0
	wRecency = 2.0
	wCwd     = 3.0
	wContext = 1.5

	maxContextMatches = 4

	halfLifeGlobal = 14 * 24 * 3600.0
	halfLifeCwd    = 7 * 24 * 3600.0

	// no timestamp -> treat as ancient so recency term ~0, freq/context still count
	unknownAgeSeconds = halfLifeGlobal * 10
)

// Denylist excludes noisy commands nobody wants suggested.
var Denylist = map[string]bool{
	"ls": true, "ll": true, "pwd": true, "clear": true, "exit": true,
	"history": true, "cd": true, "cd ..": true, "cd ~": true, "cd -": true,
	"": true,
}

// Score blends recency, frequency, cwd-affinity, and context-match into one number.
func Score(it Item) float64 {
	cm := it.ContextMatches
	if cm > maxContextMatches {
		cm = maxContextMatches
	}
	s := wFreq * math.Log1p(float64(it.GlobalCount))
	s += wRecency * math.Pow(0.5, it.GlobalAgeSeconds/halfLifeGlobal)
	s += wCwd * math.Log1p(float64(it.CwdCount)) * math.Pow(0.5, it.CwdAgeSeconds/halfLifeCwd)
	s += wContext * float64(cm)
	return s
}

// BuildItems dedupes + scores raw history entries, applying the denylist along the way.
func BuildItems(globalEntries []history.Entry, cwdEntries []history.Entry, contextKeywords []string, now time.Time) []Item {
	type agg struct {
		globalCount      int
		globalMostRecent int64
		cwdCount         int
		cwdMostRecent    int64
	}
	m := map[string]*agg{}

	add := func(cmd string, epoch int64, isCwd bool) {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" || Denylist[cmd] {
			return
		}
		a := m[cmd]
		if a == nil {
			a = &agg{}
			m[cmd] = a
		}
		if isCwd {
			a.cwdCount++
			if epoch > a.cwdMostRecent {
				a.cwdMostRecent = epoch
			}
		} else {
			a.globalCount++
			if epoch > a.globalMostRecent {
				a.globalMostRecent = epoch
			}
		}
	}

	for _, e := range globalEntries {
		add(e.Command, e.Epoch, false)
	}
	for _, e := range cwdEntries {
		add(e.Command, e.Epoch, true)
	}

	nowEpoch := now.Unix()
	items := make([]Item, 0, len(m))
	for cmd, a := range m {
		it := Item{
			Command:        cmd,
			GlobalCount:    a.globalCount,
			CwdCount:       a.cwdCount,
			ContextMatches: countMatches(cmd, contextKeywords),
		}
		if a.globalMostRecent > 0 {
			it.GlobalAgeSeconds = clampNonNegative(float64(nowEpoch - a.globalMostRecent))
		} else {
			it.GlobalAgeSeconds = unknownAgeSeconds
		}
		if a.cwdMostRecent > 0 {
			it.CwdAgeSeconds = clampNonNegative(float64(nowEpoch - a.cwdMostRecent))
		} else {
			it.CwdAgeSeconds = unknownAgeSeconds
		}
		it.score = Score(it)
		items = append(items, it)
	}
	return items
}

func clampNonNegative(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

func countMatches(cmd string, keywords []string) int {
	n := 0
	for _, k := range keywords {
		if strings.Contains(cmd, k) {
			n++
		}
	}
	return n
}

// Rank sorts by score desc and clamps to n items (n itself clamped into [3,9]).
func Rank(items []Item, n int) []Item {
	n = Clamp(n)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		if rk := recencyKey(items[i]) - recencyKey(items[j]); rk != 0 {
			return rk > 0
		}
		// map iteration order is random, so fully-tied items need a stable final tie-break
		return items[i].Command < items[j].Command
	})
	if len(items) > n {
		items = items[:n]
	}
	return items
}

func recencyKey(it Item) float64 {
	return -it.GlobalAgeSeconds - it.CwdAgeSeconds
}

// Clamp forces n into the supported suggestion-count range [3,9].
func Clamp(n int) int {
	if n < 3 {
		return 3
	}
	if n > 9 {
		return 9
	}
	return n
}
