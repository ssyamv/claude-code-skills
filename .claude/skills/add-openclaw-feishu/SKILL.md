---
name: add-openclaw-feishu
description: 在 OpenClaw 上接入讯飞私有化飞书（xfchat.iflytek.com）。包含 webhook 代理、飞书文档插件、ngrok 隧道配置。适用于腾讯云等服务器部署场景。
---

# OpenClaw 接入讯飞私有化飞书

在 OpenClaw 上部署讯飞私有化飞书机器人。由于私有化飞书与公网飞书存在兼容性差异（签名机制、API 地址），需要额外的代理层和自定义插件。

## 架构

```
讯飞飞书 → ngrok (HTTPS) → feishu-proxy (:3000) → OpenClaw webhook (:3001) → Agent
```

- **ngrok**：提供 HTTPS 隧道，解决讯飞内网无法直连境外服务器的问题
- **feishu-proxy**：Node.js 代理，处理 challenge 验证和签名转发
- **OpenClaw webhook**：接收消息并分发给 agent
- **iflytek_feishu_doc 插件**：替代内置 feishu_doc，通过私有化 API 读取文档

## Phase 1: Pre-flight

### 前置条件确认

Use `AskUserQuestion` to confirm:

AskUserQuestion: 请确认以下前置条件：
1. 你有一台有公网 IP 的服务器（如腾讯云），已安装 OpenClaw
2. 你已在讯飞飞书开放平台（open.xfchat.iflytek.com）创建了应用
3. 你有应用的 App ID、App Secret、Verification Token、Encrypt Key

如果还没有飞书应用，需要先去创建。你准备好了吗？

### 收集凭证

收集以下信息：
- **App ID**
- **App Secret**
- **Verification Token**
- **Encrypt Key**
- **Agent 名称**（如"小讯"）
- **Agent ID**（如 "xiaoxun"）

## Phase 2: 创建 Agent

### 2.1 创建 agent 目录

```bash
sudo mkdir -p /root/.openclaw/workspace-<agent-id>
sudo mkdir -p /root/.openclaw/agents/<agent-id>/agent
```

### 2.2 修改 openclaw.json

在 `openclaw.json` 的 `agents.list` 中添加新 agent：

```json
{
  "id": "<agent-id>",
  "name": "<agent-name>",
  "workspace": "/root/.openclaw/workspace-<agent-id>",
  "agentDir": "/root/.openclaw/agents/<agent-id>/agent"
}
```

## Phase 3: 配置飞书 Channel

### 3.1 添加飞书账号

在 `openclaw.json` 的 `channels.feishu.accounts` 中添加：

```json
{
  "iflytek": {
    "appId": "<App ID>",
    "appSecret": "<App Secret>",
    "domain": "https://open.xfchat.iflytek.com",
    "botName": "<agent-name>",
    "connectionMode": "webhook",
    "webhookPort": 3001,
    "webhookHost": "127.0.0.1",
    "groupPolicy": "open",
    "markdown": { "mode": "strip" },
    "verificationToken": "<Verification Token>",
    "encryptKey": "<Encrypt Key>",
    "tools": {
      "doc": false,
      "wiki": false,
      "drive": false,
      "chat": false,
      "perm": false,
      "scopes": false
    }
  }
}
```

**重要：** 必须将所有内置飞书工具设为 `false`，因为它们走公网飞书 API，会返回 404/502 错误。

### 3.2 添加 binding

在 `openclaw.json` 的 `bindings` 中添加：

```json
{
  "agentId": "<agent-id>",
  "match": { "channel": "feishu", "accountId": "iflytek" }
}
```

## Phase 4: 部署 Webhook 代理

讯飞私有化飞书的签名机制和公网飞书不同，OpenClaw 的 Lark SDK 会验证签名失败。需要一个代理来处理。

### 4.1 创建代理脚本

创建 `/opt/feishu-proxy.js`：

