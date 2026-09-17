package config_test

import (
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/config"
)

func TestMatchPathDoublestar(t *testing.T) {
	tests := []struct {
		pat, path string
		want      bool
	}{
		{"frontend/**", "frontend/app.ts", true},
		{"frontend/**", "backend/app.ts", false},
		{"**/migration/**", "db/migration/001.sql", true},
		{"**/*.lock", "foo/bar.lock", true},
		{"server/api/**", "server/api/x.go", true},
		{"*.go", "main.go", true},
		{"*.go", "pkg/main.go", false},
	}
	for _, tt := range tests {
		if got := config.MatchPath(tt.pat, tt.path); got != tt.want {
			t.Fatalf("MatchPath(%q,%q)=%v want %v", tt.pat, tt.path, got, tt.want)
		}
	}
}

func TestResolveDeveloper(t *testing.T) {
	cfg := config.Defaults()
	cfg.Authors = map[string][]string{
		"Yeu": {"dev@example.com", "dev2@example.com"},
	}
	if got := cfg.ResolveDeveloper("whatever", "dev2@example.com"); got != "Yeu" {
		t.Fatalf("%q", got)
	}
	if got := cfg.ResolveDeveloper("Alice", "alice@x.com"); got != "Alice" {
		t.Fatalf("%q", got)
	}
}

func TestModulesForRepoOverride(t *testing.T) {
	cfg := config.Defaults()
	cfg.Modules = map[string]config.ModuleDef{
		"api": {Paths: []string{"api/**"}, Weight: 1.0, Layer: "backend"},
	}
	cfg.Repositories = map[string]config.RepoConfig{
		"svc": {Modules: map[string]config.ModuleDef{
			"api": {Paths: []string{"svc/api/**"}, Weight: 1.2, Layer: "backend"},
			"db":  {Paths: []string{"**/migration/**"}, Weight: 1.3, Layer: "database"},
		}},
	}
	mods := cfg.ModulesForRepo("svc")
	if mods["api"].Weight != 1.2 || mods["api"].Paths[0] != "svc/api/**" {
		t.Fatalf("%+v", mods["api"])
	}
	if _, ok := mods["db"]; !ok {
		t.Fatal("missing db override add")
	}
}
