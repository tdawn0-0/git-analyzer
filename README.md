# git-workstats

Local-first multi-repository Git **engineering activity** analysis (not employee performance evaluation).

## Status

**MVP Phases 1–3:** design, metrics, discovery, analyzer, aggregation, Bubble Tea TUI (Lip Gloss + ntcharts + Score Explain).

## Run TUI

```bash
go test ./...
go build -o git-workstats ./cmd/git-workstats

./git-workstats ~/code --since 2026-09-01 --jobs 4
```

Text summary (no TUI):

```bash
./git-workstats --text ~/code
```

### Keys

`j/k` move · `Enter`/`l` open · `h` back · `Tab` focus · `e` Score Explain · `r`/`d` repo/developer · `t` trend · `s` sort · `/` filter · `?` help · `q` quit · `Ctrl+C` cancel/quit

### Flags

`--since`, `--until`, `--author`, `--repo`, `--branch`, `--max-depth`, `--jobs`, `--exclude`, `--text`

Optional config: `.workstats.yml`

## Layout

See [`docs/phase1-design.md`](docs/phase1-design.md).