```javascript
const http = require('http');
const crypto = require('crypto');

const ENCRYPT_KEY = '<Encrypt Key>';
const VERIFY_TOKEN = '<Verification Token>';
const UPSTREAM = 'http://127.0.0.1:3001';

function decrypt(encrypted) {
  const buf = Buffer.from(encrypted, 'base64');
  const key = crypto.createHash('sha256').update(ENCRYPT_KEY).digest();
  const iv = buf.slice(0, 16);
  const dec = crypto.createDecipheriv('aes-256-cbc', key, iv);
  let d = dec.update(buf.slice(16));
  d = Buffer.concat([d, dec.final()]);
  return d.toString();
}

function sign(ts, nonce, body) {
  const s = ts + nonce + ENCRYPT_KEY + body;
  return crypto.createHash('sha256').update(s).digest('hex');
}

http.createServer((req, res) => {
  let body = '';
  req.on('data', c => body += c);
  req.on('end', () => {
    try {
      let data = JSON.parse(body);
      if (data.encrypt) {
        try { data = JSON.parse(decrypt(data.encrypt)); } catch(e) {}
      }
      // Challenge 验证 - 直接响应
      if (data.type === 'url_verification' || data.challenge) {
        res.writeHead(200, {'Content-Type':'application/json'});
        res.end(JSON.stringify({challenge: data.challenge}));
        return;
      }
      // 转发到 OpenClaw，带正确签名
      const ts = String(Math.floor(Date.now()/1000));
      const nonce = crypto.randomBytes(8).toString('hex');
      const sig = sign(ts, nonce, body);
      const opts = {
        method: 'POST',
        hostname: '127.0.0.1', port: 3001, path: '/feishu/events',
        headers: {
          'Content-Type': 'application/json',
          'X-Lark-Request-Timestamp': ts,
          'X-Lark-Request-Nonce': nonce,
          'X-Lark-Signature': sig
        }
      };
      const p = http.request(opts, r => {
        let d = ''; r.on('data', c => d += c);
        r.on('end', () => { res.writeHead(r.statusCode, r.headers); res.end(d); });
      });
      p.on('error', e => {
        res.writeHead(200, {'Content-Type':'application/json'});
        res.end(JSON.stringify({code:0}));
      });
      p.write(body); p.end();
    } catch(e) {
      res.writeHead(200, {'Content-Type':'application/json'});
      res.end(JSON.stringify({code:0}));
    }
  });
}).listen(3000, '0.0.0.0', () => console.log('Feishu proxy on :3000'));
```

将用户提供的 `<Encrypt Key>` 和 `<Verification Token>` 替换进去。

### 4.2 启动代理

```bash
sudo systemd-run --unit=feishu-proxy /path/to/node /opt/feishu-proxy.js
```

## Phase 5: 部署飞书文档读取插件

### 5.1 创建 feishu-docs 脚本

创建 `/usr/local/bin/feishu-docs`，写入以下内容并 `chmod +x`：

