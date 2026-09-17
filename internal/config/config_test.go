package config

import (
	"os"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := Config{AIEnabled: true, AIProvider: "claude"}
	if err := Save(want); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if got != want {
		t.Errorf("got %+v want %+v", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got := Load()
	if got.AIEnabled || got.AIProvider != "" {
		t.Errorf("expected zero-value config, got %+v", got)
	}
}

func TestLoadTolerantOfMalformedLines(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := Path()
	if err := os.MkdirAll(dir+"/idk", 0o755); err != nil {
		t.Fatal(err)
	}
	content := "not a valid line\nai_enabled=true\n# comment\nai_provider=openai\n\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	got := Load()
	if !got.AIEnabled || got.AIProvider != "openai" {
		t.Errorf("got %+v", got)
	}
}
