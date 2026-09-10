# Changelog fragments

`CHANGELOG.md` used to grow a `## Pending` section that every in-flight PR
edited directly. That meant any two PRs open at the same time collided on
the same few lines, and every merge forced the next PR to re-resolve the
conflict — annoying on its own, and it got worse the more PRs were open at
once.

Instead, each PR that needs a changelog entry adds **a new file** here
instead of editing `CHANGELOG.md`. Two PRs adding two different files can
never conflict with each other, no matter what order they merge in.

## Adding an entry

Create `changelog.d/<issue-or-pr-number>.md` with the same shape a normal
changelog section would have — one or more `### Added` / `### Changed` /
`### Fixed` / `### Removed` headings, each followed by one or more bullet
points:

```markdown
### Fixed

- `scout widgets list` no longer crashes on an empty response (#42)
```

A single fragment can include more than one heading if a PR touches more
than one kind of change. Reference the issue or PR number in the bullet
text the same way existing changelog entries do.

## Releasing

At release time, `scripts/assemble_changelog.py <version>` reads every
fragment in this directory, merges same-type sections together (in
Added / Changed / Fixed / Removed order), writes the result into
`CHANGELOG.md` under a new `## [<version>] - <date>` heading, and deletes
the fragment files it consumed. See `.claude/commands/release.md` for the
full release process.

Do not edit `CHANGELOG.md`'s `## Pending`-equivalent state directly for an
in-progress change — add a fragment here instead.