```bash
#!/bin/bash
# Feishu Document Reader
# Supports: docx, wiki, sheets (spreadsheet), bitable (多维表格), drive (file list)

set -euo pipefail

FEISHU_API_BASE="https://open.xfchat.iflytek.com"

check_credentials() {
    if [[ -z "${FEISHU_APP_ID:-}" ]] || [[ -z "${FEISHU_APP_SECRET:-}" ]]; then
        echo "Error: FEISHU_APP_ID or FEISHU_APP_SECRET not set" >&2
        exit 1
    fi
}

get_access_token() {
    local response
    response=$(curl -s -X POST "${FEISHU_API_BASE}/open-apis/auth/v3/tenant_access_token/internal" \
        -H "Content-Type: application/json" \
        -H "User-Agent: Mozilla/5.0" \
        -d "{\"app_id\":\"${FEISHU_APP_ID}\",\"app_secret\":\"${FEISHU_APP_SECRET}\"}")
    local code
    code=$(echo "$response" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error: Failed to get access token: $response" >&2
        exit 1
    fi
    echo "$response" | jq -r '.tenant_access_token'
}

extract_id() {
    local input="$1"
    if [[ "$input" =~ https?:// ]]; then
        if [[ "$input" =~ /sheets/([a-zA-Z0-9_-]+) ]]; then
            echo "${BASH_REMATCH[1]} sheets"
        elif [[ "$input" =~ /base/([a-zA-Z0-9_-]+) ]] || [[ "$input" =~ /bitable/([a-zA-Z0-9_-]+) ]]; then
            echo "${BASH_REMATCH[1]} bitable"
        elif [[ "$input" =~ /wiki/([a-zA-Z0-9_-]+) ]]; then
            echo "${BASH_REMATCH[1]} wiki"
        elif [[ "$input" =~ /docx?/([a-zA-Z0-9_-]+) ]]; then
            echo "${BASH_REMATCH[1]} docx"
        elif [[ "$input" =~ /drive/folder/([a-zA-Z0-9_-]+) ]]; then
            echo "${BASH_REMATCH[1]} folder"
        else
            echo "Error: Cannot determine document type from URL: $input" >&2
            exit 1
        fi
    else
        if [[ "$input" =~ ^sht ]]; then echo "$input sheets"
        elif [[ "$input" =~ ^bascn ]] || [[ "$input" =~ ^basc ]]; then echo "$input bitable"
        else echo "$input docx"; fi
    fi
}

read_docx() {
    local doc_id="$1" token="$2"
    local response
    response=$(curl -s "${FEISHU_API_BASE}/open-apis/docx/v1/documents/${doc_id}/raw_content" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code=$(echo "$response" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading docx: $(echo "$response" | jq -r '.msg')" >&2; exit 1
    fi
    echo "$response" | jq -r '.data.content // ""'
}

read_wiki() {
    local node_token="$1" token="$2"
    local node_resp
    node_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/get_node?token=${node_token}" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code=$(echo "$node_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading wiki node: $(echo "$node_resp" | jq -r '.msg')" >&2; exit 1
    fi
    local obj_token=$(echo "$node_resp" | jq -r '.data.node.obj_token')
    local obj_type=$(echo "$node_resp" | jq -r '.data.node.obj_type')
    local title=$(echo "$node_resp" | jq -r '.data.node.title')
    echo "# $title"; echo ""
    if [[ "$obj_type" == "docx" ]]; then read_docx "$obj_token" "$token"
    else echo "(obj_type: $obj_type, obj_token: $obj_token)"; fi
}

read_sheets() {
    local spreadsheet_token="$1" token="$2"
    local sheets_resp
    sheets_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/sheets/v3/spreadsheets/${spreadsheet_token}/sheets/query" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code=$(echo "$sheets_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading spreadsheet: $(echo "$sheets_resp" | jq -r '.msg')" >&2; exit 1
    fi
    echo "$sheets_resp" | jq -c '.data.sheets[]?' | while IFS= read -r sheet; do
        [[ -z "$sheet" ]] && continue
        local sid=$(echo "$sheet" | jq -r '.sheet_id')
        local stitle=$(echo "$sheet" | jq -r '.title')
        echo "## Sheet: $stitle (id: $sid)"; echo ""
        local val_resp
        val_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/sheets/v2/spreadsheets/${spreadsheet_token}/values/${sid}!A1:ZZ1000" \
            -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        if [[ "$(echo "$val_resp" | jq -r '.code // 1')" == "0" ]]; then
            echo "$val_resp" | jq -r '.data.valueRange.values[]? | map(if . == null then "" else (. | tostring) end) | join("\t")'
        fi; echo ""
    done
}

read_bitable() {
    local app_token="$1" token="$2"
    local tables_resp
    tables_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables?page_size=100" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code=$(echo "$tables_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading bitable: $(echo "$tables_resp" | jq -r '.msg')" >&2; exit 1
    fi
    echo "$tables_resp" | jq -c '.data.items[]?' | while IFS= read -r table; do
        [[ -z "$table" ]] && continue
        local tid=$(echo "$table" | jq -r '.table_id')
        local tname=$(echo "$table" | jq -r '.name')
        echo "## Table: $tname (id: $tid)"; echo ""
        local fields_resp
        fields_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables/${tid}/fields?page_size=100" \
            -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        echo "$fields_resp" | jq -r '[.data.items[]?.field_name] | join("\t")' 2>/dev/null || true
        local page_token=""
        while true; do
            local url="${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables/${tid}/records?page_size=100"
            [[ -n "$page_token" ]] && url="${url}&page_token=${page_token}"
            local rec_resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
            echo "$rec_resp" | jq -r '.data.items[]?.fields | to_entries | map(.value | if type == "array" then map(if type == "object" then (.text // .name // tostring) else tostring end) | join(", ") elif type == "object" then (.text // .value // tostring) else tostring end) | join("\t")' 2>/dev/null || true
            [[ "$(echo "$rec_resp" | jq -r '.data.has_more // false')" != "true" ]] && break
            page_token=$(echo "$rec_resp" | jq -r '.data.page_token // ""')
            [[ -z "$page_token" ]] && break
        done; echo ""
    done
}

list_drive() {
    local folder_token="${1:-}" token="$2"
    local url="${FEISHU_API_BASE}/open-apis/drive/v1/files?page_size=50"
    [[ -n "$folder_token" ]] && url="${url}&folder_token=${folder_token}"
    local resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code=$(echo "$resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error listing drive: $(echo "$resp" | jq -r '.msg')" >&2; exit 1
    fi
    echo "$resp" | jq -r '.data.files[]? | "\(.type)\t\(.name)\t\(.token)"'
}

main() {
    [[ $# -lt 1 ]] && { echo "Usage: feishu-docs read|list|list-drive <url_or_id>" >&2; exit 1; }
    check_credentials
    local token=$(get_access_token)
    local command="$1" input="${2:-}"
    case "$command" in
        read)
            [[ -z "$input" ]] && { echo "Usage: feishu-docs read <url_or_id>" >&2; exit 1; }
            local id_info=$(extract_id "$input")
            local doc_id=$(echo "$id_info" | cut -d' ' -f1)
            local doc_type=$(echo "$id_info" | cut -d' ' -f2)
            case "$doc_type" in
                docx) read_docx "$doc_id" "$token" ;; wiki) read_wiki "$doc_id" "$token" ;;
                sheets) read_sheets "$doc_id" "$token" ;; bitable) read_bitable "$doc_id" "$token" ;;
                folder) list_drive "$doc_id" "$token" ;; *) echo "Error: Unknown type $doc_type" >&2; exit 1 ;;
            esac ;;
        list-drive) list_drive "${input:-}" "$token" ;;
        *) echo "Usage: feishu-docs read|list-drive <url_or_id>" >&2; exit 1 ;;
    esac
}
main "$@"
```

