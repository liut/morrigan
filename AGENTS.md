# AGENTS.md - Morrigan 项目代理指南

## 项目概述

Morrigan 是一个基于 PostgreSQL + Redis 的知识库系统后端，用于 AI 聊天应用。

### 核心功能

- 从 CSV 导入知识库文档并存储到 PostgreSQL
- 使用 Embedding API 生成文档向量
- 基于向量搜索实现高质量问答
- 欢迎消息和预设消息
- 基于 Redis 的对话历史
- RESTful API + Server-Sent Events 流式响应
- OAuth2 客户端登录认证
- 内置 MCP (Model Context Protocol) 工具支持

## 技术栈

- **语言**: Go 1.25
- **Web 框架**: chi/v5
- **数据库**: PostgreSQL (含 vector 扩展)
- **缓存/会话**: Redis
- **LLM**: 自定义实现 (pkg/services/llm)，支持 OpenAI/Anthropic/OpenRouter/Ollama
- **MCP**: github.com/mark3labs/mcp-go
- **ORM**: github.com/uptrace/bun

## 项目结构

```
.
├── main.go                 # CLI 入口
├── data/                  # 预设数据 (YAML)
├── pkg/
│   ├── models/            # 数据模型 (qas/aigc/mcps/convo)
│   ├── services/          # 业务服务 (stores/tools/llm)
│   ├── settings/          # 配置管理
│   └── web/               # HTTP 服务
│       ├── api/           # API 处理器 (convo/user/convo_gen)
│       ├── i18n/         # 国际化
│       ├── resp/         # 响应处理
│       ├── routes/       # 路由注册和认证
│       ├── server.go     # 服务器主文件
│       └── util.go       # 工具函数
├── htdocs/                # 前端静态资源
└── docs/                  # API 文档
```

> 更多使用说明（命令行用法、环境变量配置）请参考 [README.md](./README.md)

## 编码规范

1. **错误处理**: 使用 `logger().Infow()` 或 `logger().Warnw()` 记录错误，避免直接 panic
2. **日志**: 使用 uber-zap 日志库，通过 `zlog.Get()` 获取 logger 实例
3. **配置**: 通过环境变量配置，使用 `envconfig` 库加载
4. **API 响应**: 使用 `render.JSON` 返回 JSON 响应，错误使用 `apiFail`
5. **类型断言**: 需要做类型断言时注意处理 `ok` 检查

## LLM 幻觉防护

项目已实现多层防护机制，避免 LLM 在知识库未命中时编造答案：

1. **系统提示约束** (`pkg/web/api/convo_basic.go`)
   - `dftSystemMsg`: 无工具场景，添加"不知道时诚实回答"约束
   - `dftToolsMsg`: 工具场景，添加类似约束

2. **检索未命中处理** (`pkg/web/api/handle_convo.go`)
   - 当知识库检索无结果时，添加明确的 System 消息提示

3. **检索命中处理** (`pkg/web/api/handle_convo.go`)
   - 命中文档拼接为单个 System 消息，格式：
     ```
     Found X relevant documents in the knowledge base:

     Heading1
     Content1

     Heading2
     Content2
     ...
     ```

## 关键文件说明

### main.go

- 定义 CLI 命令: `initdb`, `import`, `embedding`, `agent`, `web`, `usage`, `version`
- 启动 Web 服务器逻辑

### pkg/services/llm/

- LLM 统一客户端接口，支持多种 Provider
- `Client` 接口: `Chat`, `StreamChat`, `Generate`, `Embedding`
- 支持的 Provider: `openai`, `anthropic`, `openrouter`, `ollama`
- `anthropic.go`: Anthropic Provider 实现，包含流式响应支持

### pkg/web/server.go

- Web 服务器主文件
- `Service` 接口：定义 Serve() 和 Stop() 方法
- `New()`: 创建服务器实例
- `strapRouter()`: 注册路由，使用 routes.Routers() 加载子路由

