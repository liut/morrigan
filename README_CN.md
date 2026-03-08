# Morign

AI 聊天应用知识库系统的后端实现。

## 功能特性

- 从表格（CSV）导入知识库文档，保存到 PostgreSQL
- 基于文档标题和内容，使用 Embedding API 生成文档向量
- 使用 Embedding API 生成文档向量
- 基于向量搜索实现高质量问答
- 欢迎消息和预设消息
- 基于 Redis 的对话历史
- RESTful API
- 支持 text/event-stream 流式响应
- OAuth2 客户端登录认证
- 内置 MCP（Model Context Protocol）工具支持

## 支持的前端

<details>
 <summary>chatgpt-svelte 基于 Svelte  ⤸</summary>

 ![chatgpt-svelte](./docs/screen-svelte-s.png)

> forked: https://github.com/liut/chatgpt-svelte

</details>

<details>
 <summary>chatgpt-web 基于 Vue.js  ⤸</summary>

 ![chatgpt-web](./docs/screen-web-s.png)

> forked: https://github.com/liut/chatgpt-web

</details>

## API 接口

### 获取欢迎消息和新会话 ID

<details>
 <summary><code>GET</code> <code><b>/api/welcome</b></code></summary>

##### 参数

> 无

##### 响应

> | http 状态码 | content-type                      | 响应                                           |
> |---------------|-----------------------------------|---------------------------------------------------------------------|
> | `200`         | `application/json`        | `{"message": "welcome message", "id": "new-cid"}`                                         |


</details>

### 获取已验证或已登录用户的信息

<details>
 <summary><code>GET</code> <code><b>/api/me</b></code></summary>

##### 参数

> 无

##### 响应

> | http 状态码 | content-type                      | 响应                                           |
> |---------------|-----------------------------------|---------------------------------------------------------------------|
> | `200`         | `application/json`        | `{"data": {"avatar": "", "name": "name", "uid": "uid"}}`                                         |
> | `401`         | `application/json`        | `{"error": "", "message": ""}`                                         |


</details>

### 发送聊天消息并返回流式响应

<details>
 <summary><code>POST</code> <code><b>/api/chat-sse</b></code> 或 <code><b>/api/chat</b></code> 搭配 <code>{stream: true}</code></summary>

##### 参数

> | 名称       |  类型     | 数据类型      | 描述                         |
> |------------|-----------|----------------|-------------------------------------|
> | `csid`     |  可选 | string       | 会话 ID        |
> | `prompt`   |  必填 | string       | 提问消息        |
> | `stream`   |  可选 |  bool        | 启用 event-stream，在 <code><b>/api/chat-sse</b></code> 中强制开启       |


##### 响应

> | http 状态码 | content-type               | 响应                                           |
> |---------------|----------------------------|----------------------------------------------------|
> | `200`         | `text/event-stream`        | `{"delta": "消息片段", "id": "会话 ID"}`                                          |
> | `401`         | `application/json`        | `{"status": "Unauthorized", "message": ""}`                                         |


</details>

<details>
 <summary><code>POST</code> <code><b>/api/chat-process</b></code> 仅适用于 chatgpt-web</summary>

##### 参数

> | 名称        |  类型     | 数据类型      | 描述                         |
> |-------------|-----------|----------------|-------------------------------------|
> | `prompt`    | 必填  |    string      | 提问消息        |
> | `options`   | 可选  |    object      | <code>{ conversationId: "" }</code>    |


##### 响应

> | http 状态码 | content-type                    | 响应                                           |
> |---------------|---------------------------------|-----------------------------------------------------|
> | `200`         | `application/octet-stream`      | `{"delta": "消息片段", "text": "完整消息", "conversationId": ""}`                                          |
> | `401`         | `application/json`        | `{"status": "Unauthorized", "message": ""}`                                         |


</details>

## 快速开始

```bash

# 安装依赖（包含 forego 用于加载环境变量）
make deps

# 或手动执行
go mod tidy
go install github.com/ddollar/forego@latest

# 从示例创建 .env 并配置（更新 PG/Redis URL、API Key 等）
test -e .env || cp .env.example .env

# 使用 .env 中的环境变量启动服务器
forego start

# 或直接运行（开发模式）
forego run go run . web

```

### 准备 YAML 预设数据文件

```yaml

welcome:
  content: 你好，我是你的虚拟助手。有什么可以帮助你的吗？
  role: assistant

messages:
  - content: 你是一个有用的助手。
    role: system
  - content: 我的生日是什么时候？
    role: user
  - content: 我怎么会知道？
    role: assistant

 # 更多消息

```

### 准备数据库

```sql
CREATE USER morrigan WITH LOGIN PASSWORD 'mydbusersecret';
CREATE DATABASE morrigan WITH OWNER = morrigan ENCODING = 'UTF8';
GRANT ALL PRIVILEGES ON DATABASE morrigan to morrigan;


\c morrigan

# 可选：安装扩展 https://github.com/pgvector/pgvector
CREATE EXTENSION vector;

```

### 命令行用法

