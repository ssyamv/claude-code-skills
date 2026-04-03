# NanoClaw Feishu Skill

[NanoClaw](https://github.com/qwibitai/nanoclaw) 的飞书（Lark）通道集成 skill。

## 功能

- 飞书机器人消息收发（webhook 模式）
- 支持私有化飞书部署（如讯飞 xfchat.iflytek.com）
- 包含 feishu-docs 容器工具（读取飞书文档、Wiki、电子表格、多维表格）
- 群聊 @机器人 触发 + 私聊直接触发

## 安装

### 方式一：拷贝 skill 文件

将 `.claude/skills/add-feishu/SKILL.md` 复制到你的 NanoClaw 项目中：

```bash
mkdir -p .claude/skills/add-feishu
curl -o .claude/skills/add-feishu/SKILL.md \
  https://raw.githubusercontent.com/ssyamv/nanoclaw-skill-feishu/main/.claude/skills/add-feishu/SKILL.md
```

然后在 Claude Code 中运行 `/add-feishu`。

### 方式二：git merge

```bash
git remote add feishu-skill https://github.com/ssyamv/nanoclaw-skill-feishu.git
git fetch feishu-skill main
git merge feishu-skill/main --allow-unrelated-histories
```

然后运行 `/add-feishu`。

## 环境变量

| 变量 | 说明 |
|------|------|
| `FEISHU_APP_ID` | 飞书应用 App ID |
| `FEISHU_APP_SECRET` | 飞书应用 App Secret |
| `FEISHU_WEBHOOK_PORT` | Webhook 监听端口（默认 3000） |

## 工作原理

`/add-feishu` skill 会引导 Claude Code 自动：

1. 创建 `src/channels/feishu.ts`（通道实现）
2. 创建 `container/skills/feishu-docs/`（容器内文档读取工具）
3. 修改 `src/channels/index.ts`（注册通道）
4. 修改 `src/container-runner.ts`（传递凭证到容器）
5. 配置环境变量和 webhook

所有代码内嵌在 SKILL.md 中，无需额外的代码仓库。
