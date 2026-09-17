package metrics

import (
	"path/filepath"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// ScoreInput is everything needed to compute an explainable ChangeScore.
type ScoreInput struct {
	AddedLines   int
	DeletedLines int

	ChangedFiles   int
	ChangedModules int
	Languages      int
	Layers         int

	ModuleLines map[string]int
	Weights     map[string]float64 // module name → weight

	ChangeType  model.ChangeType
	TypeWeights map[string]float64

	HasTests bool
	IsRevert bool
}

// Compute returns a full ScoreBreakdown from ScoreInput.
func Compute(in ScoreInput) model.ScoreBreakdown {
	size := SizeScore(in.AddedLines, in.DeletedLines)
	complexity := ComplexityFactor(in.ChangedFiles, in.ChangedModules, in.Languages, in.Layers)
	module := ModuleFactor(in.ModuleLines, in.Weights)
	typ := TypeFactor(string(in.ChangeType), in.TypeWeights)
	quality := QualityFactor(in.HasTests, in.IsRevert)
	return model.ScoreBreakdown{
		SizeScore:        size,
		ComplexityFactor: complexity,
		ModuleFactor:     module,
		TypeFactor:       typ,
		QualityFactor:    quality,
		Final:            ChangeScore(size, complexity, module, typ, quality),
	}
}

// ParseChangeType extracts a Conventional Commit type from a subject line.
// Supports "type:", "type(scope):", and "type!:" / "type(scope)!:".
func ParseChangeType(message string) model.ChangeType {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return model.ChangeTypeUnknown
	}
	// First line only.
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	msg = strings.TrimSpace(msg)

	lower := strings.ToLower(msg)
	if strings.HasPrefix(lower, "revert") {
		return model.ChangeTypeRevert
	}

	// type(scope)!: subject  OR  type!: subject  OR  type: subject
	i := 0
	for i < len(msg) {
		c := msg[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			i++
			continue
		}
		break
	}
	if i == 0 {
		return model.ChangeTypeUnknown
	}
	typ := strings.ToLower(msg[:i])
	rest := msg[i:]

	if strings.HasPrefix(rest, "(") {
		close := strings.IndexByte(rest, ')')
		if close < 0 {
			return model.ChangeTypeUnknown
		}
		rest = rest[close+1:]
	}
	if strings.HasPrefix(rest, "!") {
		rest = rest[1:]
	}
	if !strings.HasPrefix(rest, ":") {
		return model.ChangeTypeUnknown
	}

	switch model.ChangeType(typ) {
	case model.ChangeTypeFeat, model.ChangeTypeFix, model.ChangeTypeRefactor,
		model.ChangeTypePerf, model.ChangeTypeTest, model.ChangeTypeDocs,
		model.ChangeTypeBuild, model.ChangeTypeCI, model.ChangeTypeChore,
		model.ChangeTypeStyle, model.ChangeTypeRevert:
		return model.ChangeType(typ)
	default:
		return model.ChangeTypeUnknown
	}
}

// InferChangeType applies path heuristics when Conventional Commits are absent.
func InferChangeType(message string, paths []string) model.ChangeType {
	if t := ParseChangeType(message); t != model.ChangeTypeUnknown {
		return t
	}
	if len(paths) == 0 {
		return model.ChangeTypeUnknown
	}
	allTest, allDocs, allCI := true, true, true
	for _, p := range paths {
		p = filepath.ToSlash(p)
		base := filepath.Base(p)
		if !(strings.HasSuffix(p, "_test.go") ||
			strings.HasSuffix(p, ".test.ts") ||
			strings.HasSuffix(p, ".spec.ts") ||
			strings.HasSuffix(p, ".test.js") ||
			strings.HasSuffix(p, ".spec.js")) {
			allTest = false
		}
		if !(strings.HasPrefix(p, "docs/") || strings.Contains(p, "/docs/") ||
			strings.HasPrefix(strings.ToUpper(base), "README")) {
			allDocs = false
		}
		if !(strings.HasPrefix(p, ".github/") || strings.Contains(p, "/.github/") ||
			base == ".gitlab-ci.yml" || strings.HasSuffix(p, "/.gitlab-ci.yml")) {
			allCI = false
		}
	}
	switch {
	case allTest:
		return model.ChangeTypeTest
	case allDocs:
		return model.ChangeTypeDocs
	case allCI:
		return model.ChangeTypeCI
	default:
		return model.ChangeTypeUnknown
	}
}
