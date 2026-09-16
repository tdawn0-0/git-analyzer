# git-workstats

Local-first multi-repository Git **engineering activity** analysis (not employee performance evaluation).

## Status

**Phase 1–2 complete** (design, metrics, discovery, Git analyzer, aggregation, Cobra CLI).

Not yet: Bubble Tea TUI / Lip Gloss / ntcharts / Score Explain UI.

## Usage

```bash
go test ./...
go build -o git-workstats ./cmd/git-workstats

./git-workstats ~/code \
  --since 2026-09-01 \
  --until 2026-09-30 \
  --jobs 4 \
  --max-depth 6
```

Flags: `--since`, `--until`, `--author`, `--repo`, `--branch`, `--max-depth`, `--jobs`, `--exclude`.

Optional workspace config: `.workstats.yml` (modules, types, authors, ignore/generated).

## Layout

See [`docs/phase1-design.md`](docs/phase1-design.md) for architecture and ChangeScore math.
