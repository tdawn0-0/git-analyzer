package git

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/config"
)

// LanguageFromPath returns a deterministic language label from a file path extension.
// Empty string means unknown / not counted toward ComplexityFactor languages.
func LanguageFromPath(path string) string {
	path = filepath.ToSlash(path)
	base := filepath.Base(path)
	if base == "Dockerfile" || strings.HasPrefix(base, "Dockerfile.") {
		return "Docker"
	}
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".rb":
		return "Ruby"
	case ".c", ".h":
		return "C"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh":
		return "C++"
	case ".cs":
		return "C#"
	case ".swift":
		return "Swift"
	case ".md", ".mdx":
		return "Markdown"
	case ".yml", ".yaml":
		return "YAML"
	case ".json":
		return "JSON"
	case ".css", ".scss", ".sass", ".less":
		return "CSS"
	case ".html", ".htm":
		return "HTML"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".sql":
		return "SQL"
	case ".php":
		return "PHP"
	case ".vue", ".svelte":
		return "Frontend"
	default:
		return ""
	}
}

// MatchModules returns module names whose path globs match the repo-relative path.
// Results are sorted for determinism.
func MatchModules(path string, modules map[string]config.ModuleDef) []string {
	path = filepath.ToSlash(path)
	var names []string
	for name, def := range modules {
		for _, pat := range def.Paths {
			if config.MatchPath(pat, path) {
				names = append(names, name)
				break
			}
		}
	}
	sort.Strings(names)
	return names
}

// LayersForModules collects unique non-empty layer labels from matched modules.
func LayersForModules(moduleNames []string, modules map[string]config.ModuleDef) []string {
	seen := map[string]struct{}{}
	var layers []string
	for _, name := range moduleNames {
		def, ok := modules[name]
		if !ok {
			continue
		}
		layer := strings.TrimSpace(def.Layer)
		if layer == "" {
			continue
		}
		if _, ok := seen[layer]; ok {
			continue
		}
		seen[layer] = struct{}{}
		layers = append(layers, layer)
	}
	sort.Strings(layers)
	return layers
}

// PrimaryModule picks the first matched module (sorted) for line attribution.
func PrimaryModule(moduleNames []string) string {
	if len(moduleNames) == 0 {
		return ""
	}
	return moduleNames[0]
}

// IsTestPath reports whether a path looks like a test file (QualityFactor / type hints).
func IsTestPath(path string) bool {
	path = filepath.ToSlash(path)
	return strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".test.ts") ||
		strings.HasSuffix(path, ".spec.ts") ||
		strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.HasSuffix(path, ".test.tsx") ||
		strings.HasSuffix(path, ".spec.tsx")
}

// IsSourceExt reports whether header scanning is worthwhile for this path.
func IsSourceExt(path string) bool {
	lang := LanguageFromPath(path)
	switch lang {
	case "Go", "TypeScript", "JavaScript", "Python", "Rust", "Java", "Kotlin",
		"Ruby", "C", "C++", "C#", "Swift", "PHP", "Frontend":
		return true
	default:
		return false
	}
}

// NormalizeNumstatPath extracts the destination path from rename/copy numstat paths.
func NormalizeNumstatPath(path string) string {
	path = filepath.ToSlash(path)
	// git may emit "old => new" or "{old => new}" forms with -M/-C; without those
	// flags paths are usually plain. Handle simple " => " split defensively.
	if i := strings.LastIndex(path, " => "); i >= 0 {
		path = strings.TrimSpace(path[i+4:])
		path = strings.TrimPrefix(path, "{")
		path = strings.TrimSuffix(path, "}")
	}
	return path
}
