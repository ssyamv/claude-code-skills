---
name: setup-lark-cli-xfchat
description: 配置讯飞私有化飞书（xfchat.iflytek.com 及其子域）官方 CLI 工具 lark-cli。包含源码编译、config init 配置 brand、auth login OAuth 授权、常见坑点（brand 域名、代理、Redirect URL 白名单、权限 scope）排查。适用于需要通过 CLI 读写飞书文档、消息、日历、云盘、表格等的私有化部署场景。
---

# Setup lark-cli for 讯飞私有化飞书 (xfchat)

将讯飞私有化飞书 `xfchat.iflytek.com`（及子域）官方 CLI 工具 `lark-cli` 安装并完成登录授权。

## Phase 1: 收集信息

使用 `AskUserQuestion` 收集：

1. **开放平台域名** — 如 `xfchat.iflytek.com`（必须是应用所在的开放平台域，不是 `feishu`/`feishu.cn`）
2. **App ID** — 形如 `cli_xxxxxxxxxxxxx`，位于 `https://open.<brand>/app/<app-id>/baseinfo`
3. **App Secret** — 同页面获取
4. 是否已有仓库本地 clone，还是需要从 `https://code.iflytek.com/cbgme/devops/larksuite-cli.git` 克隆

## Phase 2: 安装（源码编译）

前置：Go 1.23+，Python 3，Git。

```bash
git clone https://code.iflytek.com/cbgme/devops/larksuite-cli.git
cd larksuite-cli
make build
```

若网络不稳定导致 metadata fetch 失败：

```bash
LARKSUITE_CLI_REMOTE_META=off go build -trimpath -o lark-cli .
```

安装到 PATH（二选一）：

```bash
# A. 安装到 ~/.local/bin
make install PREFIX=~/.local

# B. 符号链接跟随仓库最新构建
ln -sf "$(pwd)/lark-cli" ~/.local/bin/lark-cli
```

确认 `~/.local/bin` 在 `$PATH` 中，然后验证：

```bash
lark-cli --version
```

## Phase 3: 配置（关键：brand 必须正确）

⚠️ **最常见的坑**：交互式 `config init` 默认 `brand=feishu`，导致认证时报 `The specified app does not exist.`。**必须用非交互模式显式指定 `--brand`**。

若之前配置错误，先清除：

```bash
lark-cli config remove
```

然后：

```bash
export LARK_CLI_NO_PROXY=1

echo "<你的AppSecret>" | lark-cli config init \
  --brand xfchat.iflytek.com \
  --app-id <你的AppID> \
  --app-secret-stdin
```

参数说明：
- `--brand xfchat.iflytek.com`：必须和应用创建的开放平台域一致
- `--app-secret-stdin`：从 stdin 读取，避免 Secret 进入 shell history
- `LARK_CLI_NO_PROXY=1`：强制禁用 HTTP(S)_PROXY，防止凭据经由本地代理转发

验证：

```bash
lark-cli config show
```

关键字段：`"brand": "xfchat.iflytek.com"`（**不能是 `feishu`**）。配置文件位于 `~/.lark-cli/config.json`，Secret 存储在 OS 原生密钥链（macOS Keychain 等）。

## Phase 4: OAuth 登录

### 4.1 配置 Redirect URL 白名单

打开 `https://open.<brand>/app/<app-id>/safe`，在安全设置中添加：

```
http://localhost:8080/callback
```

未加白名单会报"请求非法"。

### 4.2 登录

```bash
export LARK_CLI_NO_PROXY=1
lark-cli auth login
```

CLI 会在 8080 端口启动本地 HTTP 服务，打开浏览器完成授权后自动捕获 callback。

验证：

```bash
lark-cli auth status
# 输出: Logged in as: <姓名> (ou_xxxxxxxx)
```

## Phase 5: 配置应用权限（如需要）

若调用 API 时报 `permission denied (230027 / 99991672)`：

1. 打开 `https://open.<brand>/app/<app-id>/auth`
2. 添加所需 scope（如 `docs:document:readonly`、`im:message:create_by_bot`、`calendar:event:create` 等）
3. **发布应用新版本**（权限需新版本才生效）
4. 重新运行 `lark-cli auth login`

## Phase 6: 验证联通

```bash
# 读取一篇文档（token 是 URL /docx/xxx 的 xxx 部分）
lark-cli docs +fetch --doc <doc_token> --format pretty

# 发一条群消息
lark-cli im +messages-send --chat-id <chat_id> --as bot --text "hi"
```

## 故障排查速查

| 错误 | 原因 | 解决 |
|------|------|------|
| `The specified app does not exist.` | brand 域名错误 | `config remove` 后重新用 `--brand xfchat.iflytek.com` 初始化 |
| `请求非法` | Redirect URL 不在白名单 | 在应用 safe 页面添加 `http://localhost:8080/callback` |
| `permission denied (230027 / 99991672)` | 权限不足 | 在 auth 页添加 scope，发布新版本，重新登录 |
| `proxy detected` 警告 | HTTPS_PROXY 未禁用 | `export LARK_CLI_NO_PROXY=1` |
| Port 8080 occupied | 端口被占用 | 关闭占用进程 |
| Token expired | 登录过期 | `lark-cli auth login` 重新登录 |

## 关键事实

- 默认 `brand=feishu`（公有云），讯飞私有化必须显式改为 `xfchat.iflytek.com`
- App Secret 通过 `--app-secret-stdin` 传入，不要放命令行参数
- 凡涉及凭据/token 的命令，前置 `export LARK_CLI_NO_PROXY=1`
- 文档操作默认 `--as user`，消息操作默认 `--as bot`
- 某些端点（mail、approval）在私有化部署可能未开放；search 需用 v1 原始 API (`/open-apis/suite/docs-api/search/object`)
- 快捷命令未覆盖的 API 用 `lark-cli api GET|POST <path> --as user|bot --params '{}' --data '{}'` 调用
