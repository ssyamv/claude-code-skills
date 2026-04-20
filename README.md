# Claude Code Skills

Claude Code 的 skill 合集。每个 skill 是一个自包含的 SKILL.md，Claude Code 执行时会自动生成代码和配置。

## 可用 Skills

| Skill | 说明 | 适用平台 |
|-------|------|----------|
| `/add-feishu` | 添加飞书（Lark）通道，支持私有化部署 | NanoClaw |
| `/add-openclaw-feishu` | 接入讯飞私有化飞书，含代理层和文档插件 | OpenClaw |
| `/setup-lark-cli-xfchat` | 配置讯飞私有化飞书官方 CLI（lark-cli）| 通用 |

## 安装

### 安装单个 skill

```bash
# 例：安装 add-feishu
mkdir -p .claude/skills/add-feishu
curl -o .claude/skills/add-feishu/SKILL.md \
  https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/.claude/skills/add-feishu/SKILL.md
```

### 安装全部 skills

```bash
git remote add skills https://github.com/ssyamv/claude-code-skills.git
git fetch skills main
git merge skills/main --allow-unrelated-histories
```

## 使用

安装后在 Claude Code 中直接输入对应的 slash 命令（如 `/add-feishu`），Claude 会自动按照 skill 引导完成安装和配置。

## 构建与发布

```bash
make build
make test
make release
```

Windows 环境下可使用：

```powershell
./scripts/build-release.ps1
```

Windows installer 和 release 产物当前仅支持 x64（AMD64）；对应的发布资产是 `xfchat-bootstrapper-windows-amd64.exe`。

## Xfchat Bootstrapper Install

如果你是在另一台电脑上直接安装 `xfchat-bootstrapper`，不需要先 clone 仓库，也不需要先安装 Go。

安装流程是：

1. 一键命令先从 `raw.githubusercontent.com` 拉取安装脚本
2. 安装脚本再去 GitHub Releases 解析目标版本并下载对应平台的二进制
3. 二进制安装到当前用户目录
4. 安装脚本自动补 PATH
5. 安装完成后立即运行一次 `xfchat-bootstrapper`

### Prerequisites

- macOS 或 Windows
- 能访问 GitHub `raw.githubusercontent.com` 和 GitHub Releases
- GitHub Releases 上已经存在对应版本的发布资产
- Windows 当前只支持 `x64 (AMD64)`，对应资产是 `xfchat-bootstrapper-windows-amd64.exe`

### Install On Another Machine

macOS 最新版：

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash
```

macOS 指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash -s -- --version v0.1.0
```

Windows 最新版：

```powershell
irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1 | iex
```

Windows 指定版本：

```powershell
&([scriptblock]::Create((irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1))) -Version v0.1.0
```

### Install Result

- macOS 会安装到 `~/.local/bin/xfchat-bootstrapper`
- Windows 会安装到 `%LocalAppData%\Programs\XfchatBootstrapper\xfchat-bootstrapper.exe`
- 安装脚本会尝试自动更新 PATH
- 即使当前 shell / PowerShell 会话还没重新加载 PATH，脚本也会用绝对路径直接启动程序

### Version Notes

- 不带版本参数时，默认安装 GitHub Releases 上的 `latest`
- 带 `--version v0.1.0` 或 `-Version v0.1.0` 时，安装对应 tag 的发布资产
- 如果 GitHub Releases 上没有对应平台的资产，安装会失败

## Smoke Test

发布前按照 [`docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md`](docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md) 执行 smoke test。最低限度先确认 `make test` 通过，然后检查打包产物是否生成在 `dist/`。`make build` 和发布脚本都会生成 `dist/` 里的二进制文件；如果想清理这些产物，运行 `make clean`。

## Standalone Runtime Note

`xfchat-bootstrapper` no longer requires an external `lark-cli` binary in the normal startup path (`lark-cli.exe` on Windows, `lark-cli` on macOS and Linux). For the release regression checklist, see [`docs/superpowers/specs/standalone-runtime-regression-checklist.md`](docs/superpowers/specs/standalone-runtime-regression-checklist.md).
