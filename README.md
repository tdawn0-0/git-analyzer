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

`j/k` move (scroll in Explain) · `PgUp/PgDn` page · `Home/End` scroll · `Enter`/`l` open · `h` back · `Tab` focus · `e` Score Explain · `r`/`d` repo/developer · `t` trend · `s` sort · `/` filter · `?` help · `q` quit · `Ctrl+C` cancel/quit

### Flags

`--since`, `--until`, `--author`, `--repo`, `--branch`, `--max-depth`, `--jobs`, `--exclude`, `--text`

Optional config: `.workstats.yml`

## Layout

See [`docs/phase1-design.md`](docs/phase1-design.md).

### Analysis behavior

Text mode and the TUI share the same analysis pipeline. Author filters match resolved
mailmap/config names and emails, using case-insensitive substring matching.

Checkouts with the same `origin` URL are grouped into one logical repository. Linked
worktrees without an origin are grouped by their shared Git directory. Shared commit
hashes count once; commits unique to a checkout or branch remain included. The first
checkout in discovery order supplies the display name and metadata. Separate origins
remain separate repositories, even if they share commit history.

Lists keep the selected row visible. Use `j/k` to scroll Score Explain and `PgUp/PgDn`
to page through lists or the explanation; `h` restores the previous view and selection.

### Table columns

- `Developer` / `Repo`: developer or repository name.
- `Intensity`: total Change Intensity score.
- `Commits`: commit count.
- `Added` / `Deleted`: added/deleted lines, excluding generated changes.
- `State`: `OK` complete, `ERR` error, `SKIP` skipped.
- `Commit`, `Date`, `Type`, `Message`: commit hash, UTC date, change type and subject.

Narrow panels show fewer columns so headers and values stay aligned. Open a detail
view for more information. Press `?` for column definitions and keyboard help.
