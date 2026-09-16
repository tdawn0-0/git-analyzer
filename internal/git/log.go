package git

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const commitMarker = "===COMMIT==="

// RawCommit is one commit as emitted by git log --numstat before scoring.
type RawCommit struct {
	Hash    string
	Author  string
	Email   string
	Date    time.Time
	Subject string
	Files   []RawFileStat
}

// RawFileStat is one numstat line.
type RawFileStat struct {
	Path    string
	Added   int // -1 means binary / unparseable ("-")
	Deleted int
	Binary  bool
}

// AnalyzeOptions controls git log filtering for one repository.
type AnalyzeOptions struct {
	Since  string
	Until  string
	Author string // passed to git log --author when set (raw filter)
	Branch string // empty = HEAD
}

// LogCommits runs a single git log --numstat --format pass and parses commits.
func LogCommits(ctx context.Context, repoPath string, opts AnalyzeOptions) ([]RawCommit, error) {
	args := []string{
		"log",
		"--use-mailmap",
		"--date=iso-strict",
		"--numstat",
		"--format=" + commitMarker + "%n%H%n%an%n%ae%n%aI%n%s",
	}
	if opts.Since != "" {
		args = append(args, "--since="+opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until="+opts.Until)
	}
	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}

	out, err := RunGit(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return ParseLogOutput(out)
}

// ParseLogOutput parses the custom commitMarker + numstat stream.
func ParseLogOutput(out string) ([]RawCommit, error) {
	sc := bufio.NewScanner(strings.NewReader(out))
	// Commits with huge subjects / paths are rare; raise limit anyway.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024)

	var commits []RawCommit
	var cur *RawCommit
	metaLeft := 0

	flush := func() {
		if cur != nil {
			commits = append(commits, *cur)
			cur = nil
		}
	}

	for sc.Scan() {
		line := sc.Text()
		if line == commitMarker {
			flush()
			cur = &RawCommit{}
			metaLeft = 5
			continue
		}
		if cur == nil {
			continue
		}
		if metaLeft > 0 {
			switch metaLeft {
			case 5:
				cur.Hash = strings.TrimSpace(line)
			case 4:
				cur.Author = line
			case 3:
				cur.Email = strings.TrimSpace(line)
			case 2:
				t, err := time.Parse(time.RFC3339, strings.TrimSpace(line))
				if err != nil {
					// Fallback: try without timezone seconds variants.
					t, err = time.Parse("2006-01-02T15:04:05-07:00", strings.TrimSpace(line))
					if err != nil {
						t = time.Time{}
					}
				}
				cur.Date = t
			case 1:
				cur.Subject = line
			}
			metaLeft--
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fs, ok := parseNumstatLine(line)
		if ok {
			cur.Files = append(cur.Files, fs)
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan git log: %w", err)
	}
	return commits, nil
}

func parseNumstatLine(line string) (RawFileStat, bool) {
	// format: <added>\t<deleted>\t<path>
	// renames may appear as "old => new" in path with -M; we keep raw path text.
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return RawFileStat{}, false
	}
	fs := RawFileStat{Path: parts[2]}
	if parts[0] == "-" || parts[1] == "-" {
		fs.Binary = true
		fs.Added = 0
		fs.Deleted = 0
		return fs, true
	}
	a, err1 := strconv.Atoi(parts[0])
	d, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return RawFileStat{}, false
	}
	fs.Added = a
	fs.Deleted = d
	return fs, true
}

// BranchExists reports whether ref exists in the repository (local branch or commit-ish).
func BranchExists(ctx context.Context, repoPath, branch string) bool {
	if branch == "" {
		return true
	}
	_, err := RunGit(ctx, repoPath, "rev-parse", "--verify", "--quiet", branch)
	return err == nil
}

// ShowFilePrefix returns up to maxBytes of file content at commit:path.
func ShowFilePrefix(ctx context.Context, repoPath, commit, path string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 2048
	}
	spec := commit + ":" + path
	out, err := RunGit(ctx, repoPath, "show", spec)
	if err != nil {
		return "", err
	}
	if len(out) > maxBytes {
		return out[:maxBytes], nil
	}
	return out, nil
}
