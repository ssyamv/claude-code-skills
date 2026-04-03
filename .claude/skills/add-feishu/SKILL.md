---
name: add-feishu
description: Add Feishu (飞书/Lark) as a channel. Supports private Feishu deployments (e.g., Iflytek xfchat). Uses webhook mode for message receiving. Includes feishu-docs tool for reading Feishu documents, wikis, spreadsheets, and bitables inside containers.
---

# Add Feishu Channel

This skill adds Feishu (飞书) support to NanoClaw, including a container tool for reading Feishu documents. Supports both public Feishu and private deployments (e.g., Iflytek xfchat.iflytek.com).

## Phase 1: Pre-flight

### Check if already applied

Check if `src/channels/feishu.ts` exists. If it does, skip to Phase 3 (Setup). The code changes are already in place.

### Ask the user

Use `AskUserQuestion` to collect configuration:

AskUserQuestion: Do you have a Feishu app with App ID and App Secret, or do you need to create one? Also, are you using public Feishu (feishu.cn) or a private deployment (e.g., xfchat.iflytek.com)?

If they have credentials, collect them now. If not, we'll create them in Phase 3.

## Phase 2: Apply Code Changes

### 2.1 Create `src/channels/feishu.ts`

Write this file:

```typescript
import express, { Request, Response } from 'express';
import axios from 'axios';

import { readEnvFile } from '../env.js';
import { logger } from '../logger.js';
import { ASSISTANT_NAME } from '../config.js';
import { registerChannel, ChannelOpts } from './registry.js';
import {
  Channel,
  OnChatMetadata,
  OnInboundMessage,
  RegisteredGroup,
} from '../types.js';

const FEISHU_API_BASE = 'https://open.xfchat.iflytek.com';

export class FeishuChannel implements Channel {
  name = 'feishu';

  private appId: string;
  private appSecret: string;
  private opts: ChannelOpts;
  private accessToken: string | null = null;
  private tokenExpireTime = 0;
  private server: ReturnType<typeof express> | null = null;
  private httpServer: any = null;
  private port: number;
  private connected = false;
  private lastMessageId: Map<string, string> = new Map();
  private botOpenId: string | null = null;

  constructor(appId: string, appSecret: string, port: number, opts: ChannelOpts) {
    this.appId = appId;
    this.appSecret = appSecret;
    this.port = port;
    this.opts = opts;
  }

  // 获取机器人自身的 open_id
  private async getBotOpenId(): Promise<string | null> {
    if (this.botOpenId) return this.botOpenId;
    const token = await this.getAccessToken();
    if (!token) return null;
    try {
      const res = await axios.get(`${FEISHU_API_BASE}/open-apis/bot/v3/info`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.data.code === 0) {
        this.botOpenId = res.data.bot?.open_id || null;
        logger.info({ botOpenId: this.botOpenId }, 'Feishu: bot open_id fetched');
        return this.botOpenId;
      }
    } catch (err) {
      logger.error({ err }, 'Feishu: error fetching bot open_id');
    }
    return null;
  }

  // 获取 tenant_access_token
  private async getAccessToken(): Promise<string | null> {
    if (this.accessToken && Date.now() < this.tokenExpireTime) {
      return this.accessToken;
    }
    try {
      const res = await axios.post(
        `${FEISHU_API_BASE}/open-apis/auth/v3/tenant_access_token/internal`,
        { app_id: this.appId, app_secret: this.appSecret }
      );
      if (res.data.code === 0) {
        this.accessToken = res.data.tenant_access_token;
        this.tokenExpireTime = Date.now() + (res.data.expire - 300) * 1000;
        logger.info('Feishu: access_token refreshed');
        return this.accessToken;
      } else {
        logger.error({ data: res.data }, 'Feishu: failed to get access_token');
        return null;
      }
    } catch (err) {
      logger.error({ err }, 'Feishu: error getting access_token');
      return null;
    }
  }

  async connect(): Promise<void> {
    const app = express();
    app.use(express.json());

    // 事件回调接口
    app.post('/webhook/event', async (req: Request, res: Response) => {
      const body = req.body;

      logger.info({ body: JSON.stringify(body).slice(0, 200) }, 'Feishu: incoming request');

      // URL 验证（v1.0 和 v2.0 都支持）
      if (body.type === 'url_verification' || body.challenge) {
        logger.info('Feishu: URL verification request');
        return res.json({ challenge: body.challenge });
      }

      // 兼容 v1.0 和 v2.0 事件格式
      const isV2 = body.schema === '2.0';
      const eventType = isV2 ? body.header?.event_type : body.event?.type;
      const event = body.event;
      const type = body.type;

      // 处理消息事件
      const isMessageEvent =
        (isV2 && eventType === 'im.message.receive_v1') ||
        (!isV2 && type === 'event_callback' && eventType === 'im.message.receive_v1');

      if (isMessageEvent) {
        res.json({ code: 0 }); // 立即返回

        try {
          const { sender, message } = event;
          const openId = sender?.sender_id?.open_id;
          const chatId = message?.chat_id;
          const messageType = message?.message_type;
          const chatType = message?.chat_type; // 'p2p' or 'group'

          if (messageType !== 'text' || !openId) return;

          const content = JSON.parse(message.content);
          let text: string = content.text || '';

          // 群聊中，只有 @机器人 才响应
          const isGroupChat = chatType === 'group';
          if (isGroupChat) {
            const botId = await this.getBotOpenId();
            const mentions: any[] = message.mentions || [];
            logger.info({ botId, mentions: JSON.stringify(mentions) }, 'Feishu: group message mentions');
            if (!botId) {
              logger.warn('Feishu: cannot determine bot open_id, ignoring group message');
              return;
            }
            const botMentioned = mentions.some((m: any) => m.id?.open_id === botId);
            if (!botMentioned) return;
          }

          // 去掉 @机器人 的 mention 标签，保留其他文本
          text = text.replace(/@\S+/g, '').trim();

          // prepend trigger word so the trigger check passes
          text = `@${ASSISTANT_NAME} ${text}`.trim();

          if (!text) return;

          const chatJid = `feishu:${chatId || openId}`;
          const timestamp = new Date().toISOString();

          // Store message ID for reply
          this.lastMessageId.set(chatJid, message.message_id);

          logger.info({ openId, chatJid, text }, 'Feishu: message received');

          // 先通知 chat metadata（必须在 onMessage 之前）
          // 使用 chatId 作为名称，如果没有则使用 openId
          const chatName = chatId ? `飞书群聊-${chatId}` : `飞书私聊-${openId}`;
          this.opts.onChatMetadata(chatJid, timestamp, chatName, 'feishu', !!chatId);

          // 再通知 nanoclaw 有新消息
          this.opts.onMessage(chatJid, {
            id: message.message_id || `feishu-${Date.now()}`,
            chat_jid: chatJid,
            sender: openId,
            sender_name: sender?.sender_id?.user_id || openId,
            content: text,
            timestamp,
          });

        } catch (err) {
          logger.error({ err }, 'Feishu: error handling message event');
        }
        return;
      }

      res.json({ code: 0 });
    });

    // 健康检查
    app.get('/health', (_req: Request, res: Response) => {
      res.json({ status: 'ok', channel: 'feishu' });
    });

    await new Promise<void>((resolve) => {
      this.httpServer = app.listen(this.port, () => {
        logger.info({ port: this.port }, 'Feishu: webhook server started');
        resolve();
      });
    });

    // 获取初始 token
    await this.getAccessToken();
    this.connected = true;
    logger.info('Feishu channel connected');
  }

  // 将 markdown 转换为飞书友好的纯文本
  private stripMarkdown(text: string): string {
    return text
      .replace(/\*\*(.+?)\*\*/g, '$1')       // **bold** → bold
      .replace(/\*(.+?)\*/g, '$1')            // *italic* → italic
      .replace(/`{3}[\s\S]*?`{3}/g, (m) =>   // ```code block``` → 保留内容
        m.replace(/`{3}\w*\n?/g, '').trim()
      )
      .replace(/`(.+?)`/g, '$1')              // `inline code` → inline code
      .replace(/^#{1,6}\s+/gm, '')            // # heading → heading
      .replace(/^\s*[-*+]\s+/gm, '• ')        // - item → • item
      .replace(/^\s*\d+\.\s+/gm, (m) => m)   // 1. item → 保留
      .replace(/\[(.+?)\]\((.+?)\)/g, '$1: $2') // [text](url) → text: url
      .replace(/\n{3,}/g, '\n\n')             // 多余空行压缩
      .trim();
  }

  async sendMessage(jid: string, text: string): Promise<void> {
    const token = await this.getAccessToken();
    if (!token) {
      logger.error('Feishu: no access token, cannot send message');
      return;
    }

    // 转换 markdown 为纯文本
    text = this.stripMarkdown(text);

    // jid 格式: feishu:<chat_id 或 open_id>
    const id = jid.replace(/^feishu:/, '');
    const isOpenId = id.startsWith('ou_');
    const receiveIdType = isOpenId ? 'open_id' : 'chat_id';

    // Get last message ID for reply
    const replyMessageId = this.lastMessageId.get(jid);

    // 飞书单条消息最大长度限制
    const MAX_LENGTH = 4000;
    const chunks = [];
    for (let i = 0; i < text.length; i += MAX_LENGTH) {
      chunks.push(text.slice(i, i + MAX_LENGTH));
    }

    for (const chunk of chunks) {
      try {
        let res;
        if (replyMessageId) {
          // Use reply API to quote the original message
          res = await axios.post(
            `${FEISHU_API_BASE}/open-apis/im/v1/messages/${replyMessageId}/reply`,
            {
              msg_type: 'text',
              content: JSON.stringify({ text: chunk }),
            },
            {
              headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
              },
            }
          );
        } else {
          // Fallback to regular send
          res = await axios.post(
            `${FEISHU_API_BASE}/open-apis/im/v1/messages`,
            {
              receive_id: id,
              msg_type: 'text',
              content: JSON.stringify({ text: chunk }),
            },
            {
              headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
              },
              params: { receive_id_type: receiveIdType },
            }
          );
        }
        if (res.data.code === 0) {
          logger.info({ jid }, 'Feishu: message sent');
        } else {
          logger.error({ jid, data: res.data }, 'Feishu: failed to send message');
        }
      } catch (err) {
        logger.error({ jid, err }, 'Feishu: error sending message');
      }
    }
  }

  isConnected(): boolean {
    return this.connected;
  }

  ownsJid(jid: string): boolean {
    return jid.startsWith('feishu:');
  }

  async disconnect(): Promise<void> {
    if (this.httpServer) {
      this.httpServer.close();
      this.httpServer = null;
    }
    this.connected = false;
    logger.info('Feishu channel disconnected');
  }
}

