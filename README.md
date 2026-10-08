<div align="center">

# typush

[English](README.en.md) | 简体中文

</div>

typush 是 [Typst](https://typst.app/) 包开发与发布工具，支持本地开发、校验与安装，并发布到 [Typst Universe](https://github.com/typst/packages)。

项目名称由 Typst 与 push 组合而成，意为将包推送到 [universe](https://typst.app/universe/)。

## 安装

```sh
go install github.com/Vncntvx/typush@latest
```

也可以从 [Releases](https://github.com/Vncntvx/typush/releases) 下载对应平台的二进制文件。

## 前置要求

`publish universe` 与 `dev --check` 命令依赖 [GitHub CLI](https://cli.github.com)。安装 `gh` 并执行 `gh auth login` 完成登录即可；使用 `typush login universe` 可以检查当前的认证状态。

## 快速上手

创建新包：

```sh
typush init
```

开发模板或本地调试时，将当前目录链接到 `@preview` 命名空间：

```sh
typush dev
```

发布到 Universe：

```sh
typush publish universe
```

## 命令

```sh
typush --help
typush init [name]        # 交互式初始化 typst.toml、README.md、LICENSE 与入口文件
typush bump [patch|minor|major|<ver>] # 更新版本号并同步 README 中的包版本引用
typush check [--local] [--no-compile]  # 校验包规范与编译状态
typush install <ns>       # 安装到指定命名空间（例如 @local）
typush download <repo> [-c ref] [-n ns] # 从 Git 仓库下载并安装包
typush dev [--check]      # 链接到 @preview（可选检查线上命名冲突）
typush dev list           # 列出 @preview 中已有的开发软链接及路径
typush clean [package]    # 清理 @preview 中的开发软链接
typush exclude <globs...> # 将指定文件添加到发布排除列表
typush login universe     # 验证 GitHub CLI 认证状态
typush publish universe [--dry-run]   # 校验并提交 PR 到 Universe（自动处理 fork 与分支）
typush pr status [number|url]         # 查看 PR 状态与审查意见（默认自动匹配当前包）
typush pr checks [number|url] [-w]    # 查看或等待 Universe 官方 CI 结果
typush ci plan [--packages "a b"]     # 扫描工作区并输出 CI matrix JSON
typush ci generate [...]              # 生成自动化发布工作流
```

`publish` 在提交前会完整运行本地 Universe 校验，提前检查未知字段、作者格式、分类、SPDX 许可证、必要文件完整性、模板缩略图以及排除规则等常见 CI 问题。

## 本地校验

当系统路径中存在 `typst`（或通过 `TYPST_BIN` 环境变量指定）时，`typush check` 会在本地运行官方 CI 的检查项：

- 编译：普通包检查导入是否成功；模板包在隔离临时目录中先执行 `typst init @preview/<name>:<version>` 再编译入口文件。检查在独立的临时目录中运行，不修改本机的 Typst 数据。可用 `--no-compile` 跳过编译，或用 `--local` 仅检查元数据格式。
- README：检查图片缺失的 alt 文本、失效的相对链接、GFM 语法支持，以及是否使用了指向默认分支的可变链接。
- 文件：禁止携带字体文件，检查是否遗漏排除了样例/测试文件或大文件，并提示未引用的手册文件。
- 导入：检查入口文件是否存在相对导入、包自引用版本是否与当前版本一致，以及模板中的导入路径是否合规。

## 开发

```sh
go build ./...
go test ./...
go vet ./...
gofmt -l .
```

## 许可证

[MIT](LICENSE)
