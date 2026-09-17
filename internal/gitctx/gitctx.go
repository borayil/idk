package gitctx

import (
	"os"
	"os/exec"
	"strings"
)

// Context holds the git/project signals used to bias command ranking.
type Context struct {
	InRepo       bool
	ChangedFiles []string
	Branch       string
}

// Gather collects git status/diff/branch info, or a zero-value Context outside a repo.
func Gather() Context {
	if !inRepo() {
		return Context{}
	}
	ctx := Context{InRepo: true}

	filesSet := map[string]struct{}{}
	for _, f := range parsePorcelain(runGitLines("status", "--porcelain")) {
		filesSet[f] = struct{}{}
	}
	for _, f := range runGitLines("diff", "--name-only") {
		if f != "" {
			filesSet[f] = struct{}{}
		}
	}
	for _, f := range runGitLines("diff", "--cached", "--name-only") {
		if f != "" {
			filesSet[f] = struct{}{}
		}
	}
	for f := range filesSet {
		ctx.ChangedFiles = append(ctx.ChangedFiles, f)
	}

	if branchLines := runGitLines("rev-parse", "--abbrev-ref", "HEAD"); len(branchLines) > 0 {
		ctx.Branch = branchLines[0]
	}
	return ctx
}

func inRepo() bool {
	out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func runGitLines(args ...string) []string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil
	}
	trimmed := strings.TrimRight(string(out), "\n")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// parsePorcelain pulls file paths out of "XY path" porcelain lines.
func parsePorcelain(lines []string) []string {
	var files []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		parts := strings.Fields(l)
		if len(parts) >= 2 {
			files = append(files, parts[len(parts)-1])
		}
	}
	return files
}

// signal boosts a fixed set of keywords when its condition matches. No plugin system, just this.
type signal struct {
	match    func(ctx Context) bool
	keywords []string
}

var signals = []signal{
	{
		match:    func(ctx Context) bool { return hasExt(ctx, ".go") || fileExists("go.mod") },
		keywords: []string{"go build", "go test", "go vet", "go run"},
	},
	{
		match:    func(ctx Context) bool { return fileExists("package.json") },
		keywords: []string{"npm", "yarn", "pnpm"},
	},
	{
		match:    func(ctx Context) bool { return fileExists("requirements.txt") || fileExists("pyproject.toml") },
		keywords: []string{"pip", "python", "pytest"},
	},
	{
		match:    func(ctx Context) bool { return hasExt(ctx, ".tf") },
		keywords: []string{"terraform"},
	},
	{
		match:    func(ctx Context) bool { return gcpSignalPresent() },
		keywords: []string{"gcloud"},
	},
}

// ContextKeywords returns the keyword boosts that apply to the current repo/directory.
func ContextKeywords(ctx Context) []string {
	var kws []string
	for _, s := range signals {
		if s.match(ctx) {
			kws = append(kws, s.keywords...)
		}
	}
	return kws
}

func hasExt(ctx Context, ext string) bool {
	for _, f := range ctx.ChangedFiles {
		if strings.HasSuffix(f, ext) {
			return true
		}
	}
	return false
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

// gcpSignalPresent checks cheap, subprocess-free signals only — shelling out to `gcloud` itself
// was tried and dropped, its cold start alone can take seconds.
func gcpSignalPresent() bool {
	for _, envVar := range []string{"GOOGLE_CLOUD_PROJECT", "GCLOUD_PROJECT", "CLOUDSDK_CORE_PROJECT"} {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	for _, marker := range []string{"app.yaml", "cloudbuild.yaml", ".gcloudignore"} {
		if fileExists(marker) {
			return true
		}
	}
	return false
}
