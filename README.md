# typkg

The Typst package helper: develop locally, validate against Universe rules,
and publish to [Typst packages](https://github.com/typst/packages).

Go rewrite of [typush](https://github.com/Vncntvx/typush) — no Rust toolchain needed.

## Installation

```sh
go install github.com/Vncntvx/typkg/cmd/typkg@latest
```

Or download a release binary from GitHub Releases (Homebrew tap on the way).

## Notice

To use `publish universe`, generate a fine-grained token with the following
permissions to your fork of the packages repository:

- _Read_ access to _metadata_
- _Read and write_ access to _contents_
- _Read and write_ access to _workflow_

GitHub docs: <https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token>

Existing `typush` users: your token is migrated automatically from the old
config directory on first run.

## Usage

```sh
typkg --help
typkg init [name]        # interactive: creates typst.toml + entrypoint
typkg check [--local] [--no-compile]  # see below
typkg install <ns>       # install to @<ns> (e.g. local)
typkg download <repo> [-c ref] [-n ns]
typkg dev [--check]      # symlink into @preview (+ optional Universe conflict check)
typkg clean [package]
typkg exclude <globs...>
typkg login universe
typkg publish universe [--dry-run]   # local Universe check + sparse-checkout upload + draft PR
typkg ci plan [--packages "a b"]     # workspace scan -> CI matrix JSON
typkg ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

Deprecated aliases (still work): `typkg host` (= `ci plan`), `typkg generate` (= `ci generate`).

`publish` always runs the local Universe check first, so most rejections from
the official `bundler` CI (unknown fields, authors format, categories,
SPDX license, README/LICENSE, template thumbnail, forbidden excludes) are
caught before anything touches the network.

## Local verification (no CI needed)

If `typst` is on your `PATH` (override with `TYPST_BIN`), `typkg check`
additionally runs what the official CI would run, all locally and offline:

- **compile**: smoke-imports the library and, for templates, runs the
  official flow — `typst init @preview/<name>:<version>` into a temp
  project, then compiles the template entrypoint. Compiler errors fail
  the check; warnings are reported. Runs in isolated temp `HOME`/XDG
  dirs, so your real `~/.local/share` / `~/Library` stays untouched.
  Use `--no-compile` to skip, `--local` for manifest-minimal rules only.
- **README lint**: missing image alt text (error), dead local links
  (error), GFM alerts/task lists, default-branch repo URLs (warnings).
- **files lint**: font files (error), `example`/`test` files and large
  files not excluded, ignored-but-present files, unlinked manuals.
- **imports lint**: relative imports of the entrypoint, outdated
  self-version imports (README included), non-spec template imports.

## Differences from the Rust original

- Single static binary via Go, released with GoReleaser.
- GitHub API is hand-rolled `net/http` (no SDK); `publish` keeps the
  `sparse-checkout` path only — the slow per-file `api` upload was removed
  (requires git >= 2.25).
- `check` mirrors the official `bundler` hard errors instead of being a stub.
- `dev` skips the Universe network check by default; pass `--check` to enable.
- `host`/`generate` merged under `ci` (old names kept as hidden aliases).
- Interactive prompts read piped stdin too, so `printf ... | typkg init` works in scripts.
- Renamed from `typush` to `typkg`; old config (token) migrates automatically.

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## License

MIT — see the original project for full text.
