---
title: "Extract duplicate executeToolCallLoop into shared ToolExecutor"
category: logic-errors
date: 2026-03-26
tags: [refactor, code-duplication, golang]
related:
  - docs/plans/2026-03-26-002-refactor-handle-duplicate-logic-plan.md
---

# Extract duplicate executeToolCallLoop into shared ToolExecutor

## Problem Description

Two separate files (`handle_convo.go` and `handle_platform.go`) implemented nearly identical `executeToolCallLoop` logic with inconsistent logging packages (`slog` vs `logger().Infow`). This duplication risked divergence over time and inconsistent behavior between API and platform handler code paths.

## Root Cause

The `executeToolCallLoop` method was copy-pasted into both handlers with different logging implementations:
- `handle_convo.go`: used `logger().Infow` (project-standard custom logger)
- `handle_platform.go`: used `log/slog` package directly

When a tool call failed in `handle_platform.go`, the error was logged via `slog.Warn` but success was not logged at all. Meanwhile `handle_convo.go` logged both failures and successes with full context (`toolCallID`, `toolCallType`, `toolCallName`).

## Solution

**Created `pkg/web/api/tool_executor.go`** - unified executor:

```go
// chatExecutor defines the chat execution function type (streaming or non-streaming)
type chatExecutor func(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition) (string, []llm.ToolCall, *llm.Usage, error)

// ToolExecutor encapsulates tool call loop logic
type ToolExecutor struct {
    toolreg *tools.Registry
}

func NewToolExecutor(toolreg *tools.Registry) *ToolExecutor {
    return &ToolExecutor{toolreg: toolreg}
}

func (e *ToolExecutor) ExecuteToolCallLoop(
    ctx context.Context,
    messages []llm.Message,
    tools []llm.ToolDefinition,
    exec chatExecutor,
) (string, []llm.ToolCall, *llm.Usage, error) {
    for {
        answer, toolCalls, usage, err := exec(ctx, messages, tools)
        if err != nil {
            return "", nil, nil, err
        }
        if len(toolCalls) == 0 {
            return answer, nil, usage, nil
        }
        messages = append(messages, llm.Message{
            Role:      llm.RoleAssistant,
            ToolCalls: toolCalls,
        })
        for _, tc := range toolCalls {
            logger().Infow("chat", "toolCallID", tc.ID, "toolCallType", tc.Type, "toolCallName", tc.Function.Name)
            // ... tool execution with unified logging
        }
    }
}
```

**Modified `api.go`** - added `toolExec` field and initialization:

```go
type api struct {
    // ...
    toolExec *ToolExecutor
}

// In newapi():
return &api{
    // ...
    toolExec: NewToolExecutor(toolreg),
}
```

**Simplified `handle_convo.go`** - delegation pattern:

```go
func (a *api) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
    return a.toolExec.ExecuteToolCallLoop(ctx, messages, tools, exec)
}
```

**Simplified `handle_platform.go`** - same delegation pattern:

```go
type channelHandler struct {
    // ...
    toolExec *ToolExecutor
}

// In InitChannels():
phandler = &channelHandler{
    // ...
    toolExec: NewToolExecutor(toolreg),
}

// Simplified wrapper:
func (phandler *channelHandler) executeToolCallLoop(ctx context.Context, messages []llm.Message, tools []llm.ToolDefinition, exec chatExecutor) (string, []llm.ToolCall, *llm.Usage, error) {
    return phandler.toolExec.ExecuteToolCallLoop(ctx, messages, tools, exec)
}
```

## Verification

- `make vet lint` passed (0 issues)
- `go build ./...` succeeded

## Files Changed

| File | Change |
|------|--------|
| `pkg/web/api/tool_executor.go` | NEW - shared ToolExecutor |
| `pkg/web/api/handle_convo.go` | Removed duplicate, added delegation |
| `pkg/web/api/handle_platform.go` | Removed duplicate, added delegation |
| `pkg/web/api/api.go` | Added toolExec field |

## Prevention Strategies

1. **Extract common patterns early** - when two implementations diverge slightly, immediately extract to a shared location
2. **Unified logging interface** - use `logger().Infow` across all packages rather than mixing `slog` and custom loggers
3. **Code review for duplication** - require reviewers to explicitly verify the PR does not duplicate existing logic elsewhere
4. **Consider facade/delegation pattern** when handlers need different contexts but similar core logic

## Recommended Tests

Add unit tests for `ToolExecutor`:

```go
func TestExecuteToolCallLoop_NoToolCalls(t *testing.T) {
    // Verify: when LLM returns no tool calls, returns immediately
}

func TestExecuteToolCallLoop_SingleToolCall(t *testing.T) {
    // Verify: single tool call executes and adds result to messages
}

func TestExecuteToolCallLoop_ContinueOnFailure(t *testing.T) {
    // Verify: if one tool fails, remaining tools still execute
}
```
