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
