# Jotter roadmap

Living document — Now / Next / Later priorities for jotter.

## Now

_Nothing in flight._

## Next

_Nothing queued._

## Later

### Pluggable storage backends

Today jotter writes JSONL files directly and commits them to a git-backed data dir. That choice couples three concerns: local file layout, query mechanism, and replication. Splitting them would let others plug in alternative backends.

**Target shape:** a `Storage` interface with `Append(Entry)`, `Tail(project, branch, limit)`, `List(project)`, `Search(filters)`. The current JSONL+git implementation becomes one concrete type; callers in `cmd/` depend only on the interface.

**Candidates worth supporting:**

- **SQLite (local)** — cheapest second backend to prove the interface with. Drops per-write git overhead, gains indexed search, keeps local-first. Open question: whether to retain a git sync layer on top or drop it.
- **D1 (remote SQLite)** — same API as SQLite over HTTP. Gains multi-device sync, loses local-first; network dependency on every write.
- **Kafka** — **not a peer backend**. Append-friendly but not queryable (`Tail`, `--since`, `--type` all become stream-and-filter or require a companion read model). Better modelled as an export target: primary store + `jotter export kafka` or a sidecar shipper.

**Prep work that pays off regardless of whether this lands:**

1. Audit `cmd/` to ensure nothing touches `os` / `filepath` directly — everything should route through `internal`. Mostly true today; worth confirming.
2. Move `internal.GitCommit` / `GitPush` calls out of `cmd/write.go` and into the storage layer. Any non-local backend will not want per-write git commits, and this coupling will leak through any extracted interface if left alone.

**When to actually do it:** only once a concrete second backend is committed to. Extracting an interface with one implementation tends to encode the current impl's shape (per-write git commits, branch-name sanitisation as a filename concern) rather than a genuinely portable contract.