```plan

USAGE:
   morign [全局选项] 命令 [命令选项] [参数...]

COMMANDS:
   usage, env                   显示用法
   initdb                       初始化数据库模式
   import                       从 csv 导入文档
   export                       导出文档到 csv/jsonl
   embedding, embedding-prompt  读取提示文档并生成嵌入
   agent, llm, chat            测试 LLM 功能
   web, run                     运行 Web 服务器
   version, ver                 显示版本
   help, h                      显示命令帮助

GLOBAL OPTIONS:
   --help, -h  显示帮助

```

#### Agent 命令

测试 LLM 功能的命令行工具：

```bash
# 非流式对话
./morign agent -m "你好"

# 流式对话
./morign agent -m "你好" -s

# 显示详细日志
./morign agent -m "你好" -v
```

参数：
- `-m, --message`: 发送的消息 (必填)
- `-s, --stream`: 启用流式响应
- `-v, --verbose`: 显示日志 (默认关闭)

### 使用环境变量配置

#### 查看所有本地配置
```bash
./morign usage
```

示例：

```plan
# Interact provider（对话）
MORIGN_INTERACT_API_KEY=sk-xxx
MORIGN_INTERACT_MODEL=gpt-4o-mini
MORIGN_INTERACT_URL=https://api.openai.com/v1

# Embedding provider（向量嵌入）
MORIGN_EMBEDDING_API_KEY=sk-xxx
MORIGN_EMBEDDING_MODEL=text-embedding-3-small

# Summarize provider（可选）
MORIGN_SUMMARIZE_API_KEY=sk-xxx
MORIGN_SUMMARIZE_MODEL=gpt-4o-mini

MORIGN_HTTP_LISTEN=:3002

# 可选：预设数据
MORIGN_PRESET_FILE=./data/preset.yaml

# 可选：OAuth2 登录
MORIGN_AUTH_REQUIRED=true
OAUTH_PREFIX=https://portal.my-company.xyz

# 可选：代理
HTTPS_PROXY=socks5://proxy.my-company.xyz:1081
```

#### 环境变量列表

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MORIGN_PG_STORE_DSN` | postgres://morrigan@localhost/morrigan | PostgreSQL 连接串 |
| `MORIGN_REDIS_URI` | redis://localhost:6379/1 | Redis 连接串 |
| `MORIGN_HTTP_LISTEN` | :5001 | HTTP 监听地址 |
| `MORIGN_AUTH_REQUIRED` | false | 是否启用认证 |
| `MORIGN_KEEPER_ROLE` | keeper | 写操作工具需要的角色 |
| `MORIGN_VECTOR_THRESHOLD` | 0.39 | 向量相似度阈值 |
| `MORIGN_VECTOR_LIMIT` | 5 | 向量匹配数量 |

#### Provider 配置（AI 服务）

每个 Provider 需要 `API_KEY` 和 `MODEL`，可选 `URL` 和 `TYPE` 用于自定义端点：

| Provider | 用途 | 必需变量 |
|----------|------|----------|
| `INTERACT` | 对话/补全 | `API_KEY`, `MODEL` |
| `EMBEDDING` | 向量嵌入 | `API_KEY`, `MODEL` |
| `SUMMARIZE` | 文本摘要 | `API_KEY`, `MODEL` |

支持的 Provider Type: `openai`, `anthropic`, `openrouter`, `ollama`

示例：
```
# Interact provider (支持 openai/anthropic/openrouter/ollama)
MORIGN_INTERACT_API_KEY=sk-xxx
MORIGN_INTERACT_MODEL=gpt-4o-mini
MORIGN_INTERACT_TYPE=openai  # 可选，默认 openai

# 使用 Anthropic
MORIGN_INTERACT_TYPE=anthropic

MORIGN_EMBEDDING_API_KEY=sk-xxx

MORIGN_EMBEDDING_API_KEY=sk-xxx
MORIGN_EMBEDDING_MODEL=text-embedding-3-small

MORIGN_SUMMARIZE_API_KEY=sk-xxx
MORIGN_SUMMARIZE_MODEL=gpt-4o-mini
```

> 提示：运行 `./morign usage` 可查看当前所有配置

## 数据生成步骤

1. 准备 CSV 文件作为语料库文档
2. 导入文档
3. 使用 Completion 从文档生成问答
4. 使用 Embedding 从问答生成提示和向量
5. 完成，开始聊天

### 文档 CSV 模板

| title      | heading     | content                                   |
|------------|-------------|-------------------------------------------|
| 我的公司 | 简介 | 一家伟大的公司源于一个天才的想法。 |
|            |             |                                           |

```bash
./morign initdb
./morign import mycompany.csv
./morign embedding
```

## 挂载前端资源

1. 进入前端项目目录
2. 构建前端页面和静态资源
3. 复制到 ./htdocs

示例：

```bash
cd ../chatgpt-svelte
npm run build
rsync -a --delete dist/* ../morign/htdocs/
cd -
```

在开发和调试阶段，你仍可以使用代理与前端项目协作。
