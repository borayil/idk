package shellinit

import (
	_ "embed"
	"fmt"
	"io"
)

//go:embed templates/zsh.sh
var zshTemplate string

//go:embed templates/bash.sh
var bashTemplate string

// Write emits the shell integration script for the given shell name to w. Supported shells
// in v1 are "zsh" and "bash"; anything else is an error.
func Write(w io.Writer, shell string) error {
	switch shell {
	case "zsh":
		_, err := io.WriteString(w, zshTemplate)
		return err
	case "bash":
		_, err := io.WriteString(w, bashTemplate)
		return err
	default:
		return fmt.Errorf("idk: unsupported shell %q (supported: zsh, bash)", shell)
	}
}
