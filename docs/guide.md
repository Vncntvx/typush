# typush 使用指南

typush 的命令参考与使用说明。

---

## 目录

1. [环境准备](#1-环境准备)
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

---

## 1. 环境准备

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
typush bump [patch|minor|major|<version>] [-n|--dry-run]
```

递增或指定 `typst.toml` 中的 `package.version`。

### 执行逻辑

1. 读取 `typst.toml` 中的当前版本。
2. 计算新版本：支持 `patch`、`minor`、`major` 关键字或直接指定目标版本号。未提供参数时通过终端交互选择，默认推荐下一个 patch 版本。
3. 防降级检查：目标版本必须高于当前版本。
4. 写回 `typst.toml` 并校验格式。加上 `-n`（或 `--dry-run`）时仅预览版本变动结果，不修改文件；此时不进入交互，直接按下一个 patch 版本计算。

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
typush download <repository> [-c ref] [-n namespace] [--dry-run]
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

从远程 Git 仓库下载包并安装到指定命名空间。可用 `-c` 指定分支、tag 或 commit。加上 `--dry-run` 可在临时拉取后仅做安装预览。

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

