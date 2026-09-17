package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"idk/internal/aiprovider"
	"idk/internal/config"
	"idk/internal/gitctx"
	"idk/internal/history"
	"idk/internal/rank"
	"idk/internal/shellinit"
	"idk/internal/tty"
)

// aiTimeout caps how much latency the opt-in AI supplement can add.
const aiTimeout = 1500 * time.Millisecond

const maxAISuggestions = 2

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInit()
			return
		case "enable":
			runEnable(os.Args[2:])
			return
		case "disable":
			runDisable(os.Args[2:])
			return
		}
	}

	n := resolveN()
	cfg := config.Load()

	var wg sync.WaitGroup
	var globalEntries, cwdEntries []history.Entry
	var ctx gitctx.Context

	wg.Add(3)
	go func() { defer wg.Done(); globalEntries = loadGlobalHistory() }()
	go func() { defer wg.Done(); cwdEntries = loadCwdHistory() }()
	go func() { defer wg.Done(); ctx = gitctx.Gather() }()
	wg.Wait()

	keywords := gitctx.ContextKeywords(ctx)
	items := rank.BuildItems(globalEntries, cwdEntries, keywords, time.Now())
	heuristicRanked := rank.Rank(items, n)

	aiCmds := fetchAISuggestions(cfg, heuristicRanked, ctx)
	displayItems := buildDisplayItems(heuristicRanked, aiCmds, n)

	if len(displayItems) == 0 {
		fmt.Fprintln(os.Stderr, "idk: no history-based suggestions available")
		fmt.Println("CANCEL")
		return
	}

	action, err := tty.Pick(displayItems)
	if err != nil {
		fmt.Fprintln(os.Stderr, "idk:", err)
		fmt.Println("ERROR")
		return
	}

	switch action.Kind {
	case tty.ActionRun:
		fmt.Printf("RUN\t%s\n", displayItems[action.Index].Command)
	case tty.ActionEdit:
		fmt.Printf("EDIT\t%s\n", displayItems[action.Index].Command)
	default:
		fmt.Println("CANCEL")
	}
}

// buildDisplayItems reserves the last few slots for AI picks, rest stay heuristic, capped at n.
func buildDisplayItems(heuristicRanked []rank.Item, aiCmds []string, n int) []tty.Item {
	keep := len(heuristicRanked)
	if len(aiCmds) > 0 {
		keep = n - len(aiCmds)
		if keep < 1 {
			keep = 1
		}
		if keep > len(heuristicRanked) {
			keep = len(heuristicRanked)
		}
	}

	items := make([]tty.Item, 0, keep+len(aiCmds))
	for _, it := range heuristicRanked[:keep] {
		items = append(items, tty.Item{Command: it.Command})
	}
	for _, c := range aiCmds {
		items = append(items, tty.Item{Command: c, FromAI: true})
	}
	return items
}

// fetchAISuggestions asks the configured provider for extra commands. Any failure just
// logs to stderr and falls back to heuristic-only — never a hard dependency.
func fetchAISuggestions(cfg config.Config, heuristicRanked []rank.Item, gctx gitctx.Context) []string {
	if !cfg.AIEnabled || cfg.AIProvider == "" {
		return nil
	}
	provider, err := aiprovider.Resolve(cfg.AIProvider)
	if err != nil {
		fmt.Fprintln(os.Stderr, "idk:", err)
		return nil
	}
	if os.Getenv(provider.APIKeyEnv()) == "" {
		fmt.Fprintf(os.Stderr, "idk: %s not set, skipping ai suggestions\n", provider.APIKeyEnv())
		return nil
	}

	recent := make([]string, 0, len(heuristicRanked))
	for _, it := range heuristicRanked {
		recent = append(recent, it.Command)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), aiTimeout)
	defer cancel()
	suggestions, err := provider.Suggest(ctxTimeout, aiprovider.Request{
		Branch:         gctx.Branch,
		ChangedFiles:   gctx.ChangedFiles,
		RecentCommands: recent,
		MaxSuggestions: maxAISuggestions,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "idk: ai suggestion skipped:", err)
		return nil
	}

	seen := make(map[string]bool, len(recent))
	for _, c := range recent {
		seen[c] = true
	}
	out := make([]string, 0, len(suggestions))
	for _, s := range suggestions {
		if !seen[s] {
			out = append(out, s)
			seen[s] = true
		}
	}
	if len(out) > maxAISuggestions {
		out = out[:maxAISuggestions]
	}
	return out
}

func runInit() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: idk-bin init <zsh|bash>")
		os.Exit(1)
	}
	if err := shellinit.Write(os.Stdout, os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runEnable(args []string) {
	if len(args) < 1 || args[0] != "ai" {
		fmt.Fprintln(os.Stderr, "usage: idk-bin enable ai [claude|openai]")
		os.Exit(1)
	}
	var provider aiprovider.Provider
	var err error
	if len(args) >= 2 {
		provider, err = aiprovider.Resolve(args[1])
	} else {
		provider, err = aiprovider.AutoDetect(os.Getenv)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "idk:", err)
		os.Exit(1)
	}
	if os.Getenv(provider.APIKeyEnv()) == "" {
		fmt.Fprintf(os.Stderr, "idk: %s is not set; export it before enabling AI suggestions\n", provider.APIKeyEnv())
		os.Exit(1)
	}
	if err := config.Save(config.Config{AIEnabled: true, AIProvider: provider.Name()}); err != nil {
		fmt.Fprintln(os.Stderr, "idk:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "idk: ai suggestions on (%s), up to %d extra picks per run, labeled \"(ai, unverified)\"\n",
		provider.Name(), maxAISuggestions)
}

func runDisable(args []string) {
	if len(args) < 1 || args[0] != "ai" {
		fmt.Fprintln(os.Stderr, "usage: idk-bin disable ai")
		os.Exit(1)
	}
	if err := config.Save(config.Config{AIEnabled: false}); err != nil {
		fmt.Fprintln(os.Stderr, "idk:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "idk: ai suggestions off")
}

// -n flag > IDK_N env > default 5, clamped to [3,9]
func resolveN() int {
	n := 5
	if v := os.Getenv("IDK_N"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			n = parsed
		}
	}
	for i, a := range os.Args {
		if a == "-n" && i+1 < len(os.Args) {
			if parsed, err := strconv.Atoi(os.Args[i+1]); err == nil {
				n = parsed
			}
		} else if strings.HasPrefix(a, "-n=") {
			if parsed, err := strconv.Atoi(strings.TrimPrefix(a, "-n=")); err == nil {
				n = parsed
			}
		}
	}
	return rank.Clamp(n)
}

func loadGlobalHistory() []history.Entry {
	lines, err := history.TailLines(history.HistoryFilePath(), 5000)
	if err != nil {
		return nil
	}
	if history.IsZsh() {
		return history.ParseZshHistory(lines)
	}
	return history.ParseBashHistory(lines)
}

func loadCwdHistory() []history.Entry {
	lines, err := history.TailLines(history.LogFilePath(), 5000)
	if err != nil {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	logEntries := history.ParseIdkLog(lines)
	out := make([]history.Entry, 0, len(logEntries))
	for _, e := range logEntries {
		if e.Cwd == cwd {
			out = append(out, history.Entry{Command: e.Command, Epoch: e.Epoch})
		}
	}
	return out
}
