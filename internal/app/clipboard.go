package app

import (
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard in a background goroutine.
func (a *App) CopyToClipboard(text string) {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "linux":
		candidates = [][]string{
			{"wl-copy"},
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
		}
	case "windows":
		candidates = [][]string{{"clip"}}
	}

	for _, args := range candidates {
		bin, err := exec.LookPath(args[0])
		if err != nil {
			continue
		}
		go func(bin string, cliArgs []string) {
			cmd := exec.Command(bin, cliArgs...)
			cmd.Stdout = nil
			cmd.Stderr = nil

			stdin, err := cmd.StdinPipe()
			if err != nil {
				return
			}

			if err := cmd.Start(); err != nil {
				return
			}

			// Write full text and close stdin to signal EOF
			_, _ = io.Copy(stdin, strings.NewReader(text))
			stdin.Close()

			_ = cmd.Wait()
		}(bin, args[1:])
		return
	}
}
