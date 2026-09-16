package model

import "time"

// Workspace is a root directory and the Git repositories discovered under it.
type Workspace struct {
	Root         string
	Repositories []Repository
}

// Repository is a single local Git repository (including worktrees and nested repos).
type Repository struct {
	ID     string // remote.origin.url if set, else hash of absolute path
	Name   string // basename of repository root
	Path   string // absolute path to repository root
	Remote string // remote.origin.url when available
}

// Author is a Git identity after optional mailmap / config resolution (later phases).
type Author struct {
	Name  string
	Email string
}

// ChangeType is a Conventional Commit type or heuristic fallback.
type ChangeType string

const (
	ChangeTypeFeat     ChangeType = "feat"
	ChangeTypeFix      ChangeType = "fix"
	ChangeTypeRefactor ChangeType = "refactor"
	ChangeTypePerf     ChangeType = "perf"
	ChangeTypeTest     ChangeType = "test"
	ChangeTypeDocs     ChangeType = "docs"
	ChangeTypeBuild    ChangeType = "build"
	ChangeTypeCI       ChangeType = "ci"
	ChangeTypeChore    ChangeType = "chore"
	ChangeTypeStyle    ChangeType = "style"
	ChangeTypeRevert   ChangeType = "revert"
	ChangeTypeUnknown  ChangeType = "unknown"
)

// ScoreBreakdown is the explainable product of ChangeScore factors.
type ScoreBreakdown struct {
	SizeScore        float64
	ComplexityFactor float64
	ModuleFactor     float64
	TypeFactor       float64
	QualityFactor    float64
	Final            float64
}

// ChangeUnit is one analyzed change (v1 default: a commit).
type ChangeUnit struct {
	RepositoryID string

	Hash      string
	Author    Author
	Timestamp time.Time
	Message   string

	AddedLines   int
	DeletedLines int

	GeneratedAdded   int
	GeneratedDeleted int

	ChangedFiles int

	Modules   []string
	Languages []string
	Layers    []string

	Type  ChangeType
	Score ScoreBreakdown
}

// DeveloperStats aggregates ChangeUnits for one developer identity (populated later).
type DeveloperStats struct {
	Developer string

	RepositoryIDs []string

	ChangeCount int

	TotalScore   float64
	AverageScore float64

	AddedLines   int
	DeletedLines int

	ModulesTouched   map[string]struct{}
	TypeDistribution map[ChangeType]int
}

// RepositoryStats aggregates ChangeUnits for one repository (populated later).
type RepositoryStats struct {
	Repository Repository

	ChangeCount int
	Developers  []string

	TotalScore float64

	AddedLines   int
	DeletedLines int

	TypeDistribution map[ChangeType]int
}

// WorkspaceStats is the full analysis result for the TUI (populated later).
type WorkspaceStats struct {
	Root string

	Repositories []RepositoryStats
	Developers   []DeveloperStats
	Changes      []ChangeUnit
}

// RepositoryStatus is the per-repo analysis lifecycle state.
type RepositoryStatus string

const (
	StatusReady     RepositoryStatus = "Ready"
	StatusAnalyzing RepositoryStatus = "Analyzing"
	StatusComplete  RepositoryStatus = "Complete"
	StatusSkipped   RepositoryStatus = "Skipped"
	StatusError     RepositoryStatus = "Error"
)
