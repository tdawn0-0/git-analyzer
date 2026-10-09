package metrics_test

import (
	"math"
	"testing"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/metrics"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func almostEqual(t *testing.T, got, want, eps float64) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Fatalf("got %v want %v (eps %v)", got, want, eps)
	}
}

func TestSizeScore(t *testing.T) {
	tests := []struct {
		name          string
		added, deleted int
		want          float64
	}{
		{"zero", 0, 0, 0}, // log2(1)=0
		{"one", 1, 0, 1},
		{"ten", 5, 5, math.Log2(11)},
		{"hundred", 60, 40, math.Log2(101)},
		{"thousand", 700, 300, math.Log2(1001)},
		{"ten_thousand", 8000, 2000, math.Log2(10001)},
		{"hundred_thousand_capped", 80000, 20000, 14}, // log2(100001) > 14
		{"negative_treated_as_zero_parts", -5, 0, 0},   // changed clamped via <0 → 0 in SizeScore only if sum < 0
		{"spec_500", 320, 180, math.Log2(501)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.SizeScore(tt.added, tt.deleted)
			almostEqual(t, got, tt.want, 1e-9)
			if got > metrics.SizeScoreMax {
				t.Fatalf("SizeScore exceeded cap: %v", got)
			}
		})
	}
}

func TestComplexityFactor(t *testing.T) {
	tests := []struct {
		name                         string
		files, modules, langs, layers int
		want                         float64
	}{
		{"baseline", 0, 0, 0, 0, 1.0},
		{"one_file_one_module", 1, 1, 1, 1, 1.0 + 0.015},
		{"spec_example", 12, 3, 2, 2, 1.51}, // 1+0.18+0.20+0.05+0.08
		{"files_capped_at_20", 100, 1, 1, 1, 1.0 + 20*0.015},
		{"modules_extra_capped_at_6", 0, 100, 1, 1, 1.0 + 6*0.10},
		{"three_languages", 0, 1, 3, 1, 1.10},
		{"layers_extra_capped", 0, 1, 1, 10, 1.0 + 3*0.08},
		{"hits_max_clamp", 20, 7, 5, 5, 2.0}, // 1+0.3+0.6+0.1+0.24=2.24 → 2.0
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.ComplexityFactor(tt.files, tt.modules, tt.langs, tt.layers)
			almostEqual(t, got, tt.want, 1e-9)
			if got < metrics.ComplexityFactorMin || got > metrics.ComplexityFactorMax {
				t.Fatalf("ComplexityFactor out of range: %v", got)
			}
		})
	}
}

func TestComplexitySubFactors(t *testing.T) {
	almostEqual(t, metrics.FilesFactor(12), 0.18, 1e-9)
	almostEqual(t, metrics.ModulesFactor(3), 0.20, 1e-9)
	almostEqual(t, metrics.LanguageFactor(2), 0.05, 1e-9)
	almostEqual(t, metrics.LayerFactor(2), 0.08, 1e-9)
}

func TestModuleFactor(t *testing.T) {
	weights := map[string]float64{
		"api":      1.1,
		"database": 1.3,
		"frontend": 1.0,
	}
	tests := []struct {
		name  string
		lines map[string]int
		want  float64
	}{
		{"empty", nil, 1.0},
		{"zero_lines", map[string]int{"api": 0}, 1.0},
		{"single_known", map[string]int{"api": 100}, 1.1},
		{"unknown_defaults_to_one", map[string]int{"orphan": 50}, 1.0},
		{"multi_weighted", map[string]int{"api": 100, "database": 100}, (100*1.1 + 100*1.3) / 200},
		{"clamped_high", map[string]int{"database": 10}, 1.3}, // within [0.8,1.5]
		{"mixed_unknown", map[string]int{"frontend": 50, "misc": 50}, (50*1.0 + 50*1.0) / 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.ModuleFactor(tt.lines, weights)
			almostEqual(t, got, tt.want, 1e-9)
			if got < metrics.ModuleFactorMin || got > metrics.ModuleFactorMax {
				t.Fatalf("ModuleFactor out of range: %v", got)
			}
		})
	}

	t.Run("extreme_weight_clamped", func(t *testing.T) {
		got := metrics.ModuleFactor(map[string]int{"x": 10}, map[string]float64{"x": 9.0})
		almostEqual(t, got, metrics.ModuleFactorMax, 1e-9)
		got = metrics.ModuleFactor(map[string]int{"x": 10}, map[string]float64{"x": 0.1})
		almostEqual(t, got, metrics.ModuleFactorMin, 1e-9)
	})
}

func TestTypeFactor(t *testing.T) {
	weights := config.DefaultTypeWeights()
	tests := []struct {
		typ  string
		want float64
	}{
		{"feat", 1.15},
		{"fix", 1.10},
		{"refactor", 1.10},
		{"perf", 1.15},
		{"test", 0.90},
		{"docs", 0.70},
		{"build", 0.80},
		{"ci", 0.80},
		{"chore", 0.70},
		{"style", 0.60},
		{"revert", 0.50}, // below stated 0.6 guidance; table wins
		{"unknown", 1.00},
		{"not-a-type", 1.00}, // missing → 1.0
	}
	for _, tt := range tests {
		t.Run(tt.typ, func(t *testing.T) {
			got := metrics.TypeFactor(tt.typ, weights)
			almostEqual(t, got, tt.want, 1e-9)
		})
	}
	t.Run("nil_weights", func(t *testing.T) {
		almostEqual(t, metrics.TypeFactor("feat", nil), 1.0, 1e-9)
	})
}

