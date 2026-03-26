---
title: Add Platform Adapter Layer for Chat Platform Integration
type: feat
status: completed
date: 2026-03-25
---

# Add Platform Adapter Layer for Chat Platform Integration

## Overview

在 morrigan 中引入平台适配器层，使 AI 对话能力可以对接微信企业版(WeCom)、飞书等聊天平台。以 WeCom 为突破点，验证架构设计后扩展至其他平台。

## Problem Statement

当前 morrigan 只支持 HTTP API 方式的对话接入（通过前端）。需要支持将 AI 对话能力以 Bot 形式接入到企业常用的聊天平台（WeCom、飞书等），让用户可以在这些平台中直接与 AI 对话。

## Proposed Solution

### 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                      Chat Platforms                          │
│    WeCom    │    飞书    │   钉钉    │    Telegram    ...   │
└──────┬──────┴─────┬─────┴─────┬────┴────────┬──────────────┘
       │            │           │            │
       ▼            ▼           ▼            ▼
┌─────────────────────────────────────────────────────────────┐
│              Channel Adapter Layer (pkg/channels/)          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐         │
│  │  wecom  │  │ feishu  │  │dingtalk │  │  ...   │  (可扩展)  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘         │
│       │            │            │            │               │
│       └────────────┴────────────┴────────────┘               │
│                         │                                     │
│              ┌──────────┴──────────┐                          │
│              │  Platform Bridge    │  (统一消息格式)            │
│              │  (pkg/channels/)    │                          │
│              └──────────┬──────────┘                          │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                 Morrigan Core (Existing)                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ handle_convo│  │    LLM      │  │   tools/registry    │  │
│  │   (chat)    │◄─┤   Client    │◄─┤   (MCP tools)      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```


## Technical Considerations

### 核心接口设计

参考 `cc-connect/core/interfaces.go`，定义以下接口：

```go
// pkg/models/channel/channel.go

// Channel 平台适配器必须实现的接口
type Channel interface {
    Name() string                                    // 平台名称: "wecom", "feishu"
    Start(handler MessageHandler) error              // 启动平台连接
    Reply(ctx context.Context, replyCtx any, content string) error  // 回复消息
    Send(ctx context.Context, replyCtx any, content string) error   // 发送消息
    Stop() error                                     // 停止平台连接
}

// MessageHandler 消息处理函数类型
type MessageHandler func(p Channel, msg *Message)

// ReplyContextReconstructor 可选接口：支持从 sessionKey 重建回复上下文
type ReplyContextReconstructor interface {
    ReconstructReplyCtx(sessionKey string) (any, error)
}

// ImageSender 可选接口：支持发送图片
type ImageSender interface {
    SendImage(ctx context.Context, replyCtx any, img ImageAttachment) error
}

// 统一消息结构
type Message struct {
    Channel   string            // 平台名称
    SessionKey string           // 唯一标识: "{platform}:{chatID}:{userID}"
    MessageID  string           // 平台原始消息ID (用于去重)
    UserID     string
    UserName   string
    ChatName   string
    Content    string           // 消息内容
    Images     []ImageAttachment
    Files      []FileAttachment
    Audio      *AudioAttachment
    ReplyCtx   any              // 平台特定回复上下文
    FromVoice  bool             // 语音转文字
}
```

### 平台注册机制

```go
// pkg/channels/registry.go

type Registry struct {
    channels map[string]Factory
}

type PlatformFactory func(opts map[string]any) (Channel, error)

// 全局注册表
var registry *PlatformRegistry

func RegisterPlatform(name string, factory PlatformFactory)
func NewPlatform(name string, opts map[string]any) (Platform, error)
```

每个平台在 `init()` 中注册：

```go
// pkg/channels/wecom/wecom.go
func init() {
    channels.RegisterPlatform("wecom", New)
}
```

### WeCom 适配器设计

**HTTP Webhook 模式：**
- 接收 GET 请求验证回调 URL
- 接收 POST 请求处理加密消息（XML + AES-256-CBC）
- 消息类型：文本、图片、语音
- 回复：POST XML 消息
- Access token 缓存（提前 60 秒刷新）

**WebSocket 长连接模式：**
- 独立进程运行，通过 bridge 与主进程通信
- 支持重连（指数退避：1s -> 30s 最大）
- 心跳保活（30s ping/pong）
- 流式响应支持

### 与 Morrigan Core 的集成

消息流程：

1. WeCom 适配器接收消息
2. 解析并构建统一 `Message` 结构
3. 检查去重、过滤老消息、白名单
4. 构建 sessionKey: `wecom:{chatID}:{userID}`
5. 调用 `MessageHandler` → 转发给现有 `handle_convo.go` 的对话处理逻辑
6. AI 回复通过适配器的 `Reply()` 方法发送回平台

**配置扩展：**

在现有 `settings` 中添加平台配置：

```go
// pkg/settings/settings.go
type PlatformConfig struct {
    Enable  bool
    Type    string  // "wecom", "feishu"
    Config  map[string]any
}
```

### 关键实现细节

1. **去重机制**：使用 Redis 缓存 MsgId，60 秒 TTL
2. **Token 缓存**：Access Token 缓存，提前刷新
3. **消息分片**：大消息按 UTF-8 拆分（WeCom 限制 2000 字符）
4. **异步处理**：`go handler(p, msg)` 非阻塞处理
5. **优雅关闭**：Context 取消和连接 draining

## System-Wide Impact

### Interaction Graph

```
WeCom Message Received
    ↓
[wecom.go: message handler]
    ↓
Parse XML + AES decrypt
    ↓
[bridge.go: dispatch]
    ↓
Check dedup (Redis)
    ↓
Check allowlist
    ↓
[handle_convo.go: postChat]
    ↓
[llm.Client: Chat]
    ↓
AI Response
    ↓
[wecom.go: Reply]
    ↓
POST XML to WeCom API
```

### 错误处理

- 平台 API 调用失败：重试 + 告警
- LLM 调用失败：返回友好错误消息
- 消息处理超时：平台一般有超时限制，考虑异步回复

## Acceptance Criteria

- [ ] `pkg/channels/` 目录创建完成，核心接口定义完成
- [ ] WeCom HTTP Webhook 适配器可接收消息并回复
- [ ] 消息正确路由到现有 `handle_convo.go` 对话处理
- [ ] 集成测试通过（WeCom 模拟消息）
- [ ] 配置可通过 `settings` 管理
- [ ] 文档说明如何添加新平台

## Dependencies & Risks

**依赖：**
- 现有 Redis 存储（去重、Token 缓存）
- 现有 LLM 客户端
- 现有对话处理逻辑

**风险：**
- WeCom API 变更需同步更新
- 消息加密/签名验证需严格实现
- 平台限流需处理

## Implementation Phases

### Phase 1: Core Platform Layer (基础设施)

- 创建 `pkg/channels/` 目录结构
- 实现 `channel.go` 核心接口
- 实现 `registry.go` 平台注册表
- 实现 `message.go` 统一消息结构
- 实现 `dedup.go` 去重机制

### Phase 2: WeCom HTTP Adapter (WeCom HTTP 适配器)

- 实现 `pkg/channels/wecom/wecom.go`
- 实现消息解析（AES 解密）
- 实现回复发送
- 实现 Access Token 管理

### Phase 3: Integration (集成)

- 创建 `pkg/web/api/handle_platform.go`
- 集成到路由系统
- 对接现有 `handle_convo.go`
- 配置管理

### Phase 4: Testing & Polish (测试完善)

- 单元测试
- 集成测试
- 飞书适配器（扩展）
