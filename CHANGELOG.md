# Changelog

## v0.3.0 (2026-04-02)

### 新功能

- **平台集成**: 新增 WeCom/Feishu platform adapter layer
- **平台集成**: 新增 ThirdUser 表，优化 WeCom 集成
- **用户模型**: 新增 email等 字段，增强 OAuth 同步，支持 avatar
- **Session**: 新增 session command 系统
- **Session**: 从 sessionKey 提取 channel/chatID 到 Session 结构
- **Storage**: 支持 preset 存储，记忆加载改为仅对已认证用户生效
- **飞书**: WebSocket 客户端集成 slog 结构化日志

### Bug 修复

- 修复图片 URI 相对路径补全
- 修复 signin 前用户数据刷新
- 修复 API 文档注解和 swagger 生成
- 修复 `storeUserAndMeta` 并重命名为 `storeUserWith`

### 重构

- 移除 OAuth MCP 集成，简化用户管理
- 项目更名：morrigan → morign
- 修复测试向量维度
- Makefile 环境变量加载重构

### 文档

- API 文档 `/api/welcome` → `/api/session`
- 新增登录/登出响应示例

---

## v0.2.4 (2026-03-27)

- 新增 OAuth MCP server 注册（改为无需授权，延至请求）

## v0.2.3 (2026-03-26)

- 重构：移除 openai.go 中的 regexp 依赖

## v0.2.2 (2026-03-25)

- 重构：提取文本截断逻辑为可复用工具

## v0.2.1 (2026-03-24)

- 新增 AddHistory 重复检测

## v0.2.0 (2026-03-23)

- 新增 Anthropic provider 和 CLI agent 命令
- 新增统一 LLM service，修复流式 tool calls
- 新增 OAuth SP 作为 MCP 支持
- 新增结构化日志
- 简化 chat API，移除 Full response 格式
- API 启动时加载 preset 和初始化工具注册表

## v0.1.2 (2026-03-xx)

- 初始版本
- 基础 API 功能
- Redis 会话历史
- LLM provider 支持
