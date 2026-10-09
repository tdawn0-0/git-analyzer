package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/metrics"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

func renderExplain(c model.ChangeUnit, repoName string, width int) string {
	changed := c.AddedLines + c.DeletedLines
	sizeRaw := math.Log2(float64(changed) + 1)

	filesF := metrics.FilesFactor(c.ChangedFiles)
	modsF := metrics.ModulesFactor(len(c.Modules))
	langF := metrics.LanguageFactor(len(c.Languages))
	layerF := metrics.LayerFactor(len(c.Layers))

	hash := c.Hash
	if len(hash) > 12 {
		hash = hash[:12]
	}

	var b strings.Builder
	b.WriteString(styleTitle().Render("Change Intensity Explanation") + "\n\n")
	b.WriteString(fmt.Sprintf("Repository   %s\n", repoName))
	b.WriteString(fmt.Sprintf("Commit       %s\n", hash))
	b.WriteString(fmt.Sprintf("Author       %s <%s>\n", c.Author.Name, c.Author.Email))
	b.WriteString(fmt.Sprintf("Message      %s\n\n", c.Message))

	b.WriteString(styleHeader().Render("Size") + "\n")
	b.WriteString(fmt.Sprintf("  added=%d  deleted=%d  changed=%d\n", c.AddedLines, c.DeletedLines, changed))
	if c.GeneratedAdded > 0 || c.GeneratedDeleted > 0 {
		b.WriteString(styleMuted().Render(fmt.Sprintf(
			"  generated +%d/−%d  (Excluded from ChangeScore)\n", c.GeneratedAdded, c.GeneratedDeleted)))
	}
	b.WriteString(fmt.Sprintf("  log2(%d+1)=%.4f\n", changed, sizeRaw))
	b.WriteString(fmt.Sprintf("  SizeScore = %.4f  (cap %.0f)\n\n", c.Score.SizeScore, metrics.SizeScoreMax))

	b.WriteString(styleHeader().Render("Complexity") + "\n")
	b.WriteString(fmt.Sprintf("  files %d  +%.3f\n", c.ChangedFiles, filesF))
	b.WriteString(fmt.Sprintf("  modules %d  +%.3f  %v\n", len(c.Modules), modsF, c.Modules))
	b.WriteString(fmt.Sprintf("  languages %d  +%.3f  %v\n", len(c.Languages), langF, c.Languages))
	b.WriteString(fmt.Sprintf("  layers %d  +%.3f  %v\n", len(c.Layers), layerF, c.Layers))
	b.WriteString(fmt.Sprintf("  ComplexityFactor = %.4f\n\n", c.Score.ComplexityFactor))

	b.WriteString(styleHeader().Render("Modules") + "\n")
	if len(c.Modules) == 0 {
		b.WriteString("  (unmatched paths → weight 1.0)\n")
	} else {
		for _, name := range c.Modules {
			b.WriteString(fmt.Sprintf("  %s\n", name))
		}
	}
	b.WriteString(fmt.Sprintf("  ModuleFactor = %.4f\n\n", c.Score.ModuleFactor))

	b.WriteString(styleHeader().Render("Type") + "\n")
	b.WriteString(fmt.Sprintf("  type=%s\n", c.Type))
	b.WriteString(fmt.Sprintf("  TypeFactor = %.4f\n\n", c.Score.TypeFactor))

	b.WriteString(styleHeader().Render("Quality") + "\n")
	reasons := []string{}
	if c.Type == model.ChangeTypeTest || hasTestHint(c) {
		reasons = append(reasons, "tests changed (+0.05)")
	}
	if c.Type == model.ChangeTypeRevert {
		reasons = append(reasons, "revert (×0.5)")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "default")
	}
	b.WriteString(fmt.Sprintf("  %s\n", strings.Join(reasons, "; ")))
	b.WriteString(fmt.Sprintf("  QualityFactor = %.4f\n\n", c.Score.QualityFactor))

	b.WriteString(styleAccent().Bold(true).Render("FINAL") + "\n")
	b.WriteString(fmt.Sprintf("  %.4f × %.4f × %.4f × %.4f × %.4f\n",
		c.Score.SizeScore, c.Score.ComplexityFactor, c.Score.ModuleFactor,
		c.Score.TypeFactor, c.Score.QualityFactor))
	b.WriteString(styleAccent().Bold(true).Render(fmt.Sprintf("  = %.4f  Change Intensity", c.Score.Final)) + "\n")
	b.WriteString("\n" + styleMuted().Render("h back · j/k other changes · q quit"))

	_ = width
	return b.String()
}

func hasTestHint(c model.ChangeUnit) bool {
	// Quality bonus already baked into Score; surface heuristic from type/files.
	return c.Score.QualityFactor >= 1.05 || c.Type == model.ChangeTypeTest
}
