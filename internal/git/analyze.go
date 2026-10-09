package git

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tdawn0-0/git-analyzer/internal/config"
	"github.com/tdawn0-0/git-analyzer/internal/metrics"
	"github.com/tdawn0-0/git-analyzer/internal/model"
)

// RepoResult is the per-repository analysis outcome (isolated from workspace failures).
type RepoResult struct {
	Repository model.Repository
	Status     model.RepositoryStatus
	Changes    []model.ChangeUnit
	Err        error
	Message    string
}

// AnalyzeRepository loads commits for one repo and builds scored ChangeUnits.
// Missing branch yields StatusSkipped with Message "branch unavailable" and nil error.
func AnalyzeRepository(ctx context.Context, repo model.Repository, cfg config.Config, opts AnalyzeOptions) RepoResult {
	res := RepoResult{
		Repository: repo,
		Status:     model.StatusAnalyzing,
	}
	if err := ctx.Err(); err != nil {
		res.Status = model.StatusError
		res.Err = err
		res.Message = err.Error()
		return res
	}

	if opts.Branch != "" && !BranchExists(ctx, repo.Path, opts.Branch) {
		res.Status = model.StatusSkipped
		res.Message = "branch unavailable"
		return res
	}

	raw, err := LogCommits(ctx, repo.Path, opts)
	if err != nil {
		res.Status = model.StatusError
		res.Err = err
		res.Message = err.Error()
		return res
	}

	modules := cfg.ModulesForRepo(repo.Name)
	weights := config.ModuleWeights(modules)
	typeWeights := cfg.Types
	if typeWeights == nil {
		typeWeights = config.DefaultTypeWeights()
	}

	headerCache := map[string]bool{}
	changes := make([]model.ChangeUnit, 0, len(raw))
	for _, c := range raw {
		if err := ctx.Err(); err != nil {
			res.Status = model.StatusError
			res.Err = err
			res.Message = err.Error()
			return res
		}
		cu := buildChangeUnit(ctx, repo, c, cfg, modules, weights, typeWeights, headerCache)
		changes = append(changes, cu)
	}

	res.Changes = changes
	res.Status = model.StatusComplete
	return res
}

func buildChangeUnit(
	ctx context.Context,
	repo model.Repository,
	c RawCommit,
	cfg config.Config,
	modules map[string]config.ModuleDef,
	weights map[string]float64,
	typeWeights map[string]float64,
	headerCache map[string]bool,
) model.ChangeUnit {
	// Author/email already mailmap-resolved via `git log --use-mailmap`.
	name, email := c.Author, c.Email
	developer := cfg.ResolveDeveloper(name, email)

	var (
		added, deleted       int
		genAdded, genDeleted int
		changedFiles         int
		paths                []string
		moduleLines          = map[string]int{}
		moduleSet            = map[string]struct{}{}
		langSet              = map[string]struct{}{}
		hasTests             bool
	)

	for _, f := range c.Files {
		path := NormalizeNumstatPath(f.Path)
		if path == "" || f.Binary {
			continue
		}
		generated := isGeneratedFile(ctx, repo.Path, c.Hash, path, cfg, headerCache)
		if generated {
			genAdded += f.Added
			genDeleted += f.Deleted
			continue
		}

		changedFiles++
		added += f.Added
		deleted += f.Deleted
		paths = append(paths, path)
		if IsTestPath(path) {
			hasTests = true
		}
		if lang := LanguageFromPath(path); lang != "" {
			langSet[lang] = struct{}{}
		}
		matched := MatchModules(path, modules)
		for _, m := range matched {
			moduleSet[m] = struct{}{}
		}
		lines := f.Added + f.Deleted
		if primary := PrimaryModule(matched); primary != "" {
			moduleLines[primary] += lines
		} else if lines > 0 {
			moduleLines["(unmatched)"] += lines
			// unmatched weight defaults to 1.0 inside ModuleFactor
		}
	}

	modulesList := setKeys(moduleSet)
	langs := setKeys(langSet)
	layers := LayersForModules(modulesList, modules)

	changeType := metrics.InferChangeType(c.Subject, paths)
	isRevert := changeType == model.ChangeTypeRevert
	if changeType == model.ChangeTypeTest {
		hasTests = true
	}

	score := metrics.Compute(metrics.ScoreInput{
		AddedLines:     added,
		DeletedLines:   deleted,
		ChangedFiles:   changedFiles,
		ChangedModules: len(modulesList),
		Languages:      len(langs),
		Layers:         len(layers),
		ModuleLines:    moduleLines,
		Weights:        weights,
		ChangeType:     changeType,
		TypeWeights:    typeWeights,
		HasTests:       hasTests,
		IsRevert:       isRevert,
	})

	return model.ChangeUnit{
		RepositoryID:     repo.ID,
		Hash:             c.Hash,
		Author:           model.Author{Name: developer, Email: email},
		Timestamp:        c.Date,
		Message:          c.Subject,
		AddedLines:       added,
		DeletedLines:     deleted,
		GeneratedAdded:   genAdded,
		GeneratedDeleted: genDeleted,
		ChangedFiles:     changedFiles,
		Modules:          modulesList,
		Languages:        langs,
		Layers:           layers,
		Type:             changeType,
		Score:            score,
	}
}

func isGeneratedFile(
	ctx context.Context,
	repoPath, commit, path string,
	cfg config.Config,
	cache map[string]bool,
) bool {
	if metrics.IsGeneratedPath(path) {
		return true
	}
	if config.PathExcluded(path, cfg.Ignore) || config.PathExcluded(path, cfg.Generated) {
		return true
	}
	if !IsSourceExt(path) {
		return false
	}
	key := commit + ":" + path
	if v, ok := cache[key]; ok {
		return v
	}
	prefix, err := ShowFilePrefix(ctx, repoPath, commit, path, 2048)
	if err != nil {
		cache[key] = false
		return false
	}
	gen := metrics.IsGeneratedContent(prefix)
	cache[key] = gen
	return gen
}

func setKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FilterChangesByAuthor keeps changes whose resolved author name/email matches needle
// (case-insensitive substring). Used when --author names a config canonical identity
// that git log --author would not match on raw emails alone.
func FilterChangesByAuthor(changes []model.ChangeUnit, needle string) []model.ChangeUnit {
	if needle == "" {
		return changes
	}
	n := strings.ToLower(needle)
	var out []model.ChangeUnit
	for _, c := range changes {
		if strings.Contains(strings.ToLower(c.Author.Name), n) ||
			strings.Contains(strings.ToLower(c.Author.Email), n) {
			out = append(out, c)
		}
	}
	return out
}

// ErrBranchUnavailable is a sentinel description for skipped repos.
var ErrBranchUnavailable = fmt.Errorf("branch unavailable")