registerChannel('feishu', (opts: ChannelOpts) => {
  const envVars = readEnvFile([
    'FEISHU_APP_ID',
    'FEISHU_APP_SECRET',
    'FEISHU_WEBHOOK_PORT',
  ]);
  const appId = process.env.FEISHU_APP_ID || envVars.FEISHU_APP_ID || '';
  const appSecret = process.env.FEISHU_APP_SECRET || envVars.FEISHU_APP_SECRET || '';
  const port = parseInt(process.env.FEISHU_WEBHOOK_PORT || envVars.FEISHU_WEBHOOK_PORT || '3000', 10);

  if (!appId || !appSecret) {
    logger.warn('Feishu: FEISHU_APP_ID or FEISHU_APP_SECRET not set, skipping');
    return null;
  }

  return new FeishuChannel(appId, appSecret, port, opts);
});
```

### 2.2 Register the channel barrel import

Append `import './feishu.js';` to `src/channels/index.ts` under a `// feishu` comment. Check if it already exists first.

### 2.3 Create `container/skills/feishu-docs/SKILL.md`

Write this file:

```markdown
---
name: feishu-docs
description: Read Feishu (Lark) documents, wikis, spreadsheets, and bitables via API. ALWAYS use this tool for feishu.cn or xfchat.iflytek.com links instead of agent-browser. Works with private documents if the app has access.
allowed-tools: Bash(feishu-docs:*)
---

# Feishu Document Reader

Read content from Feishu (Lark) documents, wikis, spreadsheets, and bitables using the Feishu API.

## When to Use

**ALWAYS use this tool when you see:**
- `feishu.cn/docx/` or `feishu.cn/docs/` URLs
- `feishu.cn/wiki/` or `xfchat.iflytek.com/wiki/` URLs
- `feishu.cn/sheets/` or `xfchat.iflytek.com/sheets/` URLs
- `feishu.cn/base/` or `xfchat.iflytek.com/base/` URLs (bitable/多维表格)
- Any Feishu document, wiki, spreadsheet, or bitable link

**DO NOT use agent-browser for Feishu documents** - it will hit login pages. Use this API tool instead.

## Commands

\`\`\`bash
# Read a single document (docx, wiki page, spreadsheet, bitable)
feishu-docs read <url_or_id>

# Read ALL documents in a wiki knowledge base (递归读取所有文档)
feishu-docs read-all <wiki_url>

# List all nodes in a wiki knowledge base (show structure)
feishu-docs list <wiki_url>

# List files in cloud drive
feishu-docs list-drive [folder_token]
\`\`\`

## Examples

\`\`\`bash
# Read a wiki page
feishu-docs read https://www.xfchat.iflytek.com/wiki/LXqCwW602iY3NukDUygrpJrpzEh

# Read ALL documents in a knowledge base
feishu-docs read-all https://www.xfchat.iflytek.com/wiki/LF4twk5YhiW4BWkyhFGrX2VVzuf

# List knowledge base structure
feishu-docs list https://www.xfchat.iflytek.com/wiki/LF4twk5YhiW4BWkyhFGrX2VVzuf

# Read a spreadsheet
feishu-docs read https://www.xfchat.iflytek.com/sheets/shtcnXXXXXX

# Read a bitable (多维表格)
feishu-docs read https://www.xfchat.iflytek.com/base/bascnXXXXXX
\`\`\`

## Notes

- Uses Feishu API with app credentials (not browser automation)
- Works with private documents if the app has been granted access
- `read-all` recursively reads every document in a wiki — use when asked to "read the whole knowledge base"
- Supports both Feishu and Lark platforms (feishu.cn and xfchat.iflytek.com)
```

