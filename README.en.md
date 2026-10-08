<div align="center">

# typush

English | [简体中文](README.md)

</div>

typush is a CLI tool for developing, validating, and publishing [Typst](https://typst.app/) packages to the [Typst Universe](https://github.com/typst/packages).

The name combines Typst and push, referring to publishing packages to [universe](https://typst.app/universe/).

## Installation

```sh
go install github.com/Vncntvx/typush@latest
```

You can also download a prebuilt binary for your platform from [Releases](https://github.com/Vncntvx/typush/releases).

## Prerequisites

Commands like `publish universe` and `dev --check` rely on the [GitHub CLI](https://cli.github.com). Install `gh` and run `gh auth login` once; run `typush login universe` to verify your authentication status.

## Quick start

Initialize a new package:

```sh
typush init
```

Link the directory into `@preview` during local development:

```sh
typush dev
```

Publish to Universe:

```sh
typush publish universe
```

## Commands

```sh
typush --help
typush init [name]        # interactively create typst.toml, README.md, LICENSE, and entrypoint
typush bump [patch|minor|major|<ver>] # update version and sync package references in README
typush check [--local] [--no-compile]  # run package specifications and compiler checks
typush install <ns>       # install to a local namespace (such as @local)
typush download <repo> [-c ref] [-n ns] # clone and install a package from a git repository
typush dev [--check]      # link into @preview (optional remote conflict check)
typush dev list           # list active dev links in @preview and target paths
typush clean [package]    # remove dev symlinks from @preview
typush exclude <globs...> # add glob patterns to excluded files
typush login universe     # verify GitHub CLI authentication
typush publish universe [--dry-run]   # validate and submit a PR to Universe (handles fork and branch)
typush pr status [number|url]         # view PR details and review comments (matches current package by default)
typush pr checks [number|url] [-w]    # view or watch official Universe CI check runs
typush ci plan [--packages "a b"]     # scan workspace and output CI matrix JSON
typush ci generate [...]              # generate GitHub Actions CI release workflow
```

`publish` runs Universe checks locally before submitting. It catches missing metadata, invalid SPDX licenses, author formatting issues, forbidden file exclusions, and missing assets before opening a pull request.

## Local verification

When `typst` is found on your `PATH` (or specified by `TYPST_BIN`), `typush check` runs the official CI verification steps:

- Compile: for libraries, imports the package; for templates, runs `typst init @preview/<name>:<version>` in an isolated temporary directory and compiles the template entrypoint. Checks run in temporary directories without modifying local Typst cache. Use `--no-compile` to skip compilation, or `--local` to check only basic manifest rules.
- README: checks for missing image alt text, broken relative links, unsupported markdown features, and repository URLs pointing to mutable default branches.
- Files: flags bundled font files, unexcluded test/example files or large files, and unlinked manuals.
- Imports: flags relative imports of the entrypoint, outdated version references, and non-standard template imports.

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## License

[MIT](LICENSE)
