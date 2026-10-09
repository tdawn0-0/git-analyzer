# Phase 1 Design Review — git-workstats

Engineering Activity Analysis (not employee performance evaluation).  
Stack: Go, Cobra, yaml.v3, Bubble Tea, Lip Gloss, ntcharts (TUI later). Git via `exec.CommandContext` only.

Principles: local-first, multi-repository, fast, deterministic, explainable.

---

## 1. ChangeScore math review

### Master formula

```
ChangeScore =
  SizeScore
  × ComplexityFactor
  × ModuleFactor
  × TypeFactor
  × QualityFactor
```

UI label: **Change Intensity**. Code retains `ChangeScore`.

### Component formulas

| Factor | Formula | Clamp |
|--------|---------|-------|
| **SizeScore** | `min(log2(added+deleted+1), 14)` | ≤ 14 |
| **ComplexityFactor** | `1 + FilesFactor + ModulesFactor + LanguageFactor + LayerFactor` | [1.0, 2.0] |
| **ModuleFactor** | Σ(lines×weight) / Σ(lines); unmatched weight 1.0 | [0.8, 1.5] |
| **TypeFactor** | Conventional Commit type → config weight | defaults 0.50–1.15; stated guidance [0.6, 1.2] |
| **QualityFactor** | base 1.0; +0.05 if tests; ×0.5 if revert | [0.5, 1.1] |

**Complexity sub-terms**

- `FilesFactor = min(changed_files, 20) × 0.015`
- `ModulesFactor = min(max(changed_modules − 1, 0), 6) × 0.10` (first module free)
- `LanguageFactor`: 1 lang → 0; 2 → +0.05; ≥3 → +0.10
- `LayerFactor`: +0.08 per extra layer, max 3 extras (≤ +0.24)

Worked example: +320/−180 → changed=500 → Size≈8.97 × Complexity 1.51 × Module 1.15 × Type 1.10 × Quality 1.05 ≈ **17.99**.

Generated / ignored paths are **excluded** from SizeScore (and line-weighted ModuleFactor inputs) but may still surface in UI as “Generated changes — excluded from ChangeScore”.

### Statistical biases and skew risks

1. **Log size still rewards large diffs.** `log2` compresses LOC influence but a 10k-line dump (≈13.3) still outscores a careful 50-line change (≈5.7) before other factors. Bulk formatters, renames counted as delete+add, and dependency bumps inflate SizeScore unless filtered as generated/ignored.

2. **Multiplicative stacking amplifies config opinion.** Module × Type × Quality can push the same LOC into very different finals. Mis-tuned module weights become systematic skew, not noise.

3. **Complexity favors breadth over depth.** Touching many files/modules/languages/layers raises the factor even when each edit is trivial; a deep single-file algorithmic change scores lower on complexity by design.

4. **ModuleFactor is configuration-biased.** Unmatched paths default to 1.0, so unconfigured critical paths look “average” while labeled paths dominate. Empty line maps must default to 1.0 (no divide-by-zero → NaN).

5. **TypeFactor is message-protocol biased.** Teams without Conventional Commits collapse to `unknown` (1.0) or heuristic fallbacks (`*_test.go` → test). Message gaming (`feat:` on chores) directly scales scores. Default `revert: 0.50` sits **below** the stated 0.6–1.2 guidance range — treat the table as authoritative defaults; clamp only when applying non-default config if desired.

6. **QualityFactor is intentionally weak — and asymmetric.** +0.05 for tests barely moves the product; ×0.5 for revert is strong. That is correct for “not performance review,” but revert-heavy maintenance work will look quieter than feature work of equal LOC.

7. **Cross-repo aggregation skew.** Summing Change Intensity across repositories favors people who touch many/large repos. Totals must stay activity signals, never leaderboards. Repo size and commit style (many small vs few large) dominate developer averages (`ScorePerChange = Total / Count`).

8. **Discovery / identity skew (adjacent to scoring).** Duplicate basenames need disambiguation in UI; author identity without mailmap/config splits one person into many. Nested repos analyzed twice if parent diffs incorrectly include nested trees — mitigated by analyzing only via `git -C <repo>`.

9. **Determinism vs truth.** Heuristic language/layer/module classification will be wrong sometimes; scores remain explainable only if Score Explain surfaces every factor input.

**Mitigations (Phase 1 stance):** caps on every factor; generated/ignore filters; config-driven weights (no AI importance); Score Explain; UX forbids Top/Best/Winner framing.

---

## 2. Multi-repository Workspace model review

```
Workspace
├── Repository A → ChangeUnit…
├── Repository B → ChangeUnit…
└── Repository C → ChangeUnit…

Developer (cross-cutting identity)
├── repo A changes
├── repo B …
└── repo C …
```

**Entry:** path is a git repo → single-repo workspace; else recursively discover repos under the root.

**Strengths**