### 5.2 创建 OpenClaw 插件

创建目录 `/root/.openclaw/extensions/iflytek-feishu-doc/`，包含三个文件。

**package.json：**

```json
{
  "name": "iflytek-feishu-doc",
  "version": "1.0.0",
  "type": "module",
  "openclaw": {
    "extensions": ["./index.ts"]
  }
}
```

**openclaw.plugin.json：**

```json
{
  "id": "iflytek-feishu-doc",
  "configSchema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {}
  }
}
```

**index.ts：**

```typescript
import type { OpenClawPluginApi } from "openclaw/plugin-sdk";
import { execSync } from "child_process";

const TOOL_NAME = "iflytek_feishu_doc";

const TOOL_PARAMETERS = {
  title: "讯飞飞书文档读取",
  type: "object",
  description: "读取讯飞私有化飞书文档内容。",
  additionalProperties: false,
  required: ["url"],
  properties: {
    url: {
      type: "string",
      minLength: 1,
      description: "飞书文档的 URL 或文档 token",
    },
  },
};

export default function register(api: OpenClawPluginApi) {
  api.registerTool((ctx) => {
    return {
      name: TOOL_NAME,
      description:
        "读取讯飞私有化飞书文档。当用户提到 xfchat.iflytek.com 的文档链接时使用此工具。",
      parameters: TOOL_PARAMETERS,
      async execute(toolCallId: string, params: unknown) {
        const p = params as Record<string, unknown>;
        const url = String(p?.url || "");
        if (!url) {
          return {
            output: { ok: false, error: "请提供文档 URL 或 token" },
            content: [{ type: "text", text: "请提供文档 URL 或 token" }],
            isError: true,
          };
        }
        try {
          const safeUrl = url.replace(/'/g, "'\\''");
          const appId = process.env.FEISHU_APP_ID || "<App ID>";
          const appSecret = process.env.FEISHU_APP_SECRET || "<App Secret>";
          const cmd = `FEISHU_APP_ID=${appId} FEISHU_APP_SECRET=${appSecret} /usr/local/bin/feishu-docs read '${safeUrl}'`;
          const result = execSync(cmd, {
            timeout: 30000,
            maxBuffer: 1024 * 1024,
            encoding: "utf-8",
            stdio: ["pipe", "pipe", "pipe"],
          });
          const text = result || "文档内容为空";
          return {
            output: { ok: true, content: text },
            content: [{ type: "text", text }],
          };
        } catch (err: any) {
          const stderr = err.stderr ? String(err.stderr) : "";
          const errorText = `读取失败: ${stderr || err.message}`;
          return {
            output: { ok: false, error: errorText },
            content: [{ type: "text", text: errorText }],
            isError: true,
          };
        }
      },
    };
  });
  console.log(`[iflytek-feishu-doc] Tool ${TOOL_NAME} registered`);
}
```

