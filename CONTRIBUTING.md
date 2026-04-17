# Contributing

## Cutting a release

1. Update `CHANGELOG.md`: move `[Unreleased]` entries under a new `[vX.Y.Z] — YYYY-MM-DD` heading, then add a fresh empty `[Unreleased]` section at the top. Update the compare/tag links at the bottom of the file.
2. Commit: `chore: release vX.Y.Z`.
3. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`.
4. Push: `git push origin main vX.Y.Z`.
5. Confirm the release appears at https://github.com/sebjacobs/jotter/releases within a few minutes. Four archives (darwin×2, linux×2) and a `checksums.txt` should be attached.

## Versioning

Follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Pre-1.0: minor bumps for new features, patch bumps for fixes. No strict breaking-change guarantees — the `0.x` prefix signals the CLI surface may still shift. Post-1.0: strict semver.

## Pre-release sanity check

Before cutting a release, validate the GoReleaser config locally:

```bash
goreleaser release --snapshot --clean
```

This builds all four target archives into `dist/` without publishing. If the snapshot succeeds, the tagged release will too.
