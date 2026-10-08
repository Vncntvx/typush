# AGENTS.md

`typush` — a Go CLI (module `github.com/Vncntvx/typush`) that develops, validates, and
publishes [Typst](https://typst.app) packages to [Typst Universe](https://github.com/typst/packages).
Single binary, flat five-package layout; no workspace, no codegen.

## Verify before you commit

CI is `.github/workflows/test.yml` and it runs exactly this, in this order:

```sh
test -z "$(gofmt -l .)"   # must print nothing
go vet ./...
go test ./...
go build ./...
```

- Single package: `go test ./checker/`
- Single test: `go test ./commands/ -run TestSubmissionPRBody`
- `go test ./...` takes <2s, needs no network, and needs no services.
- `go.mod` declares `go 1.25.0`; the `go-version: ">=1.23"` in the workflows is stale.
  Treat `go.mod` as the source of truth.
- `.golangci.yml` exists (govet, ineffassign, staticcheck, unused, gofmt, misspell,
  unconvert, unparam) but golangci-lint is **not wired into CI** — run it locally if you
  touch a lot of code.
- `.goreleaser.yaml` re-runs `go mod tidy`, `go vet`, `go test` in `before.hooks`, so a
  dirty suite breaks releases too.

## Runtime prerequisites

Nothing is needed for `build`/`test`/`vet`. The CLI itself shells out to:

- `typst` on `PATH` (or `TYPST_BIN`) — enables the real compile checks. Absent binary only
  downgrades `typush check` to a warning, it does not fail.
- `gh` CLI, authenticated (`gh auth login` or `GH_TOKEN`) — required by
  `publish universe`, `pr status/checks`, and `login universe` (also used as fallback by
  `dev --check`).
  **No GitHub token is ever stored by typush**; `commands/gh.go` documents why.
- `git >= 2.25` — `publish universe` uploads through `sparse-checkout --cone`.
- `TYPST_BIN` also lets you point the checker at a specific typst build.

## Where to change things

| Directory | Responsibility |
| --- | --- |
| `main.go` | Cobra command wiring only. Every command's `RunE` is one call into `commands`. |
| `commands/` | One file per action (`dev.go`, `install.go`, …), each taking an explicit `dir string`. |
| `manifest/` | `typst.toml` parsing + Universe validation rules (kebab-case, SPDX, categories, thumbnail). |
| `checker/` | Validation pipeline for a package dir. (Package doc comment still says `checkpkg` — stale.) |
| `util/` | Walker (gitignore semantics), Typst data-dir resolution, stdin prompts. |

Adding a command = a function in `commands/` plus a `newXxxCmd()` registered in
`NewRoot()` (`main.go`).

## Invariants that get broken by "cleaning up"

- **`submission.prBody()` must stay byte-identical** to the previously accepted
  typst/packages submissions: CRLF line endings, pre-checked boxes, and the "name
  Explanation" block omitted. It is locked down by `commands/universe_internal_test.go`.
  Diff against the upstream `.github/pull_request_template.md` before touching it.
- **`ghDirNames` must keep using the Git Trees API**, not the Contents API. The Contents
  API caps directory listings at 1000 entries and `packages/preview` exceeds that, so
  updates silently look like new packages.
- **stderr vs stdout:** progress, warnings and diagnostics go to `fmt.Fprintln(os.Stderr, …)`;
  only actual command output goes to stdout (`ci plan` JSON, `dev list` table, `exclude`
  count). CI consumes `ci plan` on stdout, so never pollute it.
- **One shared stdin reader.** `util.Confirm` / `PromptLine` / `MultiSelect` all read a
  package-level `*bufio.Reader` over `os.Stdin`. Creating a second reader swallows piped
  input between prompts. EOF falls back to the prompt's default.
- **Walker semantics are a port of the Rust original**, deliberately gitignore-shaped:
  `.typstignore` then `.gitignore` are read per directory (nested), dotfiles are skipped,
  `.git` is pruned, dotfiles themselves are never returned, and `package.exclude` globs
  apply only in the install path. Don't "fix" the recursion.
- `typush host` and `typush generate` are hidden back-compat aliases for `ci plan` /
  `ci generate`. Keep them working.

## Testing quirks

- Tests are external packages (`package checker_test`, `package util_test`). Use an
  internal test package only when you need unexported symbols, as
  `commands/universe_internal_test.go` does.
- `testdata/walker_test/` is a golden fixture located via `runtime.Caller`. Adding a file
  to it breaks the exact-match assertions in `util/walker_test.go`
  (`TestPublish` / `TestInstall`); update both expectations and the fixture.
- `checker` compile tests self-skip when `typst` is not on `PATH`, so a green
  `go test ./...` does **not** mean the compile paths were exercised. `typst` is installed
  on this machine (`/opt/homebrew/bin/typst`), so they do run here.
- Compile checks are fully isolated: `checker/compile.go` builds a temp `HOME`/XDG tree,
  symlinks the package into `@preview/<name>/<version>`, and never touches the real Typst
  data dir. You do **not** need `typush dev` before running `typush check`.

## Commit conventions

Commits must follow the Conventional Commits specification with an emoji immediately after the type prefix, followed optionally by a blank line and a bulleted list for detailed messages:

```text
<type>: :<emoji>: <description>

- <detail message 1>
- <detail message 2>
```

**Strict rule:** Do NOT include any additional trailing descriptive content, notes, metadata footers, or commentary (e.g., absolutely no `note: ...`, sign-offs, or explanatory prose). Only the header line and the optional bullet list are permitted.

Common mappings:
- `feat: :sparkles: <description>` — New feature
- `fix: :bug: <description>` — Bug fix
- `docs: :memo: <description>` — Documentation changes
- `refactor: :recycle: <description>` — Refactoring without behavior change
- `test: :white_check_mark: <description>` — Adding or updating tests
- `chore: :wrench: <description>` — Toolchain, dependencies, or maintenance
- `ci: :construction_worker: <description>` — CI/CD workflows and release automation
- `perf: :zap: <description>` — Performance improvements

## Docs and release

- `README.md` (中文) and `README.en.md` (English) mirror each other; edit both.
- `typush bump` updates `package.version` in `typst.toml` only.
- `commands/release-typst.yml` is `go:embed`ed and written into *user* package repos by
  `ci generate`. Keep its `<<source>>` / `<<destination>>` / `<<push-to-fork>>` placeholders
  working in both quoted and bare forms.
- Releases are goreleaser on `v*` tags; the binary is named `typush` and `main.version` is
  injected via ldflags. The Homebrew tap is intentionally commented out.

### Release notes format

GitHub Releases must follow a concise, technical, and factual structure. Do not use emojis, sales/marketing wording, or exclamation marks.

Template:

```markdown
<一句话说明版本定位与重点变更>

## 主要变动

- **<模块或功能名>**：客观说明具体变更与参数行为。

## 安装

```sh
go install github.com/Vncntvx/typush@<version>
```

或直接下载下方 Assets 中的预编译二进制。

## 致谢（可选）

感谢 @<contributor> 贡献的 #<PR号>。
```

