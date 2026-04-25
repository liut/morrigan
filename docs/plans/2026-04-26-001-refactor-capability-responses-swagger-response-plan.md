---
title: "refactor: Capability Responses 使用 SwaggerResponse 结构"
type: refactor
status: completed
date: 2026-04-26
---

# Capability Responses 使用 SwaggerResponse 结构

## Overview

将 `Capability.Responses` 从 `string`（JSON blob）改为结构化的 `map[string]SwaggerResponse`，与 `SwaggerParam` 保持同一风格。

## Problem Statement

当前 `Responses` 存的是 JSON string，需要自行解析。`Parameters` 已使用 `[]SwaggerParam` 结构化，Responses 也应使用相同模式。

## Proposed Solution

### 1. 新增 SwaggerResponse 结构

在 `docs/capability.yaml` 中添加：

```yaml
  - name: SwaggerResponse
    comment: swagger 响应定义
    fields:
      - comment: 响应描述
        name: Description
        type: string
        tags: {json: 'description', yaml: 'description'}
      - comment: schema 定义
        name: Schema
        type: any
        tags: {json: 'schema,omitempty', yaml: 'schema'}
```

### 2. 修改 Capability.Responses 字段

将 `Responses string` 改为 `map[string]SwaggerResponse`：

```yaml
      - comment: 响应结构 map[code]SwaggerResponse
        name: Responses
        type: 'map[string]SwaggerResponse'
        tags: {bson: 'responses', json: 'responses,omitempty', pg: ",notnull,type:jsonb,default:'{}'"}
```

### 3. 更新 ImportCapabilities

在 `pkg/services/stores/capability_x.go` 的 `ImportCapabilities` 中，将：

```go
if responses, err := json.Marshal(api.Responses); err == nil {
    basic.Responses = string(responses)
}
```

改为直接赋值（因为 yaml 解析时 Responses 已转为 `map[string]SwaggerResponse`）。

### 4. 更新 InvokerForMatch

`InvokerForMatch` 返回 capability 参数时，`parameters` 和 `responses` 会以结构化形式返回，LLM 调用 tool 时能直接看到参数/响应的结构化描述。

## Technical Considerations

### 数据库兼容性

Responses 字段在 PostgreSQL 中存储为 `jsonb`，类型从 `string` 改为 `map[string]SwaggerResponse` 后：
- JSON string → JSON object，存储格式兼容
- 查询时直接获取结构化数据，无需 json.Unmarshal

### 向量匹配

`GetSubject()` 目前基于 Summary + Description 生成，Responses 结构化后不影响向量匹配逻辑。

## Acceptance Criteria

- [ ] `make codegen MDs=docs/capability.yaml` 生成代码后，`Responses` 类型为 `map[string]SwaggerResponse`
- [ ] 导入 swagger 时，Responses 正确解析为 `map[string]SwaggerResponse` 结构
- [ ] `capability_match` tool 返回结果中 responses 包含结构化的 description 和 schema

## Context

参考 `SwaggerParam` 结构（`docs/capability.yaml:94-125`）的风格和实现方式。

## Sources

- `docs/capability.yaml` - Capability 模型定义
- `pkg/models/capability/capability_gen.go:248-263` - SwaggerParam 结构定义
- `pkg/services/stores/capability_x.go:250-347` - ImportCapabilities 实现