### pkg/web/routes/registry.go

- 路由注册机制：使用 Strapper 接口解耦
- `Register()`: 各业务包 init() 时注册路由
- `Routers()`: 统一挂载所有注册路由

### pkg/web/routes/auth.go

- 认证中间件实现
- `Authzr()`: 获取 staffio Authorizer 单例
- `AuthMw()`: 返回认证中间件，支持 staffio 和简单 Token 两种模式

### pkg/web/api/api.go

- API 入口文件，注册 /api 路由
- `api` 结构体：包含 Storage、AI 客户端、工具注册器
- 集成限流器 (limiter) 防止 API 滥用
- 使用 `regHI()` 注册路由处理器

### pkg/web/api/convo_basic.go

- 聊天相关基础结构体定义
- `ChatRequest`, `ChatCompletionRequest`, `ChatMessage`
- `CompletionRequest`, `CompletionMessage`
- `ConversationResponse`
- 默认系统消息 `dftSystemMsg` 和工具消息 `dftToolsMsg`

### pkg/web/api/handle_convo.go

- 聊天请求处理核心逻辑
- `postChat`: 处理聊天请求，支持流式 SSE 响应和工具调用
- `getWelcome`: 获取欢迎消息
- `getHistory`: 获取会话历史
- `getTools`: 获取可用 MCP 工具列表
- `chatStreamResponse`: 处理流式响应和工具调用

### pkg/web/api/handle_user.go

- 用户认证相关处理
- `handleSession`: 获取会话信息 (支持 chatgpt-web)
- `handleVerify`: 验证 Token
- `handleMe`: 获取当前用户信息

### pkg/web/api/handle_convo_gen.go

- 会话 CRUD API (自动生成)
- `/api/convo/sessions`: 会话列表/获取/删除
- `/api/convo/messages`: 消息列表/获取/删除

### pkg/web/resp/resp.go

- 响应处理工具函数
- `Ok()`: 成功响应
- `Fail()`: 失败响应
- `Done`: 成功返回结构
- `Failure`: 失败返回结构

### pkg/web/i18n/locales.go

- 国际化支持
- `GetTag()`: 根据请求获取语言
- `GetPrinter()`: 获取消息打印器
- 支持中英文等多语言

### pkg/services/tools/registry.go

- 工具注册和管理中心
- `Tools()`: 返回所有工具（兼容旧接口）
- `ToolsFor(ctx)`: 根据上下文返回适合的工具列表，非 keeper 角色只返回公开工具

### pkg/services/tools/invokers.go

- MCP 工具实现：`callKBSearch`, `callKBCreate`, `callFetch`
- `fetchURL`: 网页内容获取，支持 HTML 转 Markdown
- 受限工具（如 `kb_create`）需要 keeper 角色

### pkg/services/stores/

- 使用 bun ORM 操作 PostgreSQL
- 问答文档的 CRUD 和向量匹配
- 对话历史存储在 Redis 中
- `IsKeeper(ctx)`: 检查用户是否具有 keeper 角色或 UID

### MCP 工具

项目内置以下 MCP 工具：

- `kb_search`: 知识库搜索
- `kb_create`: 创建知识库文档 (需要权限)
- `fetch`: 网页内容获取

## 数据库

### 必需表

- 问答文档表 (包含 title, heading, content, embedding 向量字段)
- 聊天日志表 (可选)

### 数据库扩展

- `pgvector` 向量数据库扩展

## 注意事项

1. 修改 API 结构体时注意 JSON tag 命名一致
2. 向量搜索需要先运行 `embedding` 命令生成向量
3. 使用 SSE 时需要实现 `http.Flusher` 接口
4. MCP 工具参数需要做类型断言，注意 JSON number 转为 float64
5. 写操作工具需要 keeper 角色，配置项 `KEEPER_ROLE`（默认 "keeper"）控制所需角色
