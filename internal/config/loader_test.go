package config_test

import (
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func TestParseMergesTypeDefaults(t *testing.T) {
	yaml := []byte(`
version: 1
types:
  feat: 1.2
workspace:
  max_depth: 3
`)
	cfg, err := config.Parse(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Workspace.MaxDepth != 3 {
		t.Fatalf("max_depth=%d", cfg.Workspace.MaxDepth)
	}
	if cfg.TypeWeight(model.ChangeTypeFeat) != 1.2 {
		t.Fatalf("feat weight=%v", cfg.TypeWeight(model.ChangeTypeFeat))
	}
	if cfg.TypeWeight(model.ChangeTypeFix) != 1.10 {
		t.Fatalf("fix default missing: %v", cfg.TypeWeight(model.ChangeTypeFix))
	}
}
