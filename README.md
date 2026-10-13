# hncapture

Captures the top Hacker News stories with their full comment trees and the profile of every author, for compliance archiving.

## Run

```sh
go run ./cmd/hncapture            # top 5 stories → tree on stdout, JSON archive in ./output
```

Requires Go 1.24+. No third-party dependencies.

```sh
go run ./cmd/hncapture capture --top 10 --concurrency 32 --format json
go run ./cmd/hncapture story 8863          # one thread
go run ./cmd/hncapture user jl             # one profile
go run ./cmd/hncapture capture -h          # all flags
go test -race ./...
```

Flags for `capture` and `story`: `--out` (archive dir, `""` = don't save), `--format tree|json|none`, `--concurrency` (default 16), `--timeout` (default 5m), `--full-profiles`. Exit codes: 0 ok (including partial captures), 1 runtime failure, 2 usage error.

Sample output from a real run: [output/capture.txt](output/capture.txt) (tree) and `output/capture-*.json` (archive).

## Reading the output

- **Thread with depth**: the tree prints `[dN]` on every line, where `d0` is the story and `d1` is a top-level comment. In JSON, each `node` has `depth` and nested `children` in HN's ranked order.
- **Author of every item**: `by <user>` on each line and `item.by` in JSON. Every author's profile is under `users`.
- **Missing items** stay in the tree as `[DELETED]`, `[DEAD]`, `[NOT FOUND]` or `[ERROR: …]`, and are listed again per story under `missing` with parent id, depth and reason. `stats` compares `captured_comments` against HN's own `expected_descendants`, so gaps the tree can't show (an id HN never put in `kids`) still show up as a mismatch. Unfetchable profiles go to `missing_users`.

## Decisions

- **Hexagonal layout.** `internal/core` (domain, ports, service) has no knowledge of HTTP, files or the CLI. The CLI (`internal/adapters/cli`) is a driving adapter that calls only the `ports.Facade`. The HN HTTP client and the JSON file writer are driven adapters behind `ports.HNClient` and `ports.ArchiveWriter`. `cmd/hncapture` is the only place the pieces are wired together. Saving goes through `Facade.Save`, so the CLI never touches a driven port.
- **Concurrency.** Each node's kids are fetched in parallel goroutines, stories in parallel, then all profiles in parallel. A per-run semaphore caps in-flight HTTP calls. It is held only for the duration of one request, never while waiting on children, so deep trees can't deadlock it. Children are written by index, so ranking order is preserved without sorting. A real top-5 capture (~780 comments, ~520 profiles) takes about 15–20 s at the default of 16.
- **Failures are data, not crashes.** Only an unreachable `topstories` call is fatal. A failed item or profile, after 3 retries with exponential backoff on network errors, 429 and 5xx, becomes an `error` node with its reason, and the rest of the capture continues.
- **Deleted and dead items are recursed into.** A deleted comment can still have live replies. Dead items keep their content but are flagged, because they're hidden on the site.
- **Profiles store `submitted_count` instead of the full `submitted` id list by default.** Those lists were ~97% of the archive (26 MB → 0.7 MB for a top-5 run). Pass `--full-profiles` to keep them.
- **JSON archive** written atomically (temp file + rename), one timestamped file per run.
- **Stdlib only** (`flag`, `net/http`, `encoding/json`): it's a small tool, and this avoids supply-chain risk.

## Assumptions

- "Top 5" means the first 5 ids of `/v0/topstories` at capture time. That list can include jobs and polls, which are captured as-is with their `type`.
- "Every author involved" means the story author plus every comment author found in the tree. Deleted items have no author.
- A snapshot is good enough: HN keeps changing during a run (a comment may land mid-capture), and `captured_at` records when the run started.

## What I cut

- Poll options (`parts`) aren't fetched.
- No live updates or Firebase subscriptions. Fetching is on demand only, as requested.
- No storage beyond a JSON file per run: no database, deduplication across runs, or diffing.
- No client-side rate limiting beyond the concurrency cap (the API documents no rate limit).
- HTML in titles and comments is kept raw in JSON. The tree view strips tags and truncates to a 120-character snippet.

## How I used AI

I built this with Claude Code. First came a planning pass ([PLAN.md](PLAN.md)) in which I decided the open questions: output format, CLI library, commands, how to treat missing items, and later how to handle `submitted` once a live run showed the 26 MB archive. Then it was implemented test-first, package by package (fake `HNClient` for the service, `httptest` for the HTTP adapter, a fake `Facade` for the CLI), with a commit per step. Each step was checked with `go vet` and `go test -race`, and the final behavior against the live API.
