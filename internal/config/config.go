package config

import "github.com/tdawn0-0/git-analyzer/internal/model"

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

// Config is the Phase 1 subset of .workstats.yml needed by metrics and discovery.
type Config struct {
	Version   int                    `yaml:"version"`
	Workspace WorkspaceConfig        `yaml:"workspace"`
	Analysis  AnalysisConfig         `yaml:"analysis"`
	Ignore    []string               `yaml:"ignore"`
	Generated []string               `yaml:"generated"`
	Modules   map[string]ModuleDef   `yaml:"modules"`
	Types     map[string]float64     `yaml:"types"`
	Authors   map[string][]string    `yaml:"authors"`
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
		Modules: map[string]ModuleDef{},
		Types:   DefaultTypeWeights(),
		Authors: map[string][]string{},
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

// ModuleWeights returns module name → weight for ModuleFactor.
func (c Config) ModuleWeights() map[string]float64 {
	out := make(map[string]float64, len(c.Modules))
	for name, def := range c.Modules {
		w := def.Weight
		if w == 0 {
			w = 1.0
		}
		out[name] = w
	}
	return out
}
