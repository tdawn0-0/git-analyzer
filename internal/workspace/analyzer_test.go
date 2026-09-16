package workspace_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/model"
	"github.com/tdawn0-0/git-analyzer/internal/workspace"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Dev One",
		"GIT_AUTHOR_EMAIL=dev@example.com",
		"GIT_COMMITTER_NAME=Dev One",
		"GIT_COMMITTER_EMAIL=dev@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, dir, "add", "README")
	gitRun(t, dir, "commit", "-m", "init")
}

func TestAnalyzerBranchIsolationAndIdentity(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	a := filepath.Join(root, "alpha")
	b := filepath.Join(root, "beta")
	initRepo(t, a)
	initRepo(t, b)
	gitRun(t, a, "branch", "shared")

	// Second identity email in alpha.
	if err := os.WriteFile(filepath.Join(a, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", a, "add", "main.go")
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}
	cmd = exec.Command("git", "-C", a, "commit", "-m", "feat: code")
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=Other",
		"GIT_AUTHOR_EMAIL=dev2@example.com",
		"GIT_COMMITTER_NAME=Other",
		"GIT_COMMITTER_EMAIL=dev2@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s", out)
	}

	cfg := config.Defaults()
	cfg.Authors = map[string][]string{
		"Yeu": {"dev@example.com", "dev2@example.com"},
	}

	an := workspace.NewAnalyzer()
	stats, results, err := an.Run(context.Background(), root, workspace.AnalyzeOptions{
		Branch: "shared",
		Jobs:   2,
		Config: cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results=%d", len(results))
	}
	var sawSkip, sawOK bool
	for _, r := range results {
		switch r.Status {
		case model.StatusSkipped:
			sawSkip = r.Message == "branch unavailable"
		case model.StatusComplete:
			sawOK = true
		}
	}
	if !sawSkip || !sawOK {
		t.Fatalf("isolation failed: %+v", results)
	}
	if len(stats.Developers) == 0 {
		t.Fatal("expected developers")
	}
	// Both emails should merge to Yeu for alpha's commits.
	foundYeu := false
	for _, d := range stats.Developers {
		if d.Developer == "Yeu" && d.ChangeCount >= 1 {
			foundYeu = true
		}
	}
	if !foundYeu {
		t.Fatalf("expected merged Yeu identity, got %+v", stats.Developers)
	}
}

func TestDefaultJobsBounded(t *testing.T) {
	j := workspace.DefaultJobs()
	if j < 1 || j > 8 {
		t.Fatalf("jobs=%d", j)
	}
}
