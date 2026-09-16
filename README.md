# git-workstats

Local-first multi-repository Git **engineering activity** analysis (not employee performance evaluation).

## Phase 1 status

Foundation only:

- Design review: [`docs/phase1-design.md`](docs/phase1-design.md)
- Pure `ChangeScore` metrics (`internal/metrics`)
- Repository discovery (`internal/git.RepositoryScanner`)
- Config defaults / YAML load (`internal/config`)

Not yet: Git analyzer, aggregation, Bubble Tea TUI.

## Build / test

```bash
go test ./...
go build -o git-workstats ./cmd/git-workstats
```

## Layout

See `docs/phase1-design.md` §3 for the `cmd/` + `internal/` package map.
