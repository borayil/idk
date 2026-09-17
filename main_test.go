package main

import (
	"testing"

	"idk/internal/rank"
)

func heuristicItems(cmds ...string) []rank.Item {
	items := make([]rank.Item, len(cmds))
	for i, c := range cmds {
		items[i] = rank.Item{Command: c}
	}
	return items
}

func TestBuildDisplayItemsNoAI(t *testing.T) {
	got := buildDisplayItems(heuristicItems("a", "b", "c"), nil, 5)
	if len(got) != 3 {
		t.Fatalf("expected all 3 heuristic items kept, got %d: %+v", len(got), got)
	}
	for _, it := range got {
		if it.FromAI {
			t.Errorf("expected no AI items, got %+v", it)
		}
	}
}

func TestBuildDisplayItemsReservesSlots(t *testing.T) {
	heuristic := heuristicItems("a", "b", "c", "d", "e")
	got := buildDisplayItems(heuristic, []string{"ai-cmd-1", "ai-cmd-2"}, 5)
	if len(got) != 5 {
		t.Fatalf("expected 5 total (3 heuristic + 2 ai), got %d: %+v", len(got), got)
	}
	wantHeuristic := []string{"a", "b", "c"}
	for i, w := range wantHeuristic {
		if got[i].Command != w || got[i].FromAI {
			t.Errorf("index %d: got %+v, want heuristic %q", i, got[i], w)
		}
	}
	if got[3].Command != "ai-cmd-1" || !got[3].FromAI {
		t.Errorf("expected ai-cmd-1 labeled FromAI, got %+v", got[3])
	}
	if got[4].Command != "ai-cmd-2" || !got[4].FromAI {
		t.Errorf("expected ai-cmd-2 labeled FromAI, got %+v", got[4])
	}
}

func TestBuildDisplayItemsAlwaysKeepsAtLeastOneHeuristic(t *testing.T) {
	heuristic := heuristicItems("only-one")
	got := buildDisplayItems(heuristic, []string{"ai-1", "ai-2"}, 3)
	if len(got) < 1 || got[0].Command != "only-one" || got[0].FromAI {
		t.Fatalf("expected the single heuristic item preserved first, got %+v", got)
	}
}

func TestBuildDisplayItemsEmptyInputs(t *testing.T) {
	got := buildDisplayItems(nil, nil, 5)
	if len(got) != 0 {
		t.Errorf("expected empty result, got %+v", got)
	}
}
