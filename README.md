# typush-go

A tool for [Typst](https://typst.app/) package development and publishing.
Go rewrite of [typush](https://github.com/Vncntvx/typush) — no Rust toolchain needed.

_The name `typush` is a portmanteau of **Typst** and **spaceship**, since it sends packages to the **[universe](https://typst.app/universe/)**._

## Installation

```sh
go install github.com/Vncntvx/typush-go/cmd/typush@latest
```

Or download a release binary from GitHub Releases (Homebrew tap on the way).

## Notice

To use `publish universe`, generate a fine-grained token with the following
permissions to your fork of the packages repository:

- _Read_ access to _metadata_
- _Read and write_ access to _contents_
- _Read and write_ access to _workflow_

GitHub docs: <https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token>

## Usage

```sh
typush --help
typush init [name]        # interactive: creates typst.toml + entrypoint
typush check              # validate the package (now a real check, not a stub)
typush install <ns>       # install to @<ns> (e.g. local)
typush download <repo> [-c ref] [-n ns]
typush dev [--check]      # symlink into @preview (+ optional Universe conflict check)
typush clean [package]
typush exclude <globs...>
typush login universe
typush publish universe [--dry-run]   # sparse-checkout upload only
typush ci plan [--packages "a b"]     # workspace scan -> CI matrix JSON
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

Deprecated aliases (still work): `typush host` (= `ci plan`), `typush generate` (= `ci generate`).

## Differences from the Rust original

- Single static binary via Go, released with GoReleaser (5 targets).
- GitHub API is hand-rolled `net/http` (no SDK); `publish` keeps the
  `sparse-checkout` path only — the slow per-file `api` upload was removed
  (requires git >= 2.25).
- `check` is now a real validation instead of a stub.
- `dev` skips the Universe network check by default; pass `--check` to enable.
- `host`/`generate` merged under `ci` (old names kept as hidden aliases).
- Interactive prompts read piped stdin too, so `printf ... | typush init` works in scripts.

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## License

MIT — see the original project for full text.
