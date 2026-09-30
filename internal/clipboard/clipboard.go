package clipboard

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ErrUnavailable indicates the clipboard could not be reached. Callers
// should treat this as non-fatal — the prompt has already been printed.
var ErrUnavailable = errors.New("clipboard unavailable")

// commandForOS returns the executable name and args used to copy stdin to
// the system clipboard on the given GOOS, or an error if no suitable tool
// is available. lookPath is injected so tests can simulate any OS/PATH
// combination without real clipboard binaries.
func commandForOS(goos string, lookPath func(string) (string, error)) (string, []string, error) {
	switch goos {
	case "darwin":
		return "pbcopy", nil, nil
	case "windows":
		return "clip", nil, nil
	case "linux":
		if _, err := lookPath("xclip"); err == nil {
			return "xclip", []string{"-selection", "clipboard"}, nil
		}
		if _, err := lookPath("wl-copy"); err == nil {
			return "wl-copy", nil, nil
		}
		return "", nil, errors.New("no clipboard utility found (tried xclip, wl-copy)")
	default:
		return "", nil, fmt.Errorf("unsupported OS %q", goos)
	}
}

// Copy writes text to the system clipboard using an OS-appropriate external
// command. Any failure to find or run that command is wrapped in
// ErrUnavailable so callers can treat it as a warning, not a fatal error.
func Copy(text string) error {
	name, args, err := commandForOS(runtime.GOOS, exec.LookPath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: running %s: %v", ErrUnavailable, name, err)
	}
	return nil
}
