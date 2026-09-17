package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseZshHistoryExtended(t *testing.T) {
	lines := []string{": 1690000000:0;git status", ": 1690000100:0;go build ./..."}
	entries := ParseZshHistory(lines)
	if len(entries) != 2 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Command != "git status" || entries[0].Epoch != 1690000000 {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].Command != "go build ./..." || entries[1].Epoch != 1690000100 {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestParseZshHistoryPlain(t *testing.T) {
	lines := []string{"git status", "go build ./..."}
	entries := ParseZshHistory(lines)
	if len(entries) != 2 {
		t.Fatalf("got %d entries", len(entries))
	}
	if entries[1].Command != "go build ./..." {
		t.Errorf("unexpected: %+v", entries[1])
	}
}

func TestParseZshHistoryContinuation(t *testing.T) {
	lines := []string{
		`: 1690000000:0;echo hello \`,
		`&& echo world`,
	}
	entries := ParseZshHistory(lines)
	if len(entries) != 1 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	want := "echo hello \n&& echo world"
	if entries[0].Command != want {
		t.Errorf("got %q want %q", entries[0].Command, want)
	}
	if entries[0].Epoch != 1690000000 {
		t.Errorf("expected epoch preserved, got %+v", entries[0])
	}
}

func TestParseBashHistoryPlain(t *testing.T) {
	lines := []string{"git status", "go build ./..."}
	entries := ParseBashHistory(lines)
	if len(entries) != 2 {
		t.Fatalf("got %d", len(entries))
	}
}

func TestParseBashHistoryTimestamped(t *testing.T) {
	lines := []string{"#1690000000", "git status", "go build ./..."}
	entries := ParseBashHistory(lines)
	if len(entries) != 2 {
		t.Fatalf("got %d", len(entries))
	}
	if entries[0].Epoch != 1690000000 {
		t.Errorf("expected epoch, got %+v", entries[0])
	}
	if entries[1].Epoch != 0 {
		t.Errorf("expected no epoch for second entry, got %+v", entries[1])
	}
}

func TestParseIdkLogTolerant(t *testing.T) {
	lines := []string{
		"1690000000\t/home/x\tgit status",
		"garbage line",
		"notanumber\t/home/x\tgit status",
		"",
	}
	entries := ParseIdkLog(lines)
	if len(entries) != 1 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Cwd != "/home/x" || entries[0].Command != "git status" {
		t.Errorf("unexpected entry: %+v", entries[0])
	}
}

func TestTailLinesBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hist")
	content := ""
	for i := 0; i < 10; i++ {
		content += string(rune('a'+i)) + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, err := TailLines(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"h", "i", "j"}
	if len(lines) != 3 {
		t.Fatalf("got %v", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("got %v want %v", lines, want)
		}
	}
}

func TestTailLinesFewerThanMax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hist")
	if err := os.WriteFile(path, []byte("a\nb\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, err := TailLines(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("got %v", lines)
	}
}
