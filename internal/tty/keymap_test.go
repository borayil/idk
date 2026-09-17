package tty

import "testing"

func TestFeedPlainDigitRuns(t *testing.T) {
	cases := []struct {
		b       byte
		n       int
		wantIdx int
	}{
		{'1', 5, 0},
		{'5', 5, 4},
		{'9', 9, 8},
	}
	for _, c := range cases {
		got := NewDecoder().Feed(c.b, c.n)
		if got.Kind != ActionRun || got.Index != c.wantIdx {
			t.Errorf("Feed(%q, %d) = %+v, want Run index %d", c.b, c.n, got, c.wantIdx)
		}
	}
}

func TestFeedOutOfRangeDigitIgnored(t *testing.T) {
	got := NewDecoder().Feed('6', 5)
	if got.Kind != ActionNone {
		t.Errorf("expected ActionNone for out-of-range digit, got %+v", got)
	}
}

func TestFeedUnrelatedByteIgnored(t *testing.T) {
	got := NewDecoder().Feed('z', 5)
	if got.Kind != ActionNone {
		t.Errorf("expected ActionNone, got %+v", got)
	}
}

func TestFeedCtrlCCancels(t *testing.T) {
	got := NewDecoder().Feed(0x03, 5)
	if got.Kind != ActionCancel {
		t.Errorf("expected Cancel, got %+v", got)
	}
}

func TestFeedZeroCancels(t *testing.T) {
	got := NewDecoder().Feed('0', 5)
	if got.Kind != ActionCancel {
		t.Errorf("expected Cancel, got %+v", got)
	}
}

func TestFeedEditSequence(t *testing.T) {
	d := NewDecoder()
	first := d.Feed('e', 5)
	if first.Kind != ActionNone {
		t.Fatalf("expected 'e' alone to be pending (ActionNone), got %+v", first)
	}
	second := d.Feed('3', 5)
	if second.Kind != ActionEdit || second.Index != 2 {
		t.Errorf("expected Edit index 2 after e3, got %+v", second)
	}
}

func TestFeedUppercaseEAlsoStartsEdit(t *testing.T) {
	d := NewDecoder()
	d.Feed('E', 5)
	got := d.Feed('1', 5)
	if got.Kind != ActionEdit || got.Index != 0 {
		t.Errorf("expected Edit index 0 after E1, got %+v", got)
	}
}

func TestFeedEditThenOutOfRangeDropsSilently(t *testing.T) {
	d := NewDecoder()
	d.Feed('e', 3)
	got := d.Feed('9', 3) // 9 is out of range when only 3 items shown
	if got.Kind != ActionNone {
		t.Errorf("expected ActionNone (dropped pending edit), got %+v", got)
	}
	// decoder should have reset, so a plain digit now runs normally
	got2 := d.Feed('1', 3)
	if got2.Kind != ActionRun || got2.Index != 0 {
		t.Errorf("expected decoder reset to normal Run behavior, got %+v", got2)
	}
}

func TestFeedEditThenCtrlCCancels(t *testing.T) {
	d := NewDecoder()
	d.Feed('e', 5)
	got := d.Feed(0x03, 5)
	if got.Kind != ActionCancel {
		t.Errorf("expected Cancel after e+ctrl-c, got %+v", got)
	}
}

func TestFeedEditThenZeroCancels(t *testing.T) {
	d := NewDecoder()
	d.Feed('e', 5)
	got := d.Feed('0', 5)
	if got.Kind != ActionCancel {
		t.Errorf("expected Cancel after e0, got %+v", got)
	}
}

func TestFeedEditThenUnrelatedByteResets(t *testing.T) {
	d := NewDecoder()
	d.Feed('e', 5)
	got := d.Feed('z', 5)
	if got.Kind != ActionNone {
		t.Errorf("expected ActionNone, got %+v", got)
	}
	got2 := d.Feed('2', 5)
	if got2.Kind != ActionRun || got2.Index != 1 {
		t.Errorf("expected decoder reset to normal Run behavior, got %+v", got2)
	}
}
