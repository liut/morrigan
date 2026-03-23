# AGENTS.md - Morign 智能体指南

## 项目概述

Morign 是一个 AI 聊天后端，集知识库问答、MCP 工具与 OAuth 认证于一体。

### 技术栈

- **语言**: Go 1.25
- **Web 框架**: chi/v5
- **数据库**: PostgreSQL (含 pgvector 扩展)
- **缓存/会话**: Redis
- **LLM**: 自定义实现 (pkg/services/llm)，支持 OpenAI/Anthropic/OpenRouter/Ollama

## 编码规范

- 简洁为上
- 假设您所编写代码的维护者和读者都是 Go 专家
- 无需用注释解释显而易见的内容
- 使用自解释的变量和函数名
- 上下文清晰时使用短变量名
- 日志使用 `logger().Infow()` 或 `logger().Warnw()`
- 修改结构体时注意 JSON tag 命名一致


## 常用操作

### 代码检查与测试

```bash
# 运行 lint
make lint

# 运行测试
make test-models    # models 包测试
make test-stores    # stores 包测试（含集成测试）
```

## 注意

- 每次提交之前都要确认并先执行 make vet lint
