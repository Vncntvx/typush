# typush

Push your Typst package to the Universe: develop locally, validate against
Universe rules, and publish to [Typst packages](https://github.com/typst/packages).

Go rewrite of [typush](https://github.com/Vncntvx/typush) — no Rust toolchain needed.

## Installation

```sh
go install github.com/Vncntvx/typush@latest
```

Or download a release binary from GitHub Releases (Homebrew tap on the way).

## Notice

`publish universe` (and `dev --check`) work through the [GitHub CLI](https://cli.github.com):
install `gh` and run `gh auth login` once — no token juggling inside typush.
`typush login universe` just verifies that setup for you.

## Usage

```sh
typush --help
typush init [name]        # interactive: creates typst.toml + entrypoint
typush check [--local] [--no-compile]  # see below
typush install <ns>       # install to @<ns> (e.g. local)
typush download <repo> [-c ref] [-n ns]
typush dev [--check]      # symlink into @preview (+ optional Universe conflict check)
typush clean [package]
typush exclude <globs...>
typush login universe
typush publish universe [--dry-run]   # local Universe check + sparse-checkout upload + draft PR
typush ci plan [--packages "a b"]     # workspace scan -> CI matrix JSON
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

Deprecated aliases (still work): `typush host` (= `ci plan`), `typush generate` (= `ci generate`).

`publish` always runs the local Universe check first, so most rejections from
the official `bundler` CI (unknown fields, authors format, categories,
SPDX license, README/LICENSE, template thumbnail, forbidden excludes) are
caught before anything touches the network.

## Local verification (no CI needed)

If `typst` is on your `PATH` (override with `TYPST_BIN`), `typush check`
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
- Interactive prompts read piped stdin too, so `printf ... | typush init` works in scripts.
- Renamed from `typush` to `typush`; GitHub access now goes through `gh` (stored tokens are no longer used).

## Development

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## License

MIT — see the original project for full text.
