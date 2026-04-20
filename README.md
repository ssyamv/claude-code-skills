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

These one-line commands fetch installer scripts from `raw.githubusercontent.com`, and those scripts then resolve and download bootstrapper binaries from GitHub Releases.

macOS latest:

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash
```

macOS pinned:

```bash
curl -fsSL https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.sh | bash -s -- --version v0.1.0
```

Windows latest:

```powershell
irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1 | iex
```

Windows pinned:

```powershell
&([scriptblock]::Create((irm https://raw.githubusercontent.com/ssyamv/claude-code-skills/main/install/install.ps1))) -Version v0.1.0
```

GitHub Releases must contain the matching platform assets for each published tag, or the installer will fail when resolving the download URL.

## Smoke Test

发布前按照 [`docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md`](docs/superpowers/specs/xfchat-bootstrapper-smoke-test.md) 执行 smoke test。最低限度先确认 `make test` 通过，然后检查打包产物是否生成在 `dist/`。`make build` 和发布脚本都会生成 `dist/` 里的二进制文件；如果想清理这些产物，运行 `make clean`。
