package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/git"
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
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
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

func TestScanNormalRepo(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	repo := filepath.Join(root, "app")
	initRepo(t, repo)

	scanner := git.NewRepositoryScanner()
	found, err := scanner.Scan(context.Background(), root, git.ScanOptions{MaxDepth: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want 1 repo, got %d: %+v", len(found), found)
	}
	if found[0].Name != "app" {
		t.Fatalf("name=%q", found[0].Name)
	}
	if filepath.Clean(found[0].Path) != filepath.Clean(repo) {
		t.Fatalf("path=%q want %q", found[0].Path, repo)
	}
	if found[0].ID == "" {
		t.Fatal("empty ID")
	}
}

func TestScanNestedRepos(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	parent := filepath.Join(root, "monorepo")
	nested := filepath.Join(parent, "examples", "demo")
	initRepo(t, parent)
	initRepo(t, nested)

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{MaxDepth: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("want 2 repos (parent+nested), got %d: %+v", len(found), found)
	}
	names := map[string]bool{}
	for _, r := range found {
		names[r.Name] = true
	}
	if !names["monorepo"] || !names["demo"] {
		t.Fatalf("unexpected names: %v", names)
	}
}

func TestScanWorktreeGitFile(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	mainRepo := filepath.Join(root, "main")
	initRepo(t, mainRepo)

	wt := filepath.Join(root, "wt-feature")
	gitRun(t, mainRepo, "worktree", "add", "-b", "feature", wt)

	info, err := os.Lstat(filepath.Join(wt, ".git"))
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("expected worktree .git to be a file")
	}

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{MaxDepth: 6})
	if err != nil {
		t.Fatal(err)
	}
	var sawWT, sawMain bool
	for _, r := range found {
		base := filepath.Base(r.Path)
		if base == "wt-feature" {
			sawWT = true
		}
		if base == "main" {
			sawMain = true
		}
	}
	if !sawMain || !sawWT {
		t.Fatalf("want main+worktree, got %+v", found)
	}
}

func TestScanExcludedDirs(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	visible := filepath.Join(root, "service")
	hidden := filepath.Join(root, "node_modules", "pkg")
	vendor := filepath.Join(root, "vendor", "lib")
	initRepo(t, visible)
	initRepo(t, hidden)
	initRepo(t, vendor)

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{
		MaxDepth: 6,
		Exclude:  config.DefaultExcludeDirs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "service" {
		t.Fatalf("want only service, got %+v", found)
	}
}

func TestScanMaxDepth(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	deep := filepath.Join(root, "d1", "d2", "d3", "deeprepo")
	shallow := filepath.Join(root, "shallow")
	initRepo(t, deep)
	initRepo(t, shallow)

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{
		MaxDepth: 2,
		Exclude:  []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Name != "shallow" {
		t.Fatalf("max-depth should hide deeprepo, got %+v", found)
	}

	found, err = git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{
		MaxDepth: 4,
		Exclude:  []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("want 2 at depth 4, got %+v", found)
	}
}

func TestScanSymlinkNotFollowedByDefault(t *testing.T) {
	requireGit(t)
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	root := t.TempDir()
	realRepo := filepath.Join(root, "real")
	initRepo(t, realRepo)

	other := filepath.Join(root, "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(other, "link-to-real")
	if err := os.Symlink(realRepo, link); err != nil {
		t.Fatal(err)
	}

	found, err := git.NewRepositoryScanner().Scan(context.Background(), other, git.ScanOptions{
		MaxDepth:      6,
		Exclude:       []string{},
		FollowSymlink: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("must not follow symlink, got %+v", found)
	}

	found, err = git.NewRepositoryScanner().Scan(context.Background(), other, git.ScanOptions{
		MaxDepth:      6,
		Exclude:       []string{},
		FollowSymlink: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("follow-symlinks should find repo, got %+v", found)
	}
}

func TestScanDuplicateNames(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	a := filepath.Join(root, "tools", "foo")
	b := filepath.Join(root, "apps", "foo")
	initRepo(t, a)
	initRepo(t, b)

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{
		MaxDepth: 6,
		Exclude:  []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("want 2, got %+v", found)
	}
	for _, r := range found {
		if r.Name != "foo" {
			t.Fatalf("both should be named foo, got %q", r.Name)
		}
	}
	display := git.DisambiguateNames(found, root)
	vals := map[string]bool{}
	for _, d := range display {
		vals[d] = true
	}
	if len(vals) != 2 {
		t.Fatalf("expected disambiguated display names, got %v", display)
	}
}

func TestScanRootIsRepo(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	initRepo(t, root)

	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{MaxDepth: 6})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("want 1, got %+v", found)
	}
}

func TestRepositoryIDUsesRemote(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	initRepo(t, root)
	gitRun(t, root, "remote", "add", "origin", "https://example.com/org/repo.git")

	repo, err := git.NewRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if repo.ID != "https://example.com/org/repo.git" {
		t.Fatalf("ID=%q", repo.ID)
	}
	if repo.Remote != repo.ID {
		t.Fatalf("Remote=%q", repo.Remote)
	}
}

func TestDefaultMaxDepth(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	initRepo(t, filepath.Join(root, "a"))
	found, err := git.NewRepositoryScanner().Scan(context.Background(), root, git.ScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("got %+v", found)
	}
}
