# typush User Guide

Command reference and workflow documentation.

---

## Contents

1. [Installation and Prerequisites](#1-installation-and-prerequisites)
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
12. [Excluding Published Files (exclude)](#12-excluding-published-files-exclude)
13. [Package Search and Details (search, info)](#13-package-search-and-details-search-info)
14. [Dependency Checking and Updating (outdated, update)](#14-dependency-checking-and-updating-outdated-update)
15. [Installed Package Management (list, uninstall)](#15-installed-package-management-list-uninstall)
16. [Cloning Package Source (clone)](#16-cloning-package-source-clone)
17. [Shell Completions (completion)](#17-shell-completions-completion)

---

## 1. Installation and Prerequisites

### Installation

#### Homebrew (macOS / Linux)

```sh
brew install Vncntvx/tap/typush
```

#### Go Install

```sh
go install github.com/Vncntvx/typush@latest
```

#### Prebuilt Binaries

Download precompiled archives for your platform from [GitHub Releases](https://github.com/Vncntvx/typush/releases).

### Prerequisites

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
typush clean [package] [-n|--dry-run]
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
- `typush clean [package] [-n|--dry-run]`: Removes development symlinks. Cleans all links when run without arguments, or only the specified package. Use `-n` to preview links that would be removed without deleting them.

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
4. **Compilation**: When `typst` is on `PATH`, runs compile checks in isolated temporary directories. Libraries verify entrypoint imports; templates verify `typst init` and entrypoint compilation; verifies local compiler version against `package.compiler`.

### Flags

- `--local`: Validates only basic manifest fields (name, version, entrypoint presence), skipping submission rules.
- `--no-compile`: Skips compiler execution and runs static lint rules only.

---

## 5. Version Bumping (bump)

```sh
typush bump [patch|minor|major|<version>] [-i files] [-t tag] [-n|--dry-run]
```

Increments or sets `package.version` in `typst.toml`, with optional multi-file version synchronization.

### Workflow

1. Reads current version from `typst.toml`.
2. Computes target version using keywords (`patch`, `minor`, `major`) or an explicit version string. Prompts interactively if omitted, defaulting to the next patch version.
3. Verifies that the target version is strictly greater than the current version.
4. Updates version references in included files: when `-i` (or `--include`) is specified, updates version strings in the given files. By default it replaces `@preview/<name>:<current_ver>`. When `-t` (or `--tag`) is also set, it replaces versions wrapped in `<tag>` elements (e.g. `<version>0.1.0</version>` or `<!-- version -->0.1.0<!-- /version -->`).
5. Writes the updated manifest and included files. Pass `-n` (or `--dry-run`) to preview diffs across all files without modifying them.

---

## 6. Publishing to Universe (publish)

```sh
typush publish universe [-n|--dry-run]
```

Submits a package to the official `typst/packages` repository.

### Workflow

1. **Validation**: Runs `typush check`; any error aborts publication.
2. **Conflict Checks**: Checks if the version already exists or if a matching PR is already pending.
3. **Fork Management**: Checks if the user has a fork of `typst/packages` and prompts to create one if missing.
4. **Branch and Files**: Creates a `<name>-<version>` branch in the fork repository. Collects files to upload respecting `.typstignore` and `.gitignore`; `package.exclude` does not filter the upload list.
5. **Sparse Checkout Upload**: Creates a shallow clone in a temporary directory, configures sparse checkout for `packages/preview/<name>/<version>`, commits the files, and pushes to the fork.
6. **Pull Request**: Opens a Draft PR on `typst/packages` with pre-filled checkboxes according to the official template.

### Flags

- `-n` (or `--dry-run`): Runs checks and displays the file list without modifying remote branches or opening a PR.

---

## 7. Tracking PRs and CI (pr)

```sh
typush pr status [number|url]
typush pr checks [number|url] [-w]
```

Inspects pull requests and CI check runs in your terminal. If the PR number or URL is omitted, typush resolves the open PR for the current package.

- `typush pr status`: Displays PR details and reviewer comments.
- `typush pr checks`: Displays official CI check runs. Use `-w` to wait until all runs complete.

---

## 8. Namespaces and Git Downloads (install, download)

```sh
typush install <namespace> [-n|--dry-run]
typush download <repository> [-c ref] [-n namespace] [--subdir dir] [--dry-run]
```

### Install to Local Namespace

```sh
typush install local
```

Copies the package into `@local/<name>/<version>`. Documents can import it using `#import "@local/<name>:<version>": *`. Pass `-n` (or `--dry-run`) to preview the target directory and files to be installed without copying them.

A preview never reads stdin: the `@` prefix normalization, the `preview` namespace warning, and the overwrite confirmation are printed as notes.

### Download from Git

```sh
typush download https://github.com/user/pkg -n local
```

Clones a repository into a temporary directory and installs it into the target namespace.

- `-c <ref>`: Checks out a specific tag, branch, or commit.
- `-n <namespace>`: Target namespace (defaults to `local`).
- `--subdir <dir>`: Installs only the specified subdirectory of the repository (useful for monorepos).
- `--dry-run`: Previews the installation file list without copying files into the local packages directory.

---

## 9. Continuous Integration Helpers (ci)

```sh
typush ci plan [--packages "pkg1 pkg2"]
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...] [-n|--dry-run]
```

Helpers for repositories maintaining multiple packages.

- `typush ci plan`: Scans the workspace and outputs a JSON matrix for GitHub Actions.
- `typush ci generate`: Generates a GitHub Actions workflow file that automates package publishing. Pass `-n` (or `--dry-run`) to print the YAML to stdout without writing files.

---

## 10. Metadata Inspection and Scripting (metadata)

```sh
typush metadata [field] [--json]
```

Inspect or extract `typst.toml` metadata from the command line.

### Overview

When run without arguments, prints a formatted summary of package metadata:

```sh
typush metadata
```

### Single Field Extraction (Scripting)

Pass a field name to output its raw value to stdout, for shell scripts or CI pipelines:

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

Prints the absolute path where Typst stores local packages.

- `typush path`: Shows the local packages root directory.
- `typush path preview` (or `typush path @preview`): Shows the `@preview` namespace directory.
- `typush path local` (or `typush path @local`): Shows the `@local` namespace directory.

Use in shell commands for quick navigation or inspection:

```sh
cd $(typush path)
ls -la $(typush path preview)
```

---

## 12. Excluding Published Files (exclude)

```sh
typush exclude <globs...> [-n|--dry-run]
```

Writes glob patterns into the `package.exclude` field of `typst.toml` to exclude matching files during local installation. This field does not remove files from submissions to the Universe repository; use `.typstignore` or `.gitignore` to exclude files from uploads.

```sh
typush exclude '*.log' 'tests/**'
typush exclude -n '*.log'
```

- Patterns are matched against `/`-separated relative paths; patterns without `/` also match file or directory names at any depth.
- This is not full `.gitignore` syntax: use `tests/**`, not `tests/`, to exclude files inside a directory. Negated patterns starting with `!` are not supported.
- Patterns already present are not added twice; the command reports only the number of new patterns.
- `-n` (or `--dry-run`) prints the patterns that would be added and leaves `typst.toml` untouched. Patterns are still validated, so an invalid glob fails the same way in both modes.

---

## 13. Package Search and Details (search, info)

Search packages on Typst Universe and inspect their metadata and release history.

### Searching packages (search)

```sh
typush search <query> [--limit 20] [--json] [--refresh]
```

Searches across package names, prefixes, keywords, categories, descriptions, and authors. Results are ordered by relevance, showing only the newest version of each package.

The index is downloaded from the official CDN and cached locally (15-minute TTL). If the CDN is unavailable but a cached copy exists, the command falls back to the cache and logs a warning to stderr.

- `-l, --limit <n>`: Maximum number of results to display. Defaults to 20; a non-positive value falls back to the default.
- `--json`: Output search results with scores in JSON format.
- `--refresh`: Download a fresh index, ignoring the local cache.

```sh
typush search diagram
typush search cetz --limit 5
typush search math --json
```

### Viewing package details (info)

```sh
typush info <package>[:version] [--json] [--refresh]
```

Displays metadata for a package, including its description, license, authors, repository URL, compiler requirements, categories, keywords, and release history.

If no version is specified, it displays the latest release. You can also inspect an earlier version. The package may be written as `fletcher`, `@preview/fletcher`, or `preview/fletcher`, and is validated against the Universe rules. The output ends with an `#import` snippet.

- `--json`: Output package metadata in JSON format.
- `--refresh`: Refresh remote package metadata.

```sh
typush info fletcher
typush info fletcher:0.5.0
typush info @preview/cetz:0.2.2 --json
```

---

## 14. Dependency Checking and Updating (outdated, update)

Inspect and update `@preview` package imports across `.typ` files. Scanning skips files matched by `.typstignore` or `.gitignore`.

### Checking for updates (outdated)

```sh
typush outdated [path] [--json] [--refresh]
```

Scans `.typ` files in the given directory or file, finds all `@preview/<name>:<version>` imports, and checks them against the newest releases on Universe.

The output table lists each package's current versions, latest version, status, and source locations (collapsed to `(+N more)` beyond the first two). There are three statuses:

- `Up to date`: already on the newest release.
- `Update available`: a newer release exists.
- `Unknown (not in Universe)`: the package is absent from the Universe index (for example, a misspelled name).

If the index cannot be loaded and no cached copy exists, the command exits with an error.

- `[path]`: Directory or `.typ` file to scan (defaults to the current directory).
- `--json`: Output dependency status in JSON format; `status` is `up-to-date`, `outdated`, or `not-in-universe`.
- `--refresh`: Refresh the Universe index before comparing.

```sh
typush outdated
typush outdated main.typ
typush outdated ./chapters --json
```

### Updating dependencies (update)

```sh
typush update [package...] [-n|--dry-run] [-f|--file path] [--refresh]
```

Updates `@preview` package references in `.typ` files to their latest versions on Universe. Only the version string is replaced, preserving quotes, indentation, and line endings (LF or CRLF), including a missing final newline. Scanning reads each file once; a file that needs changes is then read once more and written once through an atomic same-directory replacement. A reference whose version is already newer than the index's latest is left untouched, so an update never lowers a version.

- `-n` (or `--dry-run`): Shows line-level diffs (`-` / `+`) without modifying files.
- `-f, --file <path>`: Update dependencies only in the specified file.
- `[package...]`: One or more package names to update. If omitted, updates all outdated dependencies. Typush warns if a requested package was not found during scanning.

```sh
# Preview update diffs without modifying files
typush update -n

# Update all outdated dependencies
typush update

# Update specific packages only
typush update cetz fletcher

# Update dependencies in one file only
typush update -f main.typ
```

---

## 15. Installed Package Management (list, uninstall)

Inspect and remove packages in your local Typst data directories.

### Listing installed packages (list)

```sh
typush list [namespace] [-a|--all] [-t|--tree] [--json]
```

Scans the local data directory and displays packages sorted by namespace and name in ascending order, and version in descending order (newest first).

- `[namespace]`: Limits output to a specific namespace (e.g. `local` or `preview`).
- `-a, --all`: Also includes downloaded cache packages (e.g. `@preview` packages downloaded during Typst compilation).
- `-t, --tree`: Displays packages in an ASCII tree hierarchy with source badges (installed copy, dev symlink, or cache).
- `--json`: Outputs structured package data in JSON format.

```sh
# List all locally installed packages
typush list

# Show full tree including download cache
typush list -a -t

# Filter by the @local namespace
typush list local
```

### Removing installed packages (uninstall)

```sh
typush uninstall <target> [-y|--force] [-n|--dry-run]
```

Removes packages from your local Typst data directory. When removing a version leaves the package directory empty, or removing a package leaves the namespace directory empty, the empty parent directories are automatically removed.

Target syntax:
- `@ns/pkg:ver`: Removes a specific version in a namespace.
- `@ns/pkg`: Removes all versions of a package in a namespace.
- `@ns`: Removes an entire namespace.
- `pkg:ver`: Removes a specific version from `@local`.
- `pkg`: Removes all versions of a package from `@local`.

Options:
- `-y, --force`: Skips interactive confirmation prompts.
- `-n, --dry-run`: Previews the target directory path and description without removing files.

```sh
# Remove a specific version
typush uninstall @local/my-lib:0.1.0

# Remove all versions of a local package
typush uninstall my-lib

# Remove an entire custom namespace without prompt
typush uninstall @custom -y
```

---

## 16. Cloning Package Source (clone)

```sh
typush clone <package> [destination] [-f|--force] [-n|--dry-run]
```

Downloads a package `.tar.gz` archive directly from the Typst Universe CDN (`packages.typst.org`) and extracts it locally. Path and symlink targets are validated during extraction to reject any directory traversal outside the target directory.

- `<package>`: Package specifier. Accepts `@preview/<name>:<version>`, `<name>:<version>`, or `<name>` (resolves latest version via the CDN index).
- `[destination]`: Destination directory. Defaults to `./<name>` if omitted.
- `-f, --force`: Overwrites existing non-empty destination directories without prompt.
- `-n, --dry-run`: Previews the version to clone and destination directory without downloading.

```sh
# Clone latest version of cetz into ./cetz
typush clone cetz

# Clone a specific version of fletcher into ./my-fletcher
typush clone fletcher:0.5.0 ./my-fletcher
```

---

## 17. Shell Completions (completion)

```sh
typush completion <bash|zsh|fish|powershell>
```

Generates shell completion scripts for typush to standard output.

### Shell Setup

#### Bash

```sh
# Current shell session:
source <(typush completion bash)

# Persist to system completion directory:
typush completion bash > /etc/bash_completion.d/typush
```

#### Zsh

```sh
# Write to a directory in your fpath:
typush completion zsh > "${fpath[1]}/_typush"
# Reload completions:
autoload -U compinit && compinit
```

#### Fish

```sh
# Current shell session:
typush completion fish | source

# Persist:
typush completion fish > ~/.config/fish/completions/typush.fish
```

#### PowerShell

```powershell
typush completion powershell | Out-String | Invoke-Expression
```