func TestQualityFactor(t *testing.T) {
	tests := []struct {
		name               string
		hasTests, isRevert bool
		want               float64
	}{
		{"default", false, false, 1.0},
		{"tests", true, false, 1.05},
		{"revert", false, true, 0.5},
		{"tests_and_revert", true, true, 0.525}, // 1.05 * 0.5
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metrics.QualityFactor(tt.hasTests, tt.isRevert)
			almostEqual(t, got, tt.want, 1e-9)
			if got < metrics.QualityFactorMin || got > metrics.QualityFactorMax {
				t.Fatalf("QualityFactor out of range: %v", got)
			}
		})
	}
}

func TestChangeScoreProduct(t *testing.T) {
	// Spec worked example ≈ 17.99
	size := metrics.SizeScore(320, 180)
	complexity := metrics.ComplexityFactor(12, 3, 2, 2)
	module := 1.15
	typ := 1.10
	quality := 1.05
	got := metrics.ChangeScore(size, complexity, module, typ, quality)
	almostEqual(t, got, size*complexity*module*typ*quality, 1e-12)
	almostEqual(t, got, 17.99, 0.02)
}

func TestCompute(t *testing.T) {
	in := metrics.ScoreInput{
		AddedLines:     320,
		DeletedLines:   180,
		ChangedFiles:   12,
		ChangedModules: 3,
		Languages:      2,
		Layers:         2,
		ModuleLines:    map[string]int{"api": 100},
		Weights:        map[string]float64{"api": 1.15},
		ChangeType:     model.ChangeTypeFix,
		TypeWeights:    config.DefaultTypeWeights(),
		HasTests:       true,
		IsRevert:       false,
	}
	sb := metrics.Compute(in)
	almostEqual(t, sb.SizeScore, metrics.SizeScore(320, 180), 1e-12)
	almostEqual(t, sb.ComplexityFactor, 1.51, 1e-9)
	almostEqual(t, sb.ModuleFactor, 1.15, 1e-9)
	almostEqual(t, sb.TypeFactor, 1.10, 1e-9)
	almostEqual(t, sb.QualityFactor, 1.05, 1e-9)
	almostEqual(t, sb.Final, metrics.ChangeScore(sb.SizeScore, sb.ComplexityFactor, sb.ModuleFactor, sb.TypeFactor, sb.QualityFactor), 1e-12)
}

func TestParseChangeType(t *testing.T) {
	tests := []struct {
		msg  string
		want model.ChangeType
	}{
		{"feat: add login", model.ChangeTypeFeat},
		{"fix(api): nil deref", model.ChangeTypeFix},
		{"refactor!: explode module", model.ChangeTypeRefactor},
		{"perf(db)!: index", model.ChangeTypePerf},
		{"Revert \"feat: x\"", model.ChangeTypeRevert},
		{"revert: undo feat", model.ChangeTypeRevert},
		{"random message", model.ChangeTypeUnknown},
		{"", model.ChangeTypeUnknown},
		{"feat add without colon", model.ChangeTypeUnknown},
		{"chore: tidy\n\nbody", model.ChangeTypeChore},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			if got := metrics.ParseChangeType(tt.msg); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestInferChangeType(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		paths []string
		want model.ChangeType
	}{
		{"prefers_conventional", "feat: x", []string{"foo_test.go"}, model.ChangeTypeFeat},
		{"all_tests", "wip", []string{"a_test.go", "b.spec.ts"}, model.ChangeTypeTest},
		{"all_docs", "update", []string{"docs/guide.md", "README.md"}, model.ChangeTypeDocs},
		{"all_ci", "ci tweak", []string{".github/workflows/ci.yml"}, model.ChangeTypeCI},
		{"mixed_unknown", "wip", []string{"main.go", "a_test.go"}, model.ChangeTypeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := metrics.InferChangeType(tt.msg, tt.paths); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestGeneratedHelpers(t *testing.T) {
	pathTests := []struct {
		path string
		want bool
	}{
		{"src/main.go", false},
		{"package-lock.json", true},
		{"pnpm-lock.yaml", true},
		{"go.sum", true},
		{"app.min.js", true},
		{"bundle.js.map", true},
		{"dist/out.js", true},
		{"vendor/pkg/x.go", true},
		{"generated/api.ts", true},
		{"foo.generated.ts", true},
		{"internal/api.go", false},
	}
	for _, tt := range pathTests {
		t.Run("path_"+tt.path, func(t *testing.T) {
			if got := metrics.IsGeneratedPath(tt.path); got != tt.want {
				t.Fatalf("IsGeneratedPath(%q)=%v want %v", tt.path, got, tt.want)
			}
		})
	}

	contentTests := []struct {
		name    string
		content string
		want    bool
	}{
		{"go_codegen", "// Code generated by mockgen. DO NOT EDIT.\npackage x\n", true},
		{"normal", "package main\nfunc main() {}\n", false},
		{"auto", "/* This file is automatically generated */\n", true},
	}
	for _, tt := range contentTests {
		t.Run("content_"+tt.name, func(t *testing.T) {
			if got := metrics.IsGeneratedContent(tt.content); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}

	if !metrics.IsExcludedFromScore("dist/a.js", "") {
		t.Fatal("expected dist path excluded")
	}
}

func TestClamp(t *testing.T) {
	almostEqual(t, metrics.Clamp(5, 0, 10), 5, 0)
	almostEqual(t, metrics.Clamp(-1, 0, 10), 0, 0)
	almostEqual(t, metrics.Clamp(11, 0, 10), 10, 0)
}