### 2.4 Create `container/skills/feishu-docs/feishu-docs`

Write this executable bash script (`chmod +x` after creating):

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

# Extract doc ID and type from URL or raw ID
# Returns: "id type"
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
        if [[ "$input" =~ ^sht ]]; then
            echo "$input sheets"
        elif [[ "$input" =~ ^bascn ]] || [[ "$input" =~ ^basc ]]; then
            echo "$input bitable"
        else
            echo "$input docx"
        fi
    fi
}

read_docx() {
    local doc_id="$1" token="$2"
    local response
    response=$(curl -s "${FEISHU_API_BASE}/open-apis/docx/v1/documents/${doc_id}/raw_content" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code
    code=$(echo "$response" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading docx: $(echo "$response" | jq -r '.msg')" >&2
        exit 1
    fi
    echo "$response" | jq -r '.data.content // ""'
}

read_wiki() {
    local node_token="$1" token="$2"
    local node_resp
    node_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/get_node?token=${node_token}" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code
    code=$(echo "$node_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading wiki node: $(echo "$node_resp" | jq -r '.msg')" >&2
        exit 1
    fi
    local obj_token obj_type title
    obj_token=$(echo "$node_resp" | jq -r '.data.node.obj_token')
    obj_type=$(echo "$node_resp" | jq -r '.data.node.obj_type')
    title=$(echo "$node_resp" | jq -r '.data.node.title')
    echo "# $title"
    echo ""
    if [[ "$obj_type" == "docx" ]]; then
        read_docx "$obj_token" "$token"
    else
        echo "(obj_type: $obj_type, obj_token: $obj_token)"
    fi
}

list_wiki_nodes() {
    local space_id="$1" parent_token="$2" token="$3" depth="$4"
    local page_token=""
    while true; do
        local url="${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/${space_id}/nodes?parent_node_token=${parent_token}&page_size=50"
        [[ -n "$page_token" ]] && url="${url}&page_token=${page_token}"
        local resp
        resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        local code
        code=$(echo "$resp" | jq -r '.code // 1')
        if [[ "$code" != "0" ]]; then
            echo "  (permission denied for node $parent_token)" >&2
            return
        fi
        local items
        items=$(echo "$resp" | jq -c '.data.items[]?' 2>/dev/null || true)
        while IFS= read -r item; do
            [[ -z "$item" ]] && continue
            local ntitle ntoken ntype
            ntitle=$(echo "$item" | jq -r '.title')
            ntoken=$(echo "$item" | jq -r '.node_token')
            ntype=$(echo "$item" | jq -r '.obj_type')
            local indent
            indent=$(printf '%*s' $((depth * 2)) '')
            echo "${indent}- [${ntype}] ${ntitle} (${ntoken})"
            list_wiki_nodes "$space_id" "$ntoken" "$token" $((depth + 1))
        done <<< "$items"
        local has_more
        has_more=$(echo "$resp" | jq -r '.data.has_more // false')
        [[ "$has_more" != "true" ]] && break
        page_token=$(echo "$resp" | jq -r '.data.page_token // ""')
        [[ -z "$page_token" ]] && break
    done
}

list_wiki() {
    local node_token="$1" token="$2"
    local node_resp
    node_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/get_node?token=${node_token}" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local space_id title
    space_id=$(echo "$node_resp" | jq -r '.data.node.space_id')
    title=$(echo "$node_resp" | jq -r '.data.node.title')
    echo "Knowledge base: $title (space: $space_id)"
    echo ""
    list_wiki_nodes "$space_id" "$node_token" "$token" 0
}

read_wiki_nodes_content() {
    local space_id="$1" parent_token="$2" token="$3"
    local page_token=""
    while true; do
        local url="${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/${space_id}/nodes?parent_node_token=${parent_token}&page_size=50"
        [[ -n "$page_token" ]] && url="${url}&page_token=${page_token}"
        local resp
        resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        local code
        code=$(echo "$resp" | jq -r '.code // 1')
        [[ "$code" != "0" ]] && return
        local items
        items=$(echo "$resp" | jq -c '.data.items[]?' 2>/dev/null || true)
        while IFS= read -r item; do
            [[ -z "$item" ]] && continue
            local ntitle ntoken ntype
            ntitle=$(echo "$item" | jq -r '.title')
            ntoken=$(echo "$item" | jq -r '.node_token')
            ntype=$(echo "$item" | jq -r '.obj_type')
            echo ""
            echo "## $ntitle"
            echo ""
            if [[ "$ntype" == "docx" ]]; then
                read_wiki "$ntoken" "$token"
            else
                echo "(type: $ntype)"
            fi
            echo ""
            echo "---"
            read_wiki_nodes_content "$space_id" "$ntoken" "$token"
        done <<< "$items"
        local has_more
        has_more=$(echo "$resp" | jq -r '.data.has_more // false')
        [[ "$has_more" != "true" ]] && break
        page_token=$(echo "$resp" | jq -r '.data.page_token // ""')
        [[ -z "$page_token" ]] && break
    done
}

read_wiki_all() {
    local node_token="$1" token="$2"
    local node_resp
    node_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/wiki/v2/spaces/get_node?token=${node_token}" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local space_id title
    space_id=$(echo "$node_resp" | jq -r '.data.node.space_id')
    title=$(echo "$node_resp" | jq -r '.data.node.title')
    echo "# Knowledge base: $title"
    echo ""
    read_wiki "$node_token" "$token"
    echo ""
    echo "---"
    read_wiki_nodes_content "$space_id" "$node_token" "$token"
}

read_sheets() {
    local spreadsheet_token="$1" token="$2"
    local sheets_resp
    sheets_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/sheets/v3/spreadsheets/${spreadsheet_token}/sheets/query" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code
    code=$(echo "$sheets_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading spreadsheet: $(echo "$sheets_resp" | jq -r '.msg')" >&2
        exit 1
    fi
    local sheets
    sheets=$(echo "$sheets_resp" | jq -c '.data.sheets[]?')
    while IFS= read -r sheet; do
        [[ -z "$sheet" ]] && continue
        local sid stitle
        sid=$(echo "$sheet" | jq -r '.sheet_id')
        stitle=$(echo "$sheet" | jq -r '.title')
        echo "## Sheet: $stitle (id: $sid)"
        echo ""
        local range="${sid}!A1:ZZ1000"
        local val_resp
        val_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/sheets/v2/spreadsheets/${spreadsheet_token}/values/${range}" \
            -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        local vcode
        vcode=$(echo "$val_resp" | jq -r '.code // 1')
        if [[ "$vcode" == "0" ]]; then
            echo "$val_resp" | jq -r '
                .data.valueRange.values[]? |
                map(if . == null then "" else (. | tostring) end) |
                join("\t")
            '
        else
            echo "(Error reading sheet: $(echo "$val_resp" | jq -r '.msg'))"
        fi
        echo ""
    done <<< "$sheets"
}

read_bitable() {
    local app_token="$1" token="$2"
    local tables_resp
    tables_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables?page_size=100" \
        -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code
    code=$(echo "$tables_resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error reading bitable: $(echo "$tables_resp" | jq -r '.msg')" >&2
        exit 1
    fi
    local tables
    tables=$(echo "$tables_resp" | jq -c '.data.items[]?')
    while IFS= read -r table; do
        [[ -z "$table" ]] && continue
        local tid tname
        tid=$(echo "$table" | jq -r '.table_id')
        tname=$(echo "$table" | jq -r '.name')
        echo "## Table: $tname (id: $tid)"
        echo ""
        local fields_resp
        fields_resp=$(curl -s "${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables/${tid}/fields?page_size=100" \
            -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
        echo "$fields_resp" | jq -r '[.data.items[]?.field_name] | join("\t")' 2>/dev/null || true
        local page_token=""
        while true; do
            local url="${FEISHU_API_BASE}/open-apis/bitable/v1/apps/${app_token}/tables/${tid}/records?page_size=100"
            [[ -n "$page_token" ]] && url="${url}&page_token=${page_token}"
            local rec_resp
            rec_resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
            echo "$rec_resp" | jq -r '
                .data.items[]?.fields |
                to_entries |
                map(.value |
                    if type == "array" then map(if type == "object" then (.text // .name // tostring) else tostring end) | join(", ")
                    elif type == "object" then (.text // .value // tostring)
                    else tostring end) |
                join("\t")
            ' 2>/dev/null || true
            local has_more
            has_more=$(echo "$rec_resp" | jq -r '.data.has_more // false')
            [[ "$has_more" != "true" ]] && break
            page_token=$(echo "$rec_resp" | jq -r '.data.page_token // ""')
            [[ -z "$page_token" ]] && break
        done
        echo ""
    done <<< "$tables"
}

list_drive() {
    local folder_token="${1:-}" token="$2"
    local url="${FEISHU_API_BASE}/open-apis/drive/v1/files?page_size=50"
    [[ -n "$folder_token" ]] && url="${url}&folder_token=${folder_token}"
    local resp
    resp=$(curl -s "$url" -H "Authorization: Bearer ${token}" -H "User-Agent: Mozilla/5.0")
    local code
    code=$(echo "$resp" | jq -r '.code // 1')
    if [[ "$code" != "0" ]]; then
        echo "Error listing drive: $(echo "$resp" | jq -r '.msg')" >&2
        exit 1
    fi
    echo "$resp" | jq -r '.data.files[]? | "\(.type)\t\(.name)\t\(.token)"'
}

usage() {
    cat >&2 <<EOF
Usage:
  feishu-docs read <url_or_id>          Read a document (docx, wiki, sheet, bitable)
  feishu-docs read-all <wiki_url>       Read ALL documents in a wiki knowledge base
  feishu-docs list <wiki_url>           List all nodes in a wiki knowledge base
  feishu-docs list-drive [folder_token] List files in cloud drive

Supported URL types:
  https://*.xfchat.iflytek.com/wiki/TOKEN     Wiki page
  https://*.xfchat.iflytek.com/docx/TOKEN     Document
  https://*.xfchat.iflytek.com/sheets/TOKEN   Spreadsheet
  https://*.xfchat.iflytek.com/base/TOKEN     Bitable (多维表格)
EOF
    exit 1
}

main() {
    [[ $# -lt 1 ]] && usage

    local command="$1"
    local input="${2:-}"

    check_credentials
    local token
    token=$(get_access_token)

    case "$command" in
        read)
            [[ -z "$input" ]] && usage
            local id_info
            id_info=$(extract_id "$input")
            local doc_id doc_type
            doc_id=$(echo "$id_info" | cut -d' ' -f1)
            doc_type=$(echo "$id_info" | cut -d' ' -f2)
            echo "Reading feishu $doc_type: $doc_id" >&2
            case "$doc_type" in
                docx)    read_docx "$doc_id" "$token" ;;
                wiki)    read_wiki "$doc_id" "$token" ;;
                sheets)  read_sheets "$doc_id" "$token" ;;
                bitable) read_bitable "$doc_id" "$token" ;;
                folder)  list_drive "$doc_id" "$token" ;;
                *) echo "Error: Unknown type $doc_type" >&2; exit 1 ;;
            esac
            ;;
        read-all)
            [[ -z "$input" ]] && usage
            local id_info
            id_info=$(extract_id "$input")
            local doc_id doc_type
            doc_id=$(echo "$id_info" | cut -d' ' -f1)
            doc_type=$(echo "$id_info" | cut -d' ' -f2)
            [[ "$doc_type" != "wiki" ]] && { echo "Error: read-all only supports wiki URLs" >&2; exit 1; }
            echo "Reading all wiki documents from: $doc_id" >&2
            read_wiki_all "$doc_id" "$token"
            ;;
        list)
            [[ -z "$input" ]] && usage
            local id_info
            id_info=$(extract_id "$input")
            local doc_id doc_type
            doc_id=$(echo "$id_info" | cut -d' ' -f1)
            doc_type=$(echo "$id_info" | cut -d' ' -f2)
            [[ "$doc_type" != "wiki" ]] && { echo "Error: list only supports wiki URLs" >&2; exit 1; }
            list_wiki "$doc_id" "$token"
            ;;
        list-drive)
            list_drive "${input:-}" "$token"
            ;;
        *)
            usage
            ;;
    esac
}