将插件中的 `<App ID>` 和 `<App Secret>` 替换为用户的凭证，或者引导用户通过环境变量传入。

### 5.3 在 openclaw.json 中启用插件

```json
{
  "plugins": {
    "entries": {
      "iflytek-feishu-doc": { "enabled": true }
    },
    "installs": {
      "iflytek-feishu-doc": {
        "source": "path",
        "installPath": "/root/.openclaw/extensions/iflytek-feishu-doc"
      }
    }
  }
}
```

## Phase 6: 配置 Agent CLAUDE.md

创建 `/root/.openclaw/agents/<agent-id>/agent/CLAUDE.md`：

```markdown
# <agent-name> Agent

你是<agent-name>，讯飞私有化飞书的AI助手。请用中文回复。

## 重要：飞书文档读取规则

你运行在讯飞私有化飞书环境（xfchat.iflytek.com）。
内置的 feishu_doc 工具不兼容此环境，会返回 404 或 502 错误。

**当你需要读取任何飞书文档时，必须使用 iflytek_feishu_doc 工具。**
**当你收到飞书文档链接时，必须使用 iflytek_feishu_doc 工具读取，不要使用 feishu_doc。**
**当 feishu_doc 返回 404/502 错误时，必须立即用 iflytek_feishu_doc 重试。**
```

## Phase 7: 配置 ngrok

### 7.1 安装 ngrok

```bash
wget -q https://bin.equinox.io/c/bNyj1mQVY4c/ngrok-v3-stable-linux-amd64.tgz -O /tmp/ngrok.tgz
tar xzf /tmp/ngrok.tgz -C /usr/local/bin
```

### 7.2 配置并启动

