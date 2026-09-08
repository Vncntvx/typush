<div align="center">

# typush

[English](README.en.md) | 简体中文

</div>

typush 是一个用于 [Typst](https://typst.app/) 包开发与发布的命令行工具：在本地完成开发、校验与安装，然后发布到 [Typst Universe](https://github.com/typst/packages)。

名字来自 **Typst** + **push**：把包推进 [universe](https://typst.app/universe/)。

## 安装

```sh
go install github.com/Vncntvx/typush@latest
```

也可以从 [Releases](https://github.com/Vncntvx/typush/releases) 下载对应平台的二进制文件。

## 前置要求

`publish universe`（以及 `dev --check`）通过 [GitHub CLI](https://cli.github.com) 访问 GitHub。安装 `gh` 并运行一次 `gh auth login` 即可。`typush login universe` 会检查这套配置是否就绪。

## 快速上手

初始化一个新包（交互式）：

```sh
typush init
```

开发模板时，把包目录符号链接进 `@preview` 命名空间：

```sh
typush dev
```

发布到 Universe（先执行本地校验）：

```sh
typush publish universe
```

## 命令

```sh
typush --help
typush init [name]        # 交互式：创建 typst.toml + 入口文件
typush check [--local] [--no-compile]  # 校验包，见下文
typush install <ns>       # 安装到 @<ns>（如 local）
typush download <repo> [-c ref] [-n ns]
typush dev [--check]      # 符号链接进 @preview（可选 Universe 冲突检查）
typush clean [package]
typush exclude <globs...>
typush login universe
typush publish universe [--dry-run]   # 本地 Universe 校验 + sparse-checkout 上传 + 草稿 PR
typush ci plan [--packages "a b"]     # 扫描工作区 -> CI matrix JSON
typush ci generate [--source ...] [--push-to-fork ...] [--destination ...]
```

`publish` 会先运行本地 Universe 校验。官方 `bundler` CI 常见的拒绝原因（未知字段、authors 格式、categories、SPDX 许可证、缺少 README/LICENSE、模板缩略图、禁止 exclude 的文件）在联网之前就能查出来。

## 本地校验

如果 `typst` 在 `PATH` 中（可用 `TYPST_BIN` 指定路径），`typush check` 还会在本地离线运行官方 CI 会执行的检查：

- 编译：对库做一次导入检查；对模板执行官方流程，先 `typst init @preview/<name>:<version>` 到临时项目，再编译模板入口。编译错误判定为失败，警告会报告。检查在临时 `HOME`/XDG 目录中运行，不影响本机的 Typst 数据。`--no-compile` 跳过编译，`--local` 只检查 manifest 最小规则。
- README：缺失图片 alt 文本（错误）、失效的本地链接（错误）、GFM alerts/任务列表、指向默认分支的仓库 URL（警告）。
- 文件：字体文件（错误）、未排除的 `example`/`test` 文件与超大文件、被忽略但仍存在的文件、未被链接的 manual。
- 导入：对入口文件的相对导入、过时的自身版本导入（含 README）、不符合规范的模板导入。

## 设计取舍

- 用 Go 编写，编译为单一静态二进制，通过 GoReleaser 发布。
- GitHub 操作（认证检查、API 查询、创建 PR）统一通过 `gh` CLI 完成，不直接调用 GitHub API；`publish` 只支持 sparse-checkout 上传，需要 git >= 2.25。
- `check` 覆盖官方 `bundler` 的硬性错误规则。
- `dev` 默认跳过 Universe 网络检查，需要时加 `--check`。
- `host` / `generate` 合并到 `ci` 子命令，旧名称作为隐藏别名保留。
- 交互式提示也读取管道 stdin，`printf ... | typush init` 可以在脚本中使用。
- GitHub 访问统一通过 `gh` CLI，不读写本地存储的 token。

## 开发

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## 许可证

[MIT](LICENSE)