main "$@"
```

After writing the file, make it executable:

```bash
chmod +x container/skills/feishu-docs/feishu-docs
```

### 2.5 Patch `src/container-runner.ts`

Find the line that sets `ANTHROPIC_BASE_URL` env var (near `args.push('-e', \`ANTHROPIC_BASE_URL=...`). After that block, add:

```typescript
  // Pass Feishu credentials for feishu-docs tool
  const feishuEnvVars = readEnvFile(['FEISHU_APP_ID', 'FEISHU_APP_SECRET']);
  if (feishuEnvVars.FEISHU_APP_ID) {
    args.push('-e', `FEISHU_APP_ID=${feishuEnvVars.FEISHU_APP_ID}`);
    logger.debug({ appId: feishuEnvVars.FEISHU_APP_ID }, 'Passing FEISHU_APP_ID to container');
  }
  if (feishuEnvVars.FEISHU_APP_SECRET) {
    args.push('-e', `FEISHU_APP_SECRET=${feishuEnvVars.FEISHU_APP_SECRET}`);
    logger.debug('Passing FEISHU_APP_SECRET to container');
  }
```

If `readEnvFile` is not already imported, add `import { readEnvFile } from './env.js';` at the top. Check first — it may already be imported.

### 2.6 Add environment variables to `.env.example`