- Matches real engineer layouts (`~/code`, monorepo + nested tool repos, worktrees).
- Repository-centric and developer-centric views share the same `ChangeUnit` list.
- Failures are per-repo (`Ready|Analyzing|Complete|Skipped|Error`); one bad repo does not fail the workspace.

**Risks / decisions**

| Topic | Decision |
|-------|----------|
| Nested `.git` | Discover **both** parent and nested as independent repositories |
| Submodules (later analyzer) | Checked-out submodules = separate repos; no auto `submodule update` |
| Symlinks | Do **not** follow directory symlinks by default (loop safety) |
| ID | `remote.origin.url` if set, else hash of absolute path |
| Name | `basename(root)`; collisions → show relative path suffix in UI |
| Config | Workspace `.workstats.yml` applies globally; per-repo overrides; CLI wins |
| Change unit (v1) | Commit |

Phase 1 implements discovery + model types only. Analysis, aggregation, and TUI come later. Pipeline remains:

```
Git → Raw Change Data → Metrics → Aggregation → WorkspaceStats → TUI
```

Git must never be called from Bubble Tea views.

---

## 3. Final Go project architecture

```
cmd/
  git-workstats/
    main.go                 # Cobra entry (stub in Phase 1)

internal/
  app/                      # wiring / run lifecycle (later)
  git/
    command.go              # exec.CommandContext helpers
    repository.go           # ID/name/remote helpers
    discovery.go            # RepositoryScanner
  workspace/                # scan orchestration (thin; later analyzer/pool)
  model/                    # Workspace, Repository, ChangeUnit, scores, stats
  metrics/                  # pure scoring functions (no git I/O)
  config/                   # YAML load/merge + default weights
  aggregate/                # reserved (not Phase 1)
  tui/                      # reserved (not Phase 1)

docs/
  phase1-design.md
```

**Decoupling rules**

- `metrics` is pure (float/int/string in → float out); unit-tested without git.
- `git` owns process execution and filesystem discovery confirmation via `git -C … rev-parse`.
- `model` has no I/O.
- `config` supplies type/module weights to metrics; no scoring logic inside YAML loaders.

Module path: `github.com/tdawn0-0/git-analyzer` (repository name) with binary name `git-workstats`.

---

## 4. Core interfaces and structs

### Discovery

```go
type ScanOptions struct {
    MaxDepth      int      // default 6
    Exclude       []string // dir names and/or globs
    FollowSymlink bool     // default false
}

type RepositoryScanner interface {
    Scan(ctx context.Context, root string, opts ScanOptions) ([]model.Repository, error)
}
```

### Domain (Phase 1)

```go
type Workspace struct {
    Root         string
    Repositories []Repository
}

type Repository struct {
    ID     string // remote.origin.url or abs-path hash
    Name   string // basename
    Path   string
    Remote string
}

type ChangeUnit struct {
    RepositoryID string
    Hash         string
    Author       Author
    Timestamp    time.Time
    Message      string
    AddedLines, DeletedLines int
    GeneratedAdded, GeneratedDeleted int
    ChangedFiles int
    Modules, Languages, Layers []string
    Type  ChangeType
    Score ScoreBreakdown
}

type ScoreBreakdown struct {
    SizeScore, ComplexityFactor, ModuleFactor, TypeFactor, QualityFactor float64
    Final float64
}
```

Aggregation structs (`DeveloperStats`, `RepositoryStats`, `WorkspaceStats`) are defined for forward compatibility; population is out of Phase 1 scope.

### Metrics API (pure)

- `SizeScore(added, deleted int) float64`
- `ComplexityFactor(files, modules, languages, layers int) float64`
- `ModuleFactor(moduleLines map[string]int, weights map[string]float64) float64`
- `TypeFactor(changeType ChangeType, weights map[string]float64) float64`
- `QualityFactor(hasTests, isRevert bool) float64`
- `ChangeScore(...) float64` / `Compute(ScoreInput) ScoreBreakdown`
- Helpers: generated path/header detection used to exclude LOC from scoring inputs

---

## 5. Phase 1 deliverable boundary

| In scope | Out of scope (stop here) |
|----------|---------------------------|
| This design review | Git log/diff analyzer |
| Module scaffold | Aggregation / worker pool |
| `internal/metrics` + tests | Bubble Tea / Lip Gloss / ntcharts |
| `RepositoryScanner` + tests | GitHub/GitLab APIs, LLM, DB, plugins |

---

## 6. Default config weights (reference)

**Types:** feat/perf 1.15; fix/refactor 1.10; test 0.90; build/ci 0.80; docs/chore 0.70; style 0.60; revert 0.50; unknown 1.00.

**Discovery excludes (directory basenames):** `node_modules`, `vendor`, `target`, `dist`, `build`, `.next`, `.cache`, `.idea`, `.vscode`, `coverage`, `tmp`, `temp`, plus never recurse into `.git`.

**max-depth:** 6.
