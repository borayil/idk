package rank

import (
	"testing"
	"time"

	"idk/internal/history"
)

func TestRecencyOrdering(t *testing.T) {
	now := time.Now()
	entries := []history.Entry{
		{Command: "old cmd", Epoch: now.Add(-30 * 24 * time.Hour).Unix()},
		{Command: "recent cmd", Epoch: now.Add(-1 * time.Hour).Unix()},
	}
	items := BuildItems(entries, nil, nil, now)
	ranked := Rank(items, 3)
	if len(ranked) < 2 {
		t.Fatalf("expected 2, got %d", len(ranked))
	}
	if ranked[0].Command != "recent cmd" {
		t.Errorf("expected recent cmd first, got %+v", ranked)
	}
}

func TestFrequencyOrdering(t *testing.T) {
	now := time.Now()
	var entries []history.Entry
	for i := 0; i < 5; i++ {
		entries = append(entries, history.Entry{Command: "frequent", Epoch: now.Unix()})
	}
	entries = append(entries, history.Entry{Command: "rare", Epoch: now.Unix()})
	items := BuildItems(entries, nil, nil, now)
	ranked := Rank(items, 3)
	if ranked[0].Command != "frequent" {
		t.Errorf("expected frequent first, got %+v", ranked)
	}
}

func TestCwdBoostOutranksEqualGlobalFrequency(t *testing.T) {
	now := time.Now()
	entries := []history.Entry{
		{Command: "cmd-a", Epoch: now.Unix()},
		{Command: "cmd-b", Epoch: now.Unix()},
	}
	cwdEntries := []history.Entry{
		{Command: "cmd-b", Epoch: now.Unix()},
	}
	items := BuildItems(entries, cwdEntries, nil, now)
	ranked := Rank(items, 3)
	if ranked[0].Command != "cmd-b" {
		t.Errorf("expected cmd-b (cwd-boosted) first, got %+v", ranked)
	}
}

func TestContextBoostOutranks(t *testing.T) {
	now := time.Now()
	entries := []history.Entry{
		{Command: "go build ./...", Epoch: now.Unix()},
		{Command: "unrelated thing", Epoch: now.Unix()},
	}
	items := BuildItems(entries, nil, []string{"go build"}, now)
	ranked := Rank(items, 3)
	if ranked[0].Command != "go build ./..." {
		t.Errorf("expected context-boosted command first, got %+v", ranked)
	}
}

func TestDenylistExcluded(t *testing.T) {
	now := time.Now()
	entries := []history.Entry{{Command: "ls", Epoch: now.Unix()}}
	items := BuildItems(entries, nil, nil, now)
	if len(items) != 0 {
		t.Errorf("expected ls to be denylisted, got %+v", items)
	}
}

func TestDedup(t *testing.T) {
	now := time.Now()
	entries := []history.Entry{
		{Command: "git status", Epoch: now.Unix()},
		{Command: "git status", Epoch: now.Unix()},
		{Command: "git status", Epoch: now.Unix()},
	}
	items := BuildItems(entries, nil, nil, now)
	if len(items) != 1 {
		t.Fatalf("expected dedup to 1 item, got %d: %+v", len(items), items)
	}
	if items[0].GlobalCount != 3 {
		t.Errorf("expected count 3, got %d", items[0].GlobalCount)
	}
}

func TestClamp(t *testing.T) {
	cases := map[int]int{0: 3, 1: 3, 2: 3, 3: 3, 5: 5, 9: 9, 10: 9, 100: 9, -5: 3}
	for in, want := range cases {
		if got := Clamp(in); got != want {
			t.Errorf("Clamp(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestRankClampsResultCount(t *testing.T) {
	now := time.Now()
	var entries []history.Entry
	for i := 0; i < 20; i++ {
		entries = append(entries, history.Entry{Command: "cmd" + string(rune('a'+i)), Epoch: now.Unix()})
	}
	items := BuildItems(entries, nil, nil, now)
	ranked := Rank(items, 100)
	if len(ranked) != 9 {
		t.Errorf("expected clamp to 9, got %d", len(ranked))
	}
	ranked = Rank(items, 0)
	if len(ranked) != 3 {
		t.Errorf("expected clamp to 3, got %d", len(ranked))
	}
}

func BenchmarkBuildAndRank(b *testing.B) {
	now := time.Now()
	var entries []history.Entry
	for i := 0; i < 5000; i++ {
		entries = append(entries, history.Entry{Command: "cmd " + string(rune('a'+i%26)), Epoch: now.Unix()})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		items := BuildItems(entries, nil, nil, now)
		Rank(items, 5)
	}
}
