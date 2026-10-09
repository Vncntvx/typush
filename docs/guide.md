# typush 使用指南

typush 的命令参考与使用说明。

---

## 目录

1. [安装与环境准备](#1-安装与环境准备)
2. [初始化新包 (init)](#2-初始化新包-init)
3. [本地开发与调试 (dev, clean)](#3-本地开发与调试-dev-clean)
4. [本地规范校验 (check)](#4-本地规范校验-check)
5. [版本递增 (bump)](#5-版本递增-bump)
6. [发布至 Universe (publish)](#6-发布至-universe-publish)
7. [跟踪审查与 CI (pr)](#7-跟踪审查与-ci-pr)
8. [本地命名空间安装与下载 (install, download)](#8-本地命名空间安装与下载-install-download)
9. [持续集成辅助 (ci)](#9-持续集成辅助-ci)
10. [元数据查询与脚本支持 (metadata)](#10-元数据查询与脚本支持-metadata)
11. [数据目录路径 (path)](#11-数据目录路径-path)
12. [排除发布文件 (exclude)](#12-排除发布文件-exclude)
13. [包检索与信息查询 (search, info)](#13-包检索与信息查询-search-info)
14. [依赖检查与更新 (outdated, update)](#14-依赖检查与更新-outdated-update)
15. [已安装包管理 (list, uninstall)](#15-已安装包管理-list-uninstall)
16. [Universe 官方包克隆 (clone)](#16-universe-官方包克隆-clone)
17. [Shell 补全脚本 (completion)](#17-shell-补全脚本-completion)

---

## 1. 安装与环境准备

### 安装方式

#### Homebrew (macOS / Linux)

```sh
brew install Vncntvx/tap/typush
```

#### Go Install

```sh
go install github.com/Vncntvx/typush@latest
```

#### 预编译二进制

可直接从 [GitHub Releases](https://github.com/Vncntvx/typush/releases) 下载对应平台的预编译二进制文件。

### 运行时依赖

typush 是单一静态二进制，没有运行时依赖。部分子命令依赖以下工具：

- **GitHub CLI (gh)**：用于 `publish universe`、`pr status/checks`、`login universe` 与 `dev --check`。安装后执行一次 `gh auth login` 即可。typush 不会保存或记录 GitHub Token，所有相关网络操作通过 `gh` 完成。
- **Typst (typst)**：用于 `typush check` 运行编译测试。如果未安装，校验会跳过编译检查并输出提示。可用环境变量 `TYPST_BIN` 指定特定的 typst 二进制路径。
- **Git (git >= 2.25)**：用于 `publish universe` 的 `sparse-checkout` 上传，仅同步目标包路径，不需要拉取完整的 packages 仓库。

---

## 2. 初始化新包 (init)

```sh
typush init [package-name]
```

在当前目录下以交互方式生成 `typst.toml` 清单与初始代码。

### 自动推断

- 作者信息：读取 git 配置中的 `user.name` 与 `user.email`，格式为 `Name <email>`。
- 仓库地址：读取 `git remote get-url origin` 并转换为 HTTPS 链接。
- 包名：若未提供参数，默认使用当前目录名（需满足 kebab-case 规范）。

### 生成文件

- `typst.toml`：包含包名、版本、作者、分类、学科、许可证等元数据。
- `src/lib.typ`：默认入口文件（若不存在）。
- `README.md`：包含包名与使用示例。
- `LICENSE`：若许可证选择 MIT 且当前目录无协议文件，会自动生成带有当前年份与作者的 MIT 协议。

### 模板支持

创建模板包时，会询问模板目录（默认 `template`）与入口文件（默认 `main.typ`，相对于模板目录），并在清单中写入 `[template]` 配置。

---

## 3. 本地开发与调试 (dev, clean)

```sh
typush dev [--check]
typush dev list
typush clean [package] [-n|--dry-run]
```

### 工作原理

Typst 运行时会从系统的本地数据目录读取 `@preview` 命名空间下的包。`typush dev` 将当前项目目录软链接到该位置：

- **macOS**：`~/Library/Application Support/typst/packages/preview/<name>/<version>`
- **Linux**：`~/.local/share/typst/packages/preview/<name>/<version>`
- **Windows**：`%APPDATA%\typst\packages\preview\<name>\<version>`

链接建立后，本地其他 Typst 文档可以直接引用该包：

```typ
#import "@preview/<name>:<version>": *
```

修改源码后保存即可看到效果，不需要重新安装。

### 命令选项

- `typush dev`：在 `@preview` 下创建软链接。如果目标路径已存在同名目录且不是软链接，会中止操作。
- `typush dev --check`：建立链接前优先通过官方静态 CDN 索引（`packages.typst.org`）排查线上包名与版本冲突（无需认证与 GitHub API 配额，失败时自动回退到 GitHub API）。
- `typush dev list`（或 `-l`）：列出当前所有已链接的包、版本、状态与指向路径。
- `typush clean [package] [-n|--dry-run]`：清理软链接。不带参数时清理全部开发软链接，指定名称时仅清理该包。加上 `-n` 可预览将清理的软链接清单而不实际删除。

### Windows 说明

Windows 创建符号链接需要开启系统的开发者模式或使用管理员权限。如果不方便开启，可以使用 `typush install local` 安装到 `@local` 命名空间。

---

## 4. 本地规范校验 (check)

```sh
typush check [--local] [--no-compile]
```

运行官方 bundler 与 package-check 的检查规则，在本地排查合规与编译问题。

### 检查规则

1. **元数据**：包名必须满足 kebab-case；版本号遵循语义化版本；许可证必须是有效的 SPDX 表达式；分类与学科必须在官方允许列表中；禁止包含未知字段。
2. **文档与链接**：README 中的图片必须有 alt 文本；相对链接引用的文件必须存在；不支持的 GFM 扩展会被标出；仓库地址指向默认分支时会输出警告。
3. **文件规范**：禁止携带 `.ttf`、`.otf` 等字体文件；检查是否遗漏排除了样例或测试文件；确认 `README.md` 与 `LICENSE` 没有被 exclude 排除。
4. **编译检查**：当 PATH 中存在 `typst` 时，在隔离临时目录中运行测试。普通包验证入口文件能否成功导入；模板包验证 `typst init` 与模板入口编译；对比本地编译器版本是否满足 `package.compiler` 要求。

### 命令选项

- `--local`：仅检查名称、版本与入口文件是否存在，跳过 Universe 准入规则。
- `--no-compile`：跳过编译器测试，仅运行静态规则检查。

---

## 5. 版本递增 (bump)

```sh
typush bump [patch|minor|major|<version>] [-i files] [-t tag] [-n|--dry-run]
```

递增或指定 `typst.toml` 中的 `package.version`，并支持同步更新其它文件中的版本号。

### 执行逻辑

1. 读取 `typst.toml` 中的当前版本。
2. 计算新版本：支持 `patch`、`minor`、`major` 关键字或直接指定目标版本号。未提供参数时通过终端交互选择，默认推荐下一个 patch 版本。
3. 防降级检查：目标版本必须高于当前版本。
4. 替换附加文件中的版本号：若指定了 `-i`（或 `--include`），会同步更新指定文件中的版本字符串。默认匹配 `@preview/<name>:<current_ver>` 格式；若同时指定了 `-t`（或 `--tag`），则匹配由指定标签包裹的内容，例如 `<version>0.1.0</version>` 或 `<!-- version -->0.1.0<!-- /version -->`。
5. 写回 `typst.toml` 与各更新文件。加上 `-n`（或 `--dry-run`）时以 diff 形式预览所有文件中的变更行，不修改任何文件。

---

## 6. 发布至 Universe (publish)

```sh
typush publish universe [-n|--dry-run]
```

向官方 `typst/packages` 仓库提交包发布请求。

### 发布流程

1. **运行本地校验**：执行 `typush check`，出现错误时直接终止。
2. **远端排重**：检查官方仓库是否已存在该版本，检查是否有正在开放的同名 PR。
3. **Fork 处理**：检查当前 GitHub 账号是否已有 `typst/packages` 的 fork。若无，交互提示后自动创建。
4. **分支与文件过滤**：在 fork 仓库基于 `upstream/main` 创建 `<name>-<version>` 分支。根据 `.typstignore` 与 `.gitignore` 收集待上传文件；`package.exclude` 不过滤上传清单。
5. **sparse-checkout 上传**：在临时目录浅克隆 fork 仓库并配置 sparse-checkout，仅同步 `packages/preview/<name>/<version>` 目录，提交单个 Commit 并推送。
6. **创建 PR**：按官方模板预选对应检查项，在 `typst/packages` 创建 Draft Pull Request 并返回链接。

### 命令选项

- `-n`（或 `--dry-run`）：依次执行检查并打印待上传文件列表，不修改远程分支也不创建 PR。

---

## 7. 跟踪审查与 CI (pr)

```sh
typush pr status [number|url]
typush pr checks [number|url] [-w]
```

在终端查看发布 PR 的审查意见与官方 CI 运行状态。在包目录下运行时，若省略 PR 编号或 URL，会自动根据当前包名匹配对应的开放 PR。

- `typush pr status`：查看 PR 概述与审查评论。
- `typush pr checks`：查看官方 CI 检查状态。加上 `-w` 可持续等待直到全部检查结束。

---

## 8. 本地命名空间安装与下载 (install, download)

```sh
typush install <namespace> [-n|--dry-run]
typush download <repository> [-c ref] [-n namespace] [--subdir dir] [--dry-run]
```

### 安装到本地命名空间

```sh
typush install local
```

将当前包以文件副本形式安装到系统 Typst 目录的 `@local/<name>/<version>` 下，供本地使用 `#import "@local/<name>:<version>": *` 引用。加上 `-n`（或 `--dry-run`）可预览安装目标路径与待复制文件清单，不向磁盘写入任何文件。

预览模式不读取标准输入：`@` 前缀规范化、`preview` 命名空间警告与覆盖确认都以提示形式打印。

### 从 Git 仓库下载

```sh
typush download https://github.com/user/pkg -n local
```

从远程 Git 仓库下载包并安装到指定命名空间。

- `-c <ref>`：指定分支、tag 或 commit。
- `-n <namespace>`：安装到的目标命名空间（默认 `local`）。
- `--subdir <dir>`：仅安装仓库中的指定子目录（适用于多包 Monorepo 仓库）。
- `--dry-run`：克隆到临时目录后仅预览待复制的文件列表，不写入本地包目录。

---

## 9. 持续集成辅助 (ci)

```sh
typush ci plan [--packages "pkg1 pkg2"]
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...] [-n|--dry-run]
```

用于在多包仓库（Monorepo）中配合 GitHub Actions 自动化发布。

- `typush ci plan`：扫描目录并输出供 GitHub Actions Matrix 使用的 JSON 数据。
- `typush ci generate`：生成自动化发布工作流文件。加上 `-n` 可将生成的 YAML 输出到 stdout 供管道使用，不创建文件。

---

## 10. 元数据查询与脚本支持 (metadata)

```sh
typush metadata [field] [--json]
```

在终端查看，或在脚本中提取 `typst.toml` 的元数据。

### 查看概览

不带参数执行时，格式化输出当前包的全部元数据：

```sh
typush metadata
```

### 单字段提取（脚本友好）

传入字段名直接输出该字段的纯文本值，适合在 Shell 脚本或 CI 中捕获变量：

```sh
VERSION=$(typush metadata version)
NAME=$(typush metadata name)
ENTRY=$(typush metadata entrypoint)
```

支持字段：`name`、`version`、`entrypoint`、`authors`、`license`、`description`、`homepage`、`repository`、`compiler`、`categories`、`disciplines`、`keywords`、`exclude`。

### JSON 输出

加上 `--json` 标志将元数据以 JSON 格式输出到 stdout，可配合 `jq` 等工具解析：

```sh
typush metadata --json
typush metadata authors --json
```

---

## 11. 数据目录路径 (path)

```sh
typush path [namespace]
```

输出本机 Typst 本地包的目录绝对路径。

- `typush path`：输出本地包根目录（即 packages 存储路径）。
- `typush path preview`（或 `typush path @preview`）：输出 `@preview` 命名空间路径。
- `typush path local`（或 `typush path @local`）：输出 `@local` 命名空间路径。

结合 Shell 命令快速导航或管理：

```sh
cd $(typush path)
ls -la $(typush path preview)
```

---

## 12. 排除发布文件 (exclude)

```sh
typush exclude <globs...> [-n|--dry-run]
```

将 glob 模式写入 `typst.toml` 的 `package.exclude`，本地安装时排除匹配的文件。此字段不会从提交到 Universe 仓库的文件清单中移除文件；如需排除上传文件，请使用 `.typstignore` 或 `.gitignore`。

```sh
typush exclude '*.log' 'tests/**'
typush exclude -n '*.log'
```

- 模式按 `/` 分隔的相对路径匹配；不含 `/` 的模式也匹配任意层级的文件或目录名。
- 这不是完整的 `.gitignore` 语法：排除目录内文件请使用 `tests/**`，不要使用 `tests/`；不支持 `!` 否定模式。
- 已存在的模式不会重复添加，命令只输出实际新增的数量。
- `-n`（或 `--dry-run`）只打印待新增的模式，不修改 `typst.toml`；模式本身仍会被校验，非法 glob 在两种模式下都会报错。

---

## 13. 包检索与信息查询 (search, info)

在终端中检索 Typst Universe 的包并查看其元数据与历史版本。

### 线上检索 (search)

```sh
typush search <query> [--limit 20] [--json] [--refresh]
```

根据包名、前缀、关键字、分类、描述与作者进行加权搜索。结果会按相关度降序排列，每个包仅展示最新版本。

索引从官方 CDN 获取并缓存在本地（有效期 15 分钟）。若网络不可用但存在本地缓存，则回退到缓存并在 stderr 打印警告。

- `-l, --limit <n>`：限制返回条数，默认 20；传入非正数时回退为默认值。
- `--json`：以 JSON 格式输出包含匹配分数的搜索结果。
- `--refresh`：忽略本地缓存，重新从官方 CDN 获取最新索引。

```sh
typush search diagram
typush search cetz --limit 5
typush search math --json
```

### 查看包详情 (info)

```sh
typush info <package>[:version] [--json] [--refresh]
```

查看 Universe 包的详细元数据，包括描述、许可证、作者、代码仓库、最低编译器版本、分类、标签，以及全部历史发布版本。

未指定版本时默认显示最新版本；也可以查询指定历史版本。包名可写作 `fletcher`、`@preview/fletcher` 或 `preview/fletcher`，并会按 Universe 规则校验。输出末尾会提供对应的 `#import` 语句。

- `--json`：以 JSON 格式输出包详情。
- `--refresh`：重新从官方 CDN 获取最新索引。

```sh
typush info fletcher
typush info fletcher:0.5.0
typush info @preview/cetz:0.2.2 --json
```

---

## 14. 依赖检查与更新 (outdated, update)

管理本地 `.typ` 文件中引用的 `@preview` 包依赖。扫描会跳过 `.typstignore` 与 `.gitignore` 匹配的文件。

### 检查依赖版本 (outdated)

```sh
typush outdated [path] [--json] [--refresh]
```

扫描指定目录（或单个文件）下的所有 `.typ` 文件，提取 `@preview/<name>:<version>` 导入，并与 Universe 上的最新版本比对。

输出表格包含当前引用版本、最新版本、状态，以及出现的文件与行号（超过两处时折叠为 `(+N more)`）。状态有三种：

- `Up to date`：已是最新版本。
- `Update available`：存在更新版本。
- `Unknown (not in Universe)`：该包在 Universe 索引中不存在（例如包名拼写错误）。

若索引无法加载且无本地缓存，命令直接报错。

- `[path]`：指定扫描的目录或单个 `.typ` 文件路径（默认为当前目录）。
- `--json`：以 JSON 格式输出依赖状态，`status` 为 `up-to-date`、`outdated` 或 `not-in-universe`。
- `--refresh`：重新拉取 Universe 索引进行比对。

```sh
typush outdated
typush outdated main.typ
typush outdated ./chapters --json
```

### 更新依赖版本 (update)

```sh
typush update [package...] [-n|--dry-run] [-f|--file path] [--refresh]
```

将 `.typ` 文件中的 `@preview` 包版本更新为 Universe 最新版本。更新时只替换版本号字符串，保留原有的单双引号、缩进与换行风格（LF 或 CRLF），包括文件末尾缺少换行的情况。扫描时每个文件读取一次；需要修改的文件在写入阶段只读一次、只写一次（通过同目录临时文件原子替换）。文件中已有比索引最新版本更新的版本时保持不变，因此 `update` 只会向前推进版本，不会降级。

- `-n`（或 `--dry-run`）：预览每个文件的版本变更差异（`-` / `+`），不修改文件。
- `-f, --file <path>`：仅更新指定 `.typ` 文件内的依赖。
- `[package...]`：指定待更新的包名；省略时更新所有过时依赖。若指定的包未在扫描文件中出现，会输出警告。

```sh
# 预览更新差异
typush update -n

# 更新全部过时依赖
typush update

# 仅更新指定包
typush update cetz fletcher

# 仅更新单个文件内的依赖
typush update -f main.typ
```

---

## 15. 已安装包管理 (list, uninstall)

查看与清理本地 Typst 数据目录中的包。

### 列出已安装包 (list)

```sh
typush list [namespace] [-a|--all] [-t|--tree] [--json]
```

扫描本地数据目录中的包，按命名空间与包名字典序升序、版本号语义化降序展示。

- `[namespace]`：仅查看指定命名空间（如 `local` 或 `preview`）。
- `-a, --all`：同时包含 Typst 下载缓存中的包（即编译时自动拉取的 `@preview` 缓存）。
- `-t, --tree`：以树状层级展示各命名空间下的包名与版本，并标注来源（已安装副本、软链接或缓存）。
- `--json`：以 JSON 格式输出已安装包的结构化数据。

```sh
# 列出本地安装的全部包
typush list

# 包含官方下载缓存并以树状展示
typush list -a -t

# 仅查看 @local 命名空间
typush list local
```

### 卸载已安装包 (uninstall)

```sh
typush uninstall <target> [-y|--force] [-n|--dry-run]
```

从本地 Typst 数据目录中删除指定的包或命名空间。若删除后父目录为空，会自动级联清理空目录。

目标语法支持：
- `@ns/pkg:ver`：删除指定命名空间下的特定版本。
- `@ns/pkg`：删除指定包的所有版本。
- `@ns`：删除整个命名空间。
- `pkg:ver`：省略命名空间时默认为 `@local` 下的指定版本。
- `pkg`：删除 `@local` 下该包的所有版本。

参数选项：
- `-y, --force`：跳过交互确认提示。
- `-n, --dry-run`：仅预览待删除的目标路径与说明，不删除任何文件。

```sh
# 卸载指定版本
typush uninstall @local/my-lib:0.1.0

# 卸载整个本地包（所有版本）
typush uninstall my-lib

# 清理整个自定义命名空间并跳过确认
typush uninstall @custom -y
```

---

## 16. Universe 官方包克隆 (clone)

```sh
typush clone <package> [destination] [-f|--force] [-n|--dry-run]
```

直接从 Typst Universe 官方 CDN（`packages.typst.org`）下载包的 `.tar.gz` 源码压缩包并解压到本地目录。解压时会对所有文件路径与软链接目标做安全检查，拒绝解压超出目标目录的文件。

- `<package>`：包规范。支持 `@preview/<name>:<version>`、`<name>:<version>` 或 `<name>`（省略版本时自动通过 CDN 索引解析最新版本）。
- `[destination]`：目标解压目录。省略时默认解压到当前目录下的 `<name>` 文件夹。
- `-f, --force`：当目标目录已存在且非空时，跳过覆盖确认提示。
- `-n, --dry-run`：预览将要克隆的版本与目标解压路径，不发起网络下载。

```sh
# 下载最新版本的 cetz 源码到当前目录下的 ./cetz
typush clone cetz

# 下载指定版本的 fletcher 到 ./my-fletcher
typush clone fletcher:0.5.0 ./my-fletcher
```

---

## 17. Shell 补全脚本 (completion)

```sh
typush completion <bash|zsh|fish|powershell>
```

生成对应 Shell 的自动补全脚本到标准输出。

### 各 Shell 加载方式

#### Bash

```sh
# 当前会话临时生效
source <(typush completion bash)

# 写入系统补全目录（持久生效）
typush completion bash > /etc/bash_completion.d/typush
```

#### Zsh

```sh
# 写入 fpath 所在目录
typush completion zsh > "${fpath[1]}/_typush"
# 重启终端或重新加载补全
autoload -U compinit && compinit
```

#### Fish

```sh
# 当前会话临时生效
typush completion fish | source

# 持久保存
typush completion fish > ~/.config/fish/completions/typush.fish
```

#### PowerShell

```powershell
typush completion powershell | Out-String | Invoke-Expression
```