```bash
# 配置 authtoken（需要已验证邮箱的 ngrok 账号）
sudo ngrok config add-authtoken <ngrok-token>

# 启动隧道
sudo systemd-run --unit=ngrok-tunnel -E HOME=/root \
  ngrok http 3000 --config /root/.config/ngrok/ngrok.yml

# 获取公网 URL
curl -s http://127.0.0.1:4040/api/tunnels | jq -r '.tunnels[].public_url'
```

## Phase 8: 配置飞书开放平台

告诉用户：

> 1. 登录 `open.xfchat.iflytek.com`
> 2. 进入你的应用 → **事件与回调**
> 3. **订阅方式**选"将事件发送至开发者服务器"
> 4. **请求地址**填：`<ngrok URL>/feishu/events`（注意是 proxy 的端口 3000 对应的 ngrok URL）
> 5. **加密策略**：设置 Encrypt Key 和 Verification Token（必须与 feishu-proxy.js 中配置的一致）
> 6. **事件配置**：添加 `im.message.receive_v1`
> 7. **权限管理**：添加 `im:message:send_v2`、`im:message:receive`、`docx:document:readonly`、`wiki:wiki:readonly`、`sheets:spreadsheet`、`bitable:app:readonly`、`drive:drive:readonly`

## Phase 9: 重启 OpenClaw 并验证

```bash
# 重启 OpenClaw
sudo systemctl restart openclaw
```

### 批准用户访问

用户首次给机器人发消息时会收到 pairing code：

```bash
sudo env PATH=/root/.nvm/versions/node/v22.22.1/bin:$PATH \
  openclaw pairing approve feishu <PAIRING_CODE>
```

### 测试

告诉用户：

> 1. 在飞书中给机器人发一条消息
> 2. 如果是群聊，需要 @机器人
> 3. 等待几秒，机器人应该会回复
> 4. 发送一个飞书文档链接，测试 iflytek_feishu_doc 工具是否正常

## 常见问题

### webhook 配置时"请求3秒超时"

讯飞内网无法访问境外 IP 的非标准端口。必须通过 ngrok 隧道。

### "返回数据不是合法的JSON格式"

OpenClaw 的 Lark SDK 签名验证失败，返回了纯文本 `Invalid signature`。这正是 feishu-proxy 要解决的问题——确保代理正在运行。

### 文档读取返回 404

内置 feishu_doc 工具走的是公网飞书 API，不兼容私有化飞书。必须使用 iflytek_feishu_doc 插件。检查 Agent 的 CLAUDE.md 是否正确配置了文档读取规则。

### iflytek_feishu_doc 工具参数为空

OpenClaw 插件的 `registerTool` 要求用 `parameters` 字段（不是 `schema`），`execute` 签名为 `execute(toolCallId: string, params: unknown)`。

### ngrok 报 ERR_NGROK_4018

ngrok 账号邮箱未验证。去 dashboard.ngrok.com 验证邮箱。

### ngrok 重启后域名变化

免费版 ngrok 每次重启分配新域名，需要去飞书开放平台更新 webhook 地址。建议购买 ngrok 固定域名。

## 文件清单

| 文件 | 用途 |
|------|------|
| `/root/.openclaw/openclaw.json` | OpenClaw 主配置 |
| `/root/.openclaw/agents/<id>/agent/CLAUDE.md` | Agent 指令 |
| `/root/.openclaw/extensions/iflytek-feishu-doc/index.ts` | 文档读取插件 |
| `/root/.openclaw/extensions/iflytek-feishu-doc/package.json` | 插件包配置 |
| `/root/.openclaw/extensions/iflytek-feishu-doc/openclaw.plugin.json` | 插件注册配置 |
| `/opt/feishu-proxy.js` | Webhook 签名代理 |
| `/usr/local/bin/feishu-docs` | 飞书文档读取脚本 |
| `/root/.config/ngrok/ngrok.yml` | ngrok 配置 |
