package metrics

import "math"

// Clamp ranges from the Phase 1 design spec.
const (
	SizeScoreMax           = 14.0
	ComplexityFactorMin    = 1.0
	ComplexityFactorMax    = 2.0
	ModuleFactorMin        = 0.8
	ModuleFactorMax        = 1.5
	TypeFactorMin          = 0.6
	TypeFactorMax          = 1.2
	QualityFactorMin       = 0.5
	QualityFactorMax       = 1.1
	filesFactorCap         = 20
	filesFactorWeight      = 0.015
	extraModulesCap        = 6
	extraModulesWeight     = 0.10
	extraLayersCap         = 3
	extraLayersWeight      = 0.08
	qualityTestBonus       = 0.05
	qualityRevertMultiplier = 0.5
)

// Clamp returns v limited to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// SizeScore is min(log2(added+deleted+1), 14). Generated lines must be excluded by the caller.
func SizeScore(added, deleted int) float64 {
	changed := added + deleted
	if changed < 0 {
		changed = 0
	}
	return math.Min(math.Log2(float64(changed)+1), SizeScoreMax)
}

// FilesFactor = min(changed_files, 20) * 0.015
func FilesFactor(changedFiles int) float64 {
	n := changedFiles
	if n < 0 {
		n = 0
	}
	if n > filesFactorCap {
		n = filesFactorCap
	}
	return float64(n) * filesFactorWeight
}

// ModulesFactor = min(max(changed_modules-1, 0), 6) * 0.10
func ModulesFactor(changedModules int) float64 {
	extra := changedModules - 1
	if extra < 0 {
		extra = 0
	}
	if extra > extraModulesCap {
		extra = extraModulesCap
	}
	return float64(extra) * extraModulesWeight
}

// LanguageFactor: 1 → 0; 2 → 0.05; ≥3 → 0.10
func LanguageFactor(languageCount int) float64 {
	switch {
	case languageCount >= 3:
		return 0.10
	case languageCount == 2:
		return 0.05
	default:
		return 0
	}
}

// LayerFactor: +0.08 per extra layer, max 3 extras.
func LayerFactor(layerCount int) float64 {
	extra := layerCount - 1
	if extra < 0 {
		extra = 0
	}
	if extra > extraLayersCap {
		extra = extraLayersCap
	}
	return float64(extra) * extraLayersWeight
}

// ComplexityFactor = 1 + Files + Modules + Language + Layer, clamped to [1.0, 2.0].
func ComplexityFactor(changedFiles, changedModules, languages, layers int) float64 {
	v := 1.0 +
		FilesFactor(changedFiles) +
		ModulesFactor(changedModules) +
		LanguageFactor(languages) +
		LayerFactor(layers)
	return Clamp(v, ComplexityFactorMin, ComplexityFactorMax)
}

// ModuleFactor is the line-weighted average of module weights, clamped to [0.8, 1.5].
// Unmatched module names use weight 1.0. Zero total lines → 1.0.
func ModuleFactor(moduleLines map[string]int, weights map[string]float64) float64 {
	var weighted, total float64
	for name, lines := range moduleLines {
		if lines <= 0 {
			continue
		}
		w := 1.0
		if weights != nil {
			if configured, ok := weights[name]; ok {
				w = configured
			}
		}
		l := float64(lines)
		weighted += l * w
		total += l
	}
	if total == 0 {
		return 1.0
	}
	return Clamp(weighted/total, ModuleFactorMin, ModuleFactorMax)
}

// TypeFactor looks up a change type weight. Defaults to 1.0 when missing.
// Spec defaults include revert=0.50 (below the stated 0.6–1.2 guidance); we do not
// re-clamp configured defaults so Score Explain matches the weight table.
func TypeFactor(changeType string, weights map[string]float64) float64 {
	if weights != nil {
		if w, ok := weights[changeType]; ok {
			return w
		}
	}
	return 1.0
}

// QualityFactor starts at 1.0, adds 0.05 when tests are present, multiplies by 0.5 on revert,
// then clamps to [0.5, 1.1].
func QualityFactor(hasTests, isRevert bool) float64 {
	v := 1.0
	if hasTests {
		v += qualityTestBonus
	}
	if isRevert {
		v *= qualityRevertMultiplier
	}
	return Clamp(v, QualityFactorMin, QualityFactorMax)
}

// ChangeScore is the product of the five factors (no extra clamp on the product).
func ChangeScore(size, complexity, module, typ, quality float64) float64 {
	return size * complexity * module * typ * quality
}
