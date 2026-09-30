package clipboard

import (
	"errors"
	"testing"
)

func TestCommandForOSDarwin(t *testing.T) {
	name, args, err := commandForOS("darwin", func(string) (string, error) { return "", errors.New("not found") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "pbcopy" || len(args) != 0 {
		t.Fatalf("got name=%q args=%v, want pbcopy with no args", name, args)
	}
}

func TestCommandForOSWindows(t *testing.T) {
	name, _, err := commandForOS("windows", func(string) (string, error) { return "", errors.New("not found") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "clip" {
		t.Fatalf("got name=%q, want clip", name)
	}
}

func TestCommandForOSLinuxPrefersXclip(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	}
	name, args, err := commandForOS("linux", lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "xclip" || len(args) != 2 {
		t.Fatalf("got name=%q args=%v, want xclip with 2 args", name, args)
	}
}

func TestCommandForOSLinuxFallsBackToWlCopy(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", errors.New("not found")
	}
	name, _, err := commandForOS("linux", lookPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "wl-copy" {
		t.Fatalf("got name=%q, want wl-copy", name)
	}
}

func TestCommandForOSLinuxNoToolFound(t *testing.T) {
	lookPath := func(string) (string, error) { return "", errors.New("not found") }
	_, _, err := commandForOS("linux", lookPath)
	if err == nil {
		t.Fatalf("expected error when no clipboard tool is found")
	}
}

func TestCommandForOSUnsupported(t *testing.T) {
	_, _, err := commandForOS("plan9", func(string) (string, error) { return "", errors.New("not found") })
	if err == nil {
		t.Fatalf("expected error for unsupported OS")
	}
}

func TestCopySucceedsOrReturnsUnavailable(t *testing.T) {
	err := Copy("hello from prompt-creator tests")
	if err != nil && !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected nil or ErrUnavailable, got: %v", err)
	}
}
