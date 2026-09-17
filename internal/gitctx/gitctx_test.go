package gitctx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePorcelain(t *testing.T) {
	lines := []string{" M internal/rank/rank.go", "?? newfile.txt", ""}
	got := parsePorcelain(lines)
	want := map[string]bool{"internal/rank/rank.go": true, "newfile.txt": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, f := range got {
		if !want[f] {
			t.Errorf("unexpected file %q", f)
		}
	}
}

func TestContextKeywordsGoExtension(t *testing.T) {
	ctx := Context{ChangedFiles: []string{"main.go"}}
	kws := ContextKeywords(ctx)
	if !contains(kws, "go build") {
		t.Errorf("expected go build keyword, got %v", kws)
	}
}

func TestContextKeywordsPackageJSON(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	kws := ContextKeywords(Context{})
	if !contains(kws, "npm") {
		t.Errorf("expected npm keyword, got %v", kws)
	}
}

func TestContextKeywordsNoFileBasedSignals(t *testing.T) {
	// gcloud's signal depends on the host's ambient config, not the cwd, so it's excluded
	// here — this only asserts the file/extension-based signals stay silent in an empty dir.
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	kws := ContextKeywords(Context{})
	for _, unwanted := range []string{"go build", "npm", "pip", "terraform"} {
		if contains(kws, unwanted) {
			t.Errorf("expected no %q keyword in an empty dir, got %v", unwanted, kws)
		}
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
