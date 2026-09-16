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

// Author is a Git identity after optional mailmap / config resolution.
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

// DeveloperStats aggregates ChangeUnits for one developer identity.
type DeveloperStats struct {
	Developer string

	RepositoryIDs []string

	ChangeCount int

	TotalScore   float64
	AverageScore float64

	AddedLines   int
	DeletedLines int
	NetLines     int

	ModulesTouched     map[string]struct{}
	CrossModuleChanges int

	TypeDistribution map[ChangeType]int

	FeatureCount  int
	FixCount      int
	RefactorCount int
	PerfCount     int
	TestCount     int
	DocsCount     int
	ChoreCount    int
}

// RepositoryStats aggregates ChangeUnits for one repository.
type RepositoryStats struct {
	Repository Repository

	Status RepositoryStatus

	ChangeCount int
	Developers  []string

	TotalScore float64

	AddedLines   int
	DeletedLines int

	TypeDistribution map[ChangeType]int

	Error string // set when Status is Error or Skipped with reason
}

// DailyBucket is one calendar day of Change Intensity / line activity.
type DailyBucket struct {
	Date         time.Time // truncated to UTC midnight
	TotalScore   float64
	AddedLines   int
	DeletedLines int
	ChangeCount  int
}

// WorkspaceStats is the full analysis result for the CLI / future TUI.
type WorkspaceStats struct {
	Root string

	Repositories []RepositoryStats
	Developers   []DeveloperStats
	Changes      []ChangeUnit
	Timeline     []DailyBucket
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
