package tty

import (
	"bufio"
	"fmt"
	"os"

	"golang.org/x/term"
)

// Item is one displayed suggestion. FromAI only changes the label, not the Command that runs.
type Item struct {
	Command string
	FromAI  bool
}

// Pick renders to /dev/tty (not stdout, so the caller can capture stdout separately) and
// blocks for one key sequence.
func Pick(items []Item) (Action, error) {
	ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return Action{}, fmt.Errorf("idk requires an interactive terminal: %w", err)
	}
	defer ttyFile.Close()

	fd := int(ttyFile.Fd())
	if !term.IsTerminal(fd) {
		return Action{}, fmt.Errorf("idk requires an interactive terminal")
	}

	for i, item := range items {
		label := item.Command
		if item.FromAI {
			label += "  (ai, unverified)"
		}
		fmt.Fprintf(ttyFile, "  %d) %s\n", i+1, label)
	}
	fmt.Fprint(ttyFile, "\nrun: 1-9   edit: e then 1-9   cancel: 0 or ctrl+c\n")

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return Action{}, err
	}
	defer term.Restore(fd, oldState)

	reader := bufio.NewReader(ttyFile)
	buf := make([]byte, 1)
	dec := NewDecoder()
	for {
		n, err := reader.Read(buf)
		if err != nil || n == 0 {
			return Action{Kind: ActionCancel}, nil
		}
		if act := dec.Feed(buf[0], len(items)); act.Kind != ActionNone {
			return act, nil
		}
	}
}
