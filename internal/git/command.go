package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrGitNotFound is returned when the git binary is unavailable.
var ErrGitNotFound = errors.New("git executable not found")

// RunGit executes `git -C <dir> <args...>` with the given context.
func RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", ErrGitNotFound
	}
	all := make([]string, 0, len(args)+2)
	if dir != "" {
		all = append(all, "-C", dir)
	}
	all = append(all, args...)
	cmd := exec.CommandContext(ctx, "git", all...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// ShowTopLevel returns `git rev-parse --show-toplevel` for dir.
func ShowTopLevel(ctx context.Context, dir string) (string, error) {
	return RunGit(ctx, dir, "rev-parse", "--show-toplevel")
}

// RemoteOriginURL returns remote.origin.url or empty string when unset.
func RemoteOriginURL(ctx context.Context, dir string) string {
	out, err := RunGit(ctx, dir, "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}
	return out
}

// DefaultGitTimeout is used when callers pass a context without a deadline.
const DefaultGitTimeout = 30 * time.Second
