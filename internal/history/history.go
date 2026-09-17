package history

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Entry is a single historical command occurrence.
type Entry struct {
	Command string
	Epoch   int64 // unix seconds; 0 means unknown
}

// LogEntry is a single entry from idk's own cwd-aware log file.
type LogEntry struct {
	Epoch   int64
	Cwd     string
	Command string
}

// HistoryFilePath locates the current shell's history file via $HISTFILE, falling back to
// ~/.zsh_history or ~/.bash_history based on $SHELL.
func HistoryFilePath() string {
	if h := os.Getenv("HISTFILE"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	if IsZsh() {
		return filepath.Join(home, ".zsh_history")
	}
	return filepath.Join(home, ".bash_history")
}

// IsZsh reports whether $SHELL points at zsh.
func IsZsh() bool {
	return strings.Contains(os.Getenv("SHELL"), "zsh")
}

// LogFilePath is idk's own cwd-aware append log, under $XDG_DATA_HOME (or ~/.local/share).
func LogFilePath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "idk", "history.log")
}

// TailLines returns at most maxLines of the last lines in the file at path, in original order.
func TailLines(path string, maxLines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := make([]string, maxLines)
	count := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		buf[count%maxLines] = sc.Text()
		count++
	}

	n := count
	if n > maxLines {
		n = maxLines
	}
	start := 0
	if count > maxLines {
		start = count % maxLines
	}
	result := make([]string, 0, n)
	for i := 0; i < n; i++ {
		result = append(result, buf[(start+i)%maxLines])
	}
	return result, nil
}

var zshExtendedRe = regexp.MustCompile(`(?s)^: (\d+):(\d+);(.*)$`)

// ParseZshHistory handles both EXTENDED_HISTORY (": <epoch>:<dur>;<cmd>") and plain lines,
// plus backslash continuations for multi-line commands.
func ParseZshHistory(rawLines []string) []Entry {
	logical := joinBackslashContinuations(rawLines)
	entries := make([]Entry, 0, len(logical))
	for _, line := range logical {
		if line == "" {
			continue
		}
		if m := zshExtendedRe.FindStringSubmatch(line); m != nil {
			epoch, _ := strconv.ParseInt(m[1], 10, 64)
			cmd := strings.TrimSpace(m[3])
			if cmd != "" {
				entries = append(entries, Entry{Command: cmd, Epoch: epoch})
			}
			continue
		}
		cmd := strings.TrimSpace(line)
		if cmd != "" {
			entries = append(entries, Entry{Command: cmd})
		}
	}
	return entries
}

// joinBackslashContinuations merges zsh's trailing-backslash multi-line commands into one line.
func joinBackslashContinuations(lines []string) []string {
	var out []string
	var cur strings.Builder
	inCont := false
	for _, l := range lines {
		if inCont {
			cur.WriteString("\n")
			cur.WriteString(strings.TrimSuffix(l, "\\"))
			if !strings.HasSuffix(l, "\\") {
				out = append(out, cur.String())
				cur.Reset()
				inCont = false
			}
			continue
		}
		if strings.HasSuffix(l, "\\") {
			cur.WriteString(strings.TrimSuffix(l, "\\"))
			inCont = true
			continue
		}
		out = append(out, l)
	}
	if inCont {
		out = append(out, cur.String())
	}
	return out
}

var bashTimestampRe = regexp.MustCompile(`^#(\d+)$`)

// ParseBashHistory handles plain lines, plus optional "#<epoch>" HISTTIMEFORMAT timestamps.
func ParseBashHistory(rawLines []string) []Entry {
	entries := make([]Entry, 0, len(rawLines))
	var pendingEpoch int64
	havePending := false
	for _, line := range rawLines {
		if m := bashTimestampRe.FindStringSubmatch(line); m != nil {
			pendingEpoch, _ = strconv.ParseInt(m[1], 10, 64)
			havePending = true
			continue
		}
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			havePending = false
			continue
		}
		e := Entry{Command: cmd}
		if havePending {
			e.Epoch = pendingEpoch
		}
		entries = append(entries, e)
		havePending = false
	}
	return entries
}

// ParseIdkLog reads "<epoch>\t<cwd>\t<command>" lines, skipping anything malformed.
func ParseIdkLog(rawLines []string) []LogEntry {
	out := make([]LogEntry, 0, len(rawLines))
	for _, line := range rawLines {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		epoch, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}
		cmd := strings.TrimSpace(parts[2])
		if cmd == "" {
			continue
		}
		out = append(out, LogEntry{Epoch: epoch, Cwd: parts[1], Command: cmd})
	}
	return out
}
