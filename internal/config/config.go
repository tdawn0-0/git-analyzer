package config

import (
	"path/filepath"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// DefaultMaxDepth is the default filesystem recursion depth for repository discovery.
const DefaultMaxDepth = 6

// DefaultExcludeDirs are directory basenames skipped during discovery (SkipDir).
var DefaultExcludeDirs = []string{
	"node_modules",
	"vendor",
	"target",
	"dist",
	"build",
	".next",
	".cache",
	".idea",
	".vscode",
	"coverage",
	"tmp",
	"temp",
}

// ModuleDef maps path globs to a scoring weight and architectural layer.
type ModuleDef struct {
	Paths  []string `yaml:"paths"`
	Weight float64  `yaml:"weight"`
	Layer  string   `yaml:"layer"`
}

// RepoConfig is an optional per-repository override block.
type RepoConfig struct {
	Modules map[string]ModuleDef `yaml:"modules"`
}

// Config is the .workstats.yml surface used by metrics, discovery, and analysis.
type Config struct {
	Version      int                    `yaml:"version"`
	Workspace    WorkspaceConfig        `yaml:"workspace"`
	Analysis     AnalysisConfig         `yaml:"analysis"`
	Ignore       []string               `yaml:"ignore"`
	Generated    []string               `yaml:"generated"`
	Modules      map[string]ModuleDef   `yaml:"modules"`
	Repositories map[string]RepoConfig  `yaml:"repositories"`
	Types        map[string]float64     `yaml:"types"`
	Authors      map[string][]string    `yaml:"authors"`
}

// WorkspaceConfig controls repository discovery.
type WorkspaceConfig struct {
	MaxDepth int      `yaml:"max_depth"`
	Exclude  []string `yaml:"exclude"`
}

// AnalysisConfig selects the change unit (v1: commit).
type AnalysisConfig struct {
	ChangeUnit string `yaml:"change_unit"`
}

// DefaultTypeWeights returns Conventional Commit type weights from the design spec.
func DefaultTypeWeights() map[string]float64 {
	return map[string]float64{
		string(model.ChangeTypeFeat):     1.15,
		string(model.ChangeTypeFix):      1.10,
		string(model.ChangeTypeRefactor): 1.10,
		string(model.ChangeTypePerf):     1.15,
		string(model.ChangeTypeTest):     0.90,
		string(model.ChangeTypeDocs):     0.70,
		string(model.ChangeTypeBuild):    0.80,
		string(model.ChangeTypeCI):       0.80,
		string(model.ChangeTypeChore):    0.70,
		string(model.ChangeTypeStyle):    0.60,
		string(model.ChangeTypeRevert):   0.50,
		string(model.ChangeTypeUnknown):  1.00,
	}
}

// Defaults returns a Config populated with Phase 1 defaults.
func Defaults() Config {
	return Config{
		Version: 1,
		Workspace: WorkspaceConfig{
			MaxDepth: DefaultMaxDepth,
			Exclude:  append([]string(nil), DefaultExcludeDirs...),
		},
		Analysis: AnalysisConfig{ChangeUnit: "commit"},
		Ignore: []string{
			"**/node_modules/**",
			"**/dist/**",
			"**/target/**",
			"**/*.lock",
		},
		Generated: []string{
			"**/*.generated.ts",
			"**/generated/**",
		},
		Modules:      map[string]ModuleDef{},
		Repositories: map[string]RepoConfig{},
		Types:        DefaultTypeWeights(),
		Authors:      map[string][]string{},
	}
}

// TypeWeight returns the weight for a change type, falling back to defaults then 1.0.
func (c Config) TypeWeight(t model.ChangeType) float64 {
	key := string(t)
	if c.Types != nil {
		if w, ok := c.Types[key]; ok {
			return w
		}
	}
	if w, ok := DefaultTypeWeights()[key]; ok {
		return w
	}
	return 1.0
}

// ModulesForRepo returns global modules merged with repository-specific overrides.
// Repo modules replace same-named global entries; other globals remain.
func (c Config) ModulesForRepo(repoName string) map[string]ModuleDef {
	out := make(map[string]ModuleDef, len(c.Modules))
	for k, v := range c.Modules {
		out[k] = v
	}
	if c.Repositories != nil {
		if rc, ok := c.Repositories[repoName]; ok {
			for k, v := range rc.Modules {
				out[k] = v
			}
		}
	}
	return out
}

// ModuleWeights returns module name → weight for ModuleFactor from a module map.
func ModuleWeights(mods map[string]ModuleDef) map[string]float64 {
	out := make(map[string]float64, len(mods))
	for name, def := range mods {
		w := def.Weight
		if w == 0 {
			w = 1.0
		}
		out[name] = w
	}
	return out
}

// ModuleWeights returns module name → weight for ModuleFactor (global modules only).
func (c Config) ModuleWeights() map[string]float64 {
	return ModuleWeights(c.Modules)
}

// ResolveDeveloper maps a raw/mailmap identity to a canonical developer name.
// Priority: authors config (by email, then by name) > provided name.
func (c Config) ResolveDeveloper(name, email string) string {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if c.Authors != nil {
		for canon, aliases := range c.Authors {
			for _, a := range aliases {
				a = strings.TrimSpace(a)
				if a == "" {
					continue
				}
				if strings.EqualFold(a, email) || strings.EqualFold(a, name) {
					return canon
				}
				// Allow "Name <email>" style aliases.
				if strings.Contains(a, "<") && strings.Contains(a, ">") {
					start := strings.IndexByte(a, '<')
					end := strings.IndexByte(a, '>')
					if start >= 0 && end > start {
						inner := strings.TrimSpace(a[start+1 : end])
						if strings.EqualFold(inner, email) {
							return canon
						}
					}
				}
			}
			if strings.EqualFold(canon, name) {
				return canon
			}
		}
	}
	if name != "" {
		return name
	}
	if email != "" {
		return email
	}
	return "unknown"
}

// MatchPath reports whether repo-relative path matches a doublestar-ish glob.
func MatchPath(pattern, path string) bool {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	path = filepath.ToSlash(strings.TrimSpace(path))
	if pattern == "" || path == "" {
		return false
	}
	return matchDoublestar(pattern, path)
}

// PathExcluded reports whether path matches any ignore/generated style pattern.
func PathExcluded(path string, patterns []string) bool {
	for _, p := range patterns {
		if MatchPath(p, path) {
			return true
		}
	}
	return false
}

func matchDoublestar(pattern, path string) bool {
	// Fast path: filepath.Match when no **.
	if !strings.Contains(pattern, "**") {
		ok, err := filepath.Match(pattern, path)
		return err == nil && ok
	}
	return matchStars(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchStars(patParts, pathParts []string) bool {
	for len(patParts) > 0 {
		p := patParts[0]
		if p == "**" {
			if len(patParts) == 1 {
				return true
			}
			// Try consuming zero or more path segments.
			for i := 0; i <= len(pathParts); i++ {
				if matchStars(patParts[1:], pathParts[i:]) {
					return true
				}
			}
			return false
		}
		if len(pathParts) == 0 {
			return false
		}
		ok, err := filepath.Match(p, pathParts[0])
		if err != nil || !ok {
			return false
		}
		patParts = patParts[1:]
		pathParts = pathParts[1:]
	}
	return len(pathParts) == 0
}