Append these lines to `.env.example` if they don't already exist:

```
# Feishu (飞书) - get from open.feishu.cn or your private deployment
FEISHU_APP_ID=
FEISHU_APP_SECRET=
FEISHU_WEBHOOK_PORT=3000
```

### Validate code changes

```bash
npm run build
```

Build must be clean before proceeding.

## Phase 3: Setup

### Create Feishu App (if needed)

If the user doesn't have app credentials, tell them:

> I need you to create a Feishu app:
>
> 1. Go to your Feishu Open Platform (open.feishu.cn or your private deployment's open platform)
> 2. Create a new **Enterprise Self-built App** (企业自建应用)
> 3. Enable **Bot** capability (机器人能力) under **App Capabilities**
> 4. Under **Permissions & Scopes** (权限管理), add:
>    - `im:message:send_v2` (send messages)
>    - `im:message:receive` (receive messages)
>    - `im:message` (message read/write)
>    - `im:chat:readonly` (read chat info)
>    - Optional for feishu-docs: `docx:document:readonly`, `wiki:wiki:readonly`, `sheets:spreadsheet`, `bitable:app:readonly`, `drive:drive:readonly`
> 5. Under **Event Subscriptions** (事件订阅):
>    - Choose "Send events to developer server" (将事件发送至开发者服务器)
>    - Add event: `im.message.receive_v1` (receive messages)
>    - The webhook URL will be configured after NanoClaw starts
> 6. Copy the **App ID** and **App Secret** from the app's **Credentials** page

Wait for the user to provide the App ID and App Secret.

### Configure API Base URL

**IMPORTANT:** The default `FEISHU_API_BASE` in `src/channels/feishu.ts` is set to `https://open.xfchat.iflytek.com` (Iflytek private Feishu).

Ask the user which Feishu deployment they use:
- **Public Feishu**: Change `FEISHU_API_BASE` to `https://open.feishu.cn`
- **Iflytek Private Feishu**: Keep as `https://open.xfchat.iflytek.com`
- **Other Private Deployment**: Set to their open platform URL

Also update `FEISHU_API_BASE` in `container/skills/feishu-docs/feishu-docs` to match.

### Configure environment

Add to `.env`:

```bash
FEISHU_APP_ID=<their-app-id>
FEISHU_APP_SECRET=<their-app-secret>
FEISHU_WEBHOOK_PORT=3000
```

Sync to container environment:

```bash
mkdir -p data/env && cp .env data/env/env
```

### Configure webhook endpoint

Tell the user:

> After NanoClaw starts, configure the webhook URL on the Feishu Open Platform:
>
> 1. Go to your app's **Event Subscriptions** page
> 2. Set the **Request URL** to: `http://<your-server-ip>:<FEISHU_WEBHOOK_PORT>/webhook/event`
>
> **For servers not directly accessible from Feishu:**
> Use a tunnel like ngrok:
> ```bash
> ngrok http <FEISHU_WEBHOOK_PORT>
> ```
> Then use the ngrok HTTPS URL: `https://<ngrok-domain>/webhook/event`

### Build and restart

```bash
npm run build
launchctl kickstart -k gui/$(id -u)/com.nanoclaw  # macOS
# Linux: systemctl --user restart nanoclaw
```

## Phase 4: Registration

### Get Chat ID

Tell the user:

> 1. Send a message to the bot in Feishu (1-on-1 or in a group where the bot is added)
> 2. Check the NanoClaw logs for the incoming message:
>    ```bash
>    tail -f logs/nanoclaw.log | grep "Feishu: message received"
>    ```
> 3. The JID format is:
>    - 1-on-1 chat: `feishu:ou_xxxxx` (user's open_id)
>    - Group chat: `feishu:oc_xxxxx` (chat_id)

Wait for the user to provide the chat JID.

### Register the chat

For a main chat (responds to all messages):

```bash
npx tsx setup/index.ts --step register -- --jid "feishu:<chat-id>" --name "<chat-name>" --folder "feishu_main" --trigger "@${ASSISTANT_NAME}" --channel feishu --no-trigger-required --is-main
```

For additional chats (trigger-only, responds when @mentioned):

```bash
npx tsx setup/index.ts --step register -- --jid "feishu:<chat-id>" --name "<chat-name>" --folder "feishu_<group-name>" --trigger "@${ASSISTANT_NAME}" --channel feishu
```

## Phase 5: Verify

### Test the connection

Tell the user:

> Send a message to your registered Feishu chat:
> - For main chat: Any message works
> - For group chat: @mention the bot in the message
>
> The bot should respond within a few seconds.

### Check logs if needed

```bash
tail -f logs/nanoclaw.log
```

## Troubleshooting

### Bot not responding

Check:
1. `FEISHU_APP_ID` and `FEISHU_APP_SECRET` are set in `.env` AND synced to `data/env/env`
2. Chat is registered: `sqlite3 store/messages.db "SELECT * FROM registered_groups WHERE jid LIKE 'feishu:%'"`
3. Webhook URL is correctly configured on the Feishu Open Platform
4. The event `im.message.receive_v1` is subscribed
5. Service is running: `launchctl list | grep nanoclaw` (macOS) or `systemctl --user status nanoclaw` (Linux)

### Bot only responds when @mentioned in groups

This is by design. In group chats, the bot only responds when @mentioned to avoid noise.

### Webhook URL verification fails

- Ensure NanoClaw is running and the webhook port is accessible
- The `/webhook/event` endpoint handles both v1.0 and v2.0 URL verification automatically

### feishu-docs returns errors

- Verify the app has document-related permissions
- Check that `FEISHU_APP_ID` and `FEISHU_APP_SECRET` are passed to the container (check container-runner.ts)
- Ensure the API base URL matches your deployment

### Health check

```bash
curl http://localhost:<FEISHU_WEBHOOK_PORT>/health
# Expected: {"status":"ok","channel":"feishu"}
```

## Removal

1. Delete `src/channels/feishu.ts`
2. Remove `import './feishu.js'` from `src/channels/index.ts`
3. Remove `FEISHU_APP_ID`, `FEISHU_APP_SECRET`, `FEISHU_WEBHOOK_PORT` from `.env`
4. Remove Feishu registrations: `sqlite3 store/messages.db "DELETE FROM registered_groups WHERE jid LIKE 'feishu:%'"`
5. Remove feishu-docs: `rm -rf container/skills/feishu-docs/`
6. Remove Feishu credential passing from `src/container-runner.ts`
7. Rebuild: `npm run build && launchctl kickstart -k gui/$(id -u)/com.nanoclaw` (macOS) or `npm run build && systemctl --user restart nanoclaw` (Linux)
