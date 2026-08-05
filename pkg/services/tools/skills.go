package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/liut/morign/pkg/models/mcps"
	"github.com/liut/morign/pkg/models/skills"
	"github.com/liut/morign/pkg/services/stores"
)

const (
	ToolNameSkillList     = "skill_list"
	ToolNameSkillRead     = "skill_read"
	ToolNameSkillFileList = "skill_file_list"
	ToolNameSkillFileRead = "skill_file_read"
)

// SkillToolStore 技能工具所需的最小存储接口
type SkillToolStore interface {
	ListVisibleMetadata(ctx context.Context, spec *stores.SkillSpec) (skills.Skills, int, error)
	LoadForName(ctx context.Context, name string) (*skills.Skill, error)
	ListFileNames(ctx context.Context, name string) (skills.Files, error)
	ReadFile(ctx context.Context, name, path string) (*skills.File, error)
}

var (
	skillListDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameSkillList,
		Description: "List available skills (name and description) for discovery. Call when no listed skill matches the task.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "number",
					"description": "max items to return, default 10, max 50",
					"default":     10,
					"minimum":     1,
					"maximum":     50,
				},
			},
		},
	}

	skillReadDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameSkillRead,
		Description: "Load a skill's SKILL.md instructions by name. Call when an available skill is relevant to the current task.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "skill name",
				},
			},
			"required": []string{"name"},
		},
	}

	skillFileListDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameSkillFileList,
		Description: "List a skill's resource file names (scripts, references, assets) by skill name.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "skill name",
				},
			},
			"required": []string{"name"},
		},
	}

	skillFileReadDescriptor = mcps.ToolDescriptor{
		Name:        ToolNameSkillFileRead,
		Description: "Read a skill resource file content by skill name and file path.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "skill name",
				},
				"file": map[string]any{
					"type":        "string",
					"description": "resource file path within the skill",
				},
			},
			"required": []string{"name", "file"},
		},
	}
)

// callSkillList 返回可见技能的紧凑元数据列表，默认取前 10 条。
func (r *Registry) callSkillList(ctx context.Context, args map[string]any) (map[string]any, error) {
	if r.skills == nil {
		return mcps.BuildToolErrorResult("skill store unavailable"), nil
	}
	limit, _, _ := mcps.IntArg(args, "limit")
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	spec := &stores.SkillSpec{}
	spec.Limit = limit
	data, _, err := r.skills.ListVisibleMetadata(ctx, spec)
	if err != nil {
		return mcps.BuildToolErrorResult("list skills failed"), nil
	}
	var sb strings.Builder
	for _, obj := range data {
		sb.WriteString("- ")
		sb.WriteString(obj.Name)
		sb.WriteString(": ")
		sb.WriteString(obj.Description)
		sb.WriteString("\n")
	}
	return mcps.BuildToolSuccessResult(strings.TrimSuffix(sb.String(), "\n")), nil
}

// callSkillRead 返回指定技能 SKILL.md 指令全文，可见性由 LoadForName 统一校验。
func (r *Registry) callSkillRead(ctx context.Context, args map[string]any) (map[string]any, error) {
	if r.skills == nil {
		return mcps.BuildToolErrorResult("skill store unavailable"), nil
	}
	name := mcps.StringArg(args, "name")
	if name == "" {
		return mcps.BuildToolErrorResult("missing required argument: name"), nil
	}
	obj, err := r.skills.LoadForName(ctx, name)
	if err != nil {
		return mcps.BuildToolErrorResult("skill not found or not visible"), nil
	}
	return mcps.BuildToolSuccessResult(fmt.Sprintf("# Skill: %s\n%s", obj.Name, obj.Content)), nil
}

// callSkillFileList 返回指定技能的资源文件清单（path/mime/size，不含内容）。
func (r *Registry) callSkillFileList(ctx context.Context, args map[string]any) (map[string]any, error) {
	if r.skills == nil {
		return mcps.BuildToolErrorResult("skill store unavailable"), nil
	}
	name := mcps.StringArg(args, "name")
	if name == "" {
		return mcps.BuildToolErrorResult("missing required argument: name"), nil
	}
	data, err := r.skills.ListFileNames(ctx, name)
	if err != nil {
		return mcps.BuildToolErrorResult("skill not found or not visible"), nil
	}
	if len(data) == 0 {
		return mcps.BuildToolSuccessResult("(no resource files)"), nil
	}
	sort.Slice(data, func(i, j int) bool { return data[i].Path < data[j].Path })
	var sb strings.Builder
	for _, f := range data {
		fmt.Fprintf(&sb, "%s (%s, %d bytes)\n", f.Path, f.Mime, f.Size)
	}
	return mcps.BuildToolSuccessResult(strings.TrimSuffix(sb.String(), "\n")), nil
}

// callSkillFileRead 返回指定资源文件内容；binary 文件仅返回元数据，不内联内容。
func (r *Registry) callSkillFileRead(ctx context.Context, args map[string]any) (map[string]any, error) {
	if r.skills == nil {
		return mcps.BuildToolErrorResult("skill store unavailable"), nil
	}
	name := mcps.StringArg(args, "name")
	file := mcps.StringArg(args, "file")
	if name == "" || file == "" {
		return mcps.BuildToolErrorResult("missing required argument: name or file"), nil
	}
	obj, err := r.skills.ReadFile(ctx, name, file)
	if err != nil {
		if errors.Is(err, stores.ErrFileNotFound) {
			return mcps.BuildToolErrorResult("resource not found: " + file), nil
		}
		return mcps.BuildToolErrorResult("skill not found or not visible"), nil
	}
	if obj.Kind == skills.FileKindBinary {
		return mcps.BuildToolSuccessResult(fmt.Sprintf("%s: binary resource (%s, %d bytes), content not inlined", file, obj.Mime, obj.Size)), nil
	}
	return mcps.BuildToolSuccessResult(string(obj.Content)), nil
}
