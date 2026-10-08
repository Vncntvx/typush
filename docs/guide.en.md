# typush User Guide

Complete command reference and workflow documentation for typush.

---

## Contents

1. [Prerequisites](#1-prerequisites)
2. [Initializing a Package (init)](#2-initializing-a-package-init)
3. [Local Development and Debugging (dev, clean)](#3-local-development-and-debugging-dev-clean)
4. [Local Verification (check)](#4-local-verification-check)
5. [Version Bumping (bump)](#5-version-bumping-bump)
6. [Publishing to Universe (publish)](#6-publishing-to-universe-publish)
7. [Tracking PRs and CI (pr)](#7-tracking-prs-and-ci-pr)
8. [Namespaces and Git Downloads (install, download)](#8-namespaces-and-git-downloads-install-download)
9. [Continuous Integration Helpers (ci)](#9-continuous-integration-helpers-ci)
10. [Metadata Inspection and Scripting (metadata)](#10-metadata-inspection-and-scripting-metadata)
11. [Package Directory Paths (path)](#11-package-directory-paths-path)

---

## 1. Prerequisites

typush is a single static binary without runtime dependencies. Several subcommands call external tools:

- **GitHub CLI (gh)**: Required by `publish universe`, `pr status/checks`, `login universe`, and `dev --check`. Install `gh` and run `gh auth login` once. typush does not store GitHub tokens; all GitHub operations are delegated through `gh`.
- **Typst (typst)**: Used by `typush check` to run compilation tests. If missing, compile tests are skipped with a warning. You can override the binary path with the `TYPST_BIN` environment variable.
- **Git (git >= 2.25)**: `publish universe` uses `sparse-checkout` to upload package files without cloning the entire packages repository.

---

## 2. Initializing a Package (init)

```sh
typush init [package-name]
```

Interactively creates a `typst.toml` manifest and starter files in the current directory.

### Automated Defaults

- Author: Reads `user.name` and `user.email` from git configuration, formatting as `Name <email>`.
- Repository URL: Reads `git remote get-url origin` and converts it to an HTTPS GitHub URL.
- Package Name: Defaults to the current directory name if not provided as an argument (validated for kebab-case).

### Generated Files

- `typst.toml`: Manifest containing package metadata, categories, disciplines, and license.
- `src/lib.typ`: Default entrypoint file if absent.
- `README.md`: Basic usage example.
- `LICENSE`: Created with the current year and author name if the license is MIT and no license file exists.

### Templates

When creating a template, the prompt asks for the template directory (default `template`) and entrypoint (default `main.typ`, relative to template path), and writes a `[template]` section into the manifest.

---

## 3. Local Development and Debugging (dev, clean)

```sh
typush dev [--check]
typush dev list
typush clean [package]
```

### How It Works

Typst discovers packages from its data directory under `@preview`. `typush dev` creates a symlink from your working directory into that location:

- **macOS**: `~/Library/Application Support/typst/packages/preview/<name>/<version>`
- **Linux**: `~/.local/share/typst/packages/preview/<name>/<version>`
- **Windows**: `%APPDATA%\typst\packages\preview\<name>\<version>`

Once linked, any local Typst document can import the package directly:

```typ
#import "@preview/<name>:<version>": *
```

Edits to the package source reflect immediately without copying files.

### Options

- `typush dev`: Creates the symlink. If the target directory exists as a non-symlink, the command aborts with an error.
- `typush dev --check`: Queries the official static CDN index (`packages.typst.org`) to check for package and version conflicts without consuming GitHub API quota (falls back to GitHub API on network errors).
- `typush dev list` (or `-l`): Lists all active dev symlinks under `@preview`, showing package name, version, status, and target path.
- `typush clean [package]`: Removes development symlinks. Cleans all links when run without arguments, or only the specified package.

### Windows Notes

Creating symlinks on Windows requires Developer Mode or Administrator privileges. If symlinks cannot be created, use `typush install local` to install to `@local` instead.

---

## 4. Local Verification (check)

```sh
typush check [--local] [--no-compile]
```

Runs the validation rules from the official bundler and package-check offline.

### Verification Rules

1. **Manifest**: Package name must match kebab-case; version must follow semantic versioning; license must be a valid SPDX expression; categories and disciplines must match the official list; unknown fields are rejected.
2. **Documentation and Links**: Images in `README.md` must have alt text; relative links must resolve to existing files; unsupported markdown extensions are flagged; repository URLs pointing to default branches trigger warnings.
3. **Files**: Bundled font files (`.ttf`, `.otf`, etc.) are rejected; unexcluded large or test files trigger warnings; checks that `README.md` and `LICENSE` are not excluded.
4. **Compilation**: When `typst` is on `PATH`, runs isolated compile checks in temporary directories. Libraries verify entrypoint imports; templates verify `typst init` and entrypoint compilation; verifies local compiler version against `package.compiler`.

### Flags

- `--local`: Validates only basic manifest fields (name, version, entrypoint presence), skipping submission rules.
- `--no-compile`: Skips compiler execution and runs static lint rules only.

---

## 5. Version Bumping (bump)

```sh
typush bump [patch|minor|major|<version>]
```

Increments or sets `package.version` in `typst.toml`.

### Workflow

1. Reads current version from `typst.toml`.
2. Computes target version using keywords (`patch`, `minor`, `major`) or an explicit version string. Prompts interactively if omitted, defaulting to the next patch version.
3. Verifies that the target version is strictly greater than the current version.
4. Writes updated version to `typst.toml`.

---

## 6. Publishing to Universe (publish)

```sh
typush publish universe [--dry-run]
```

Submits a package to the official `typst/packages` repository.

### Workflow

1. **Validation**: Runs `typush check`; any error aborts publication.
2. **Conflict Checks**: Checks if the version already exists or if a matching PR is already pending.
3. **Fork Management**: Checks if the user has a fork of `typst/packages` and prompts to create one if missing.
4. **Branch and Files**: Creates a `<name>-<version>` branch in the fork repository. Collects publishable files respecting `.typstignore`, `.gitignore`, and `package.exclude`.
5. **Sparse Checkout Upload**: Creates a shallow clone in a temporary directory, configures sparse checkout for `packages/preview/<name>/<version>`, commits the files, and pushes to the fork.
6. **Pull Request**: Opens a Draft PR on `typst/packages` with pre-filled checkboxes according to the official template.

### Flags

- `--dry-run`: Runs checks and displays the file list without modifying remote branches or opening a PR.

---

## 7. Tracking PRs and CI (pr)

```sh
typush pr status [number|url]
typush pr checks [number|url] [-w]
```

Inspects pull requests and CI status directly in your terminal. If the PR number or URL is omitted, typush resolves the open PR for the current package.

- `typush pr status`: Displays PR details and reviewer comments.
- `typush pr checks`: Displays official CI check runs. Use `-w` to wait until all runs complete.

---

## 8. Namespaces and Git Downloads (install, download)

```sh
typush install <namespace>
typush download <repository> [-c ref] [-n namespace]
```

### Install to Local Namespace

```sh
typush install local
```

Copies the package into `@local/<name>/<version>`. Documents can import it using `#import "@local/<name>:<version>": *`.

### Download from Git

```sh
typush download https://github.com/user/pkg -n local
```

Downloads a package from a Git repository into a specified namespace. Use `-c` to check out a specific tag, branch, or commit.

---

## 9. Continuous Integration Helpers (ci)

```sh
typush ci plan [--packages "pkg1 pkg2"]
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

Helpers for repositories maintaining multiple packages.

- `typush ci plan`: Scans the workspace and outputs a JSON matrix for GitHub Actions.
- `typush ci generate`: Generates a GitHub Actions workflow file that automates package publishing.

---

## 10. Metadata Inspection and Scripting (metadata)

```sh
typush metadata [field] [--json]
```

Inspect or extract `typst.toml` metadata directly from the command line.

### Overview

When run without arguments, prints a formatted summary of package metadata:

```sh
typush metadata
```

### Single Field Extraction (Scripting)

Pass a field name to output its raw value to stdout, ideal for shell scripts or CI pipelines:

```sh
VERSION=$(typush metadata version)
NAME=$(typush metadata name)
ENTRY=$(typush metadata entrypoint)
```

Supported fields: `name`, `version`, `entrypoint`, `authors`, `license`, `description`, `homepage`, `repository`, `compiler`, `categories`, `disciplines`, `keywords`, `exclude`.

### JSON Output

Pass `--json` to output structured metadata to stdout:

```sh
typush metadata --json
typush metadata authors --json
```

---

## 11. Package Directory Paths (path)

```sh
typush path [namespace]
```

Outputs the absolute filesystem path where Typst packages are stored locally.

- `typush path`: Shows the local packages root directory.
- `typush path preview` (or `typush path @preview`): Shows the `@preview` namespace directory.
- `typush path local` (or `typush path @local`): Shows the `@local` namespace directory.

Use in shell commands for quick navigation or inspection:

```sh
cd $(typush path)
ls -la $(typush path preview)
```

