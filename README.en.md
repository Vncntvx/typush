<div align="center">

# typush

English | [简体中文](README.md)

</div>

typush is a CLI tool for [Typst](https://typst.app/) package development and publishing: develop, validate, and install packages locally, then publish to the [Typst Universe](https://github.com/typst/packages).

The name comes from **Typst** + **push**: it pushes your packages into the [universe](https://typst.app/universe/).

## Installation

```sh
go install github.com/Vncntvx/typush@latest
```

You can also download a binary for your platform from [Releases](https://github.com/Vncntvx/typush/releases).

## Prerequisites

`publish universe` (and `dev --check`) access GitHub through the [GitHub CLI](https://cli.github.com). Install `gh` and run `gh auth login` once. `typush login universe` verifies that setup.

## Quick start

Initialize a new package (interactive):

```sh
typush init
```

While developing a template, symlink the package directory into the `@preview` namespace:

```sh
typush dev
```

Publish to the Universe (runs local checks first):

```sh
typush publish universe
```

## Commands

```sh
typush --help
typush init [name]        # interactive: creates typst.toml + entrypoint
typush check [--local] [--no-compile]  # validate the package, see below
typush install <ns>       # install to @<ns> (e.g. local)
typush download <repo> [-c ref] [-n ns]
typush dev [--check]      # symlink into @preview (+ optional Universe conflict check)
typush clean [package]
typush exclude <globs...>
typush login universe
typush publish universe [--dry-run]   # local Universe check + auto fork detection + sparse-checkout upload + draft PR
typush pr status [number|url]         # view PR details, reviewer comments and activity
typush pr checks [number|url] [-w]    # view or watch official Universe CI check runs
typush ci plan [--packages "a b"]     # workspace scan -> CI matrix JSON
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

`publish` runs the local Universe check first. Common rejection reasons from the official `bundler` CI (unknown fields, authors format, categories, SPDX license, missing README/LICENSE, template thumbnail, forbidden excludes) are caught before anything touches the network.

## Local verification

If `typst` is on your `PATH` (override with `TYPST_BIN`), `typush check` also runs what the official CI would run, locally and offline:

- Compile: smoke-imports the library; for templates, runs the official flow, `typst init @preview/<name>:<version>` into a temp project, then compiles the template entrypoint. Compiler errors fail the check; warnings are reported. Checks run in temp `HOME`/XDG dirs and do not touch your local Typst data. Use `--no-compile` to skip, `--local` for manifest-minimal rules only.
- README: missing image alt text (error), dead local links (error), GFM alerts/task lists, repo URLs pointing at the default branch (warning).
- Files: font files (error), `example`/`test` files and large files not excluded, ignored-but-present files, unlinked manuals.
- Imports: relative imports of the entrypoint, outdated self-version imports (README included), non-spec template imports.

## Design decisions

- Written in Go, built as a single static binary, released with GoReleaser.
- GitHub operations (auth checks, API queries, auto-forking, PR creation, CI monitoring) go through the `gh` CLI; the GitHub API is not called directly.
- Git network operations automatically bridge credentials via `gh auth git-credential`.
- `publish` uploads packages using single-commit sparse-checkout, which requires git >= 2.25.
- `check` covers the official `bundler` hard errors.
- `dev` skips the Universe network check by default; pass `--check` to enable it.
- `host` / `generate` are merged under the `ci` subcommand; the old names remain as hidden aliases.
- Interactive prompts read piped stdin, so `printf ... | typush init` works in scripts.
- GitHub access goes through the `gh` CLI; locally stored tokens are not read or written.

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## License

[MIT](LICENSE)
