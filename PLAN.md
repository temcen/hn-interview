# Plan: HN capture pipeline (Go, hexagonal, CLI)

## Context
Interview take-home: archive public HN discussion threads for compliance.
Capture the **top 5 stories**, each with its **complete comment tree**, and the **profile of every author involved**. Fetches must run **concurrently**.
From the output alone a reader must be able to see: (1) the full thread with depth, (2) the author of every post and comment, (3) any items missing from a thread.
Deliverables: runnable with one command, its output, and a short README (decisions, assumptions, what was cut, how AI was used).

Decisions:
- Hexagonal architecture: the CLI is a driving adapter that talks to the core **only** through the `Facade` (driving port). The core has no knowledge of HTTP, files, or the CLI.
- Stdlib only: `flag`, `net/http`, `encoding/json`, `sync`. No third-party dependencies.
- Commands: `capture` (full pipeline), `story <id>`, `user <id>`. Every fetch is on demand: no daemon and no Firebase live subscriptions.
- Output: a nested JSON archive file plus an indented text tree on stdout.
- Missing items stay in the tree as marked nodes and are also listed per story.

## Layout
```
cmd/hncapture/main.go               # composition root: build adapters, inject into service, run CLI
internal/core/domain/               # pure types, no I/O (JSON tags = archive schema)
    item.go      Item (raw HN item fields), ItemType
    status.go    Status: ok | dead | deleted | not_found | error; ErrNotFound
    thread.go    Thread{Root Node, Missing []MissingItem, Stats}, Node{ID, Depth, Status, Reason, Item, Children}, Node.Walk
    user.go      User, MissingUser
    capture.go   Capture{CapturedAt, Stories []Thread, Users map[string]User, MissingUsers}
internal/core/ports/
    driven.go    HNClient { TopStoryIDs; Item(id) (*Item, error); User(id) (*User, error) }   // (nil, nil) = not found
                 ArchiveWriter { Write(ctx, Capture, dest) (location, error) }
    driving.go   Facade { CaptureTop(ctx, n, opts); CaptureStory(ctx, id, opts); GetUser(ctx, id); Save(ctx, c, dest) }
                 CaptureOptions{Concurrency}
internal/core/service/              # implements ports.Facade; deps: HNClient, ArchiveWriter
    service.go   facade methods
    tree.go      concurrent recursive tree builder + summarize (missing list, stats)
    users.go     author collection + deduplicated concurrent profile fetch
internal/adapters/hnapi/            # driven: HTTP implementation of HNClient
internal/adapters/storage/jsonfile/ # driven: ArchiveWriter -> <dest dir>/capture-<UTC timestamp>.json
internal/adapters/cli/              # driving: subcommand dispatch (flag.FlagSet) + text renderer; depends only on ports.Facade
```
Dependency rule: `adapters` → `ports` ← `service` → `domain`. Only `cmd/` imports the concrete adapters.

## Core behavior
**CaptureTop(n)**: get `topstories`, take the first n ids, build each story thread concurrently, collect the distinct `by` values across all nodes, then fetch every profile concurrently.
- **Concurrency**: a per-run semaphore (buffered chan, size `CaptureOptions.Concurrency`, default 16) held **only during the upstream call**, never while waiting on children, so recursion cannot deadlock. Fan-out uses `sync.WaitGroup`, and children are written by index so HN's ranked `kids` order is preserved.
- **Context**: honor `ctx`; the CLI wires SIGINT through `signal.NotifyContext` and a `--timeout` flag.
- **Missing detection** (each case becomes a Node with a Status and is appended to `Thread.Missing{ID, ParentID, Depth, Status, Reason}`):
  - `null` response → `not_found`
  - `deleted: true` → `deleted` (no author or text; still recurse into `kids` if present)
  - `dead: true` → `dead` (content kept, still recurse; listed because it's hidden on HN)
  - fetch failed after retries → `error` with the message; the capture continues (partial archive, not a total failure)
  - Stats: `expected_descendants`, `captured_comments`, `missing_count`, `max_depth`, `authors`.
- **Users**: dedupe with a map; a `null` profile or a failed fetch goes into `MissingUsers`.
- Top-story ids may include jobs or polls: capture them as-is with their type. Poll `parts` are out of scope.

**CaptureStory(id)**: same pipeline for one id. **GetUser(id)**: one profile (`ErrNotFound` if null). **Save**: delegates to `ArchiveWriter`.

## CLI
```
go run ./cmd/hncapture [capture] [--top 5] [--out ./output] [--concurrency 16] [--timeout 2m] [--format tree|json|none]
go run ./cmd/hncapture story <id> [same flags except --top]
go run ./cmd/hncapture user <id> [--format tree|json]
```
- `capture` is the default, so the single command is `go run ./cmd/hncapture`.
- It writes the JSON file (`--out ""` disables saving), prints the tree to stdout, and prints a summary to stderr.
- Exit codes: 0 on success, including partial captures; 1 on fatal errors; 2 on usage errors.

## Build order (TDD, one commit each)
1. `domain` types + `ports` interfaces. ✅
2. `service` tree builder against an in-memory fake `HNClient`: nesting, depth, kid order, deleted/dead/null/error nodes, missing list, stats.
3. `service` users + facade: dedupe, missing users, `CaptureTop(n)`, concurrency limit test (`-race`).
4. `hnapi` client with `httptest.Server`: decode, null→nil, unknown fields, retry on 5xx, ctx cancel.
5. `jsonfile` writer: temp dir, round-trip decode.
6. `cli`: dispatch with a fake Facade; renderer test on a fixed tree.
7. `cmd/hncapture/main.go` wiring, fix the Dockerfile (go 1.24, no go.sum, build `./cmd/hncapture`), README.md, CLAUDE.md commands, sample output under `output/`.

## Verification
- `go vet ./... && go test -race ./...` pass.
- `go run ./cmd/hncapture` against the live API: 5 stories, tree with depth and authors, JSON file written, `jq '.stories[].stats'` makes sense.
- `go run ./cmd/hncapture story 8863` and `go run ./cmd/hncapture user jl`.
- Optional: `/review` with the `code-reviewer` subagent before the final commit.
