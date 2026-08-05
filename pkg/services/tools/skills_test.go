package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/liut/morign/pkg/models/skills"
	"github.com/liut/morign/pkg/services/stores"
)

type fakeSkillStore struct {
	byName map[string]*skills.Skill
	files  map[string]skills.Files
}

func (f *fakeSkillStore) ListVisibleMetadata(ctx context.Context, spec *stores.SkillSpec) (skills.Skills, int, error) {
	var data skills.Skills
	for _, obj := range f.byName {
		data = append(data, *obj)
	}
	if spec.Limit > 0 && len(data) > spec.Limit {
		data = data[:spec.Limit]
	}
	return data, len(data), nil
}

func (f *fakeSkillStore) LoadForName(ctx context.Context, name string) (*skills.Skill, error) {
	obj, ok := f.byName[name]
	if !ok {
		return nil, errors.New("skill not found")
	}
	return obj, nil
}

func (f *fakeSkillStore) ListFileNames(ctx context.Context, name string) (skills.Files, error) {
	if _, ok := f.byName[name]; !ok {
		return nil, errors.New("skill not found")
	}
	return f.files[name], nil
}

func (f *fakeSkillStore) ReadFile(ctx context.Context, name, path string) (*skills.File, error) {
	if _, ok := f.byName[name]; !ok {
		return nil, errors.New("skill not found")
	}
	for i := range f.files[name] {
		if f.files[name][i].Path == path {
			return &f.files[name][i], nil
		}
	}
	return nil, errors.New("file not found")
}

func skillFor(name string) *skills.Skill {
	return &skills.Skill{SkillBasic: skills.SkillBasic{
		Name: name, Description: "desc", Content: "content-" + name,
	}}
}

func fileRow(path, content string, kind skills.FileKind) skills.File {
	return skills.File{FileBasic: skills.FileBasic{
		Path:    path,
		Content: []byte(content),
		Mime:    "text/plain",
		Kind:    kind,
		Size:    int64(len(content)),
	}}
}

func TestCallSkillList(t *testing.T) {
	r := &Registry{skills: &fakeSkillStore{byName: map[string]*skills.Skill{
		"a": skillFor("a"),
		"b": skillFor("b"),
	}}}
	res, err := r.callSkillList(context.Background(), nil)
	if err != nil {
		t.Fatalf("callSkillList: %v", err)
	}
	text := resultText(res)
	if !strings.Contains(text, "- a:") || !strings.Contains(text, "- b:") {
		t.Errorf("skill_list should list names with descriptions: %q", text)
	}
	res, _ = r.callSkillList(context.Background(), map[string]any{"limit": 1})
	if strings.Count(resultText(res), "\n") != 0 {
		t.Errorf("limit 1 should return single line: %q", resultText(res))
	}
	if res, _ := (&Registry{}).callSkillList(context.Background(), nil); !resultError(res) {
		t.Error("nil store should error")
	}
}

func TestCallSkillRead(t *testing.T) {
	r := &Registry{skills: &fakeSkillStore{byName: map[string]*skills.Skill{
		"invoice": skillFor("invoice"),
	}}}
	res, err := r.callSkillRead(context.Background(), map[string]any{"name": "invoice"})
	if err != nil {
		t.Fatalf("callSkillRead: %v", err)
	}
	text := resultText(res)
	if !strings.Contains(text, "content-invoice") {
		t.Errorf("skill_read should return SKILL.md content: %q", text)
	}
	if strings.Contains(text, "scripts/a.py") {
		t.Errorf("skill_read should not include file list: %q", text)
	}
}

func TestCallSkillReadErrors(t *testing.T) {
	r := &Registry{skills: &fakeSkillStore{byName: map[string]*skills.Skill{
		"invoice": skillFor("invoice"),
	}}}
	if res, _ := r.callSkillRead(context.Background(), nil); !resultError(res) {
		t.Error("missing name should error")
	}
	if res, _ := r.callSkillRead(context.Background(), map[string]any{"name": "nope"}); !resultError(res) {
		t.Error("not found should error")
	}
	empty := &Registry{}
	if res, _ := empty.callSkillRead(context.Background(), map[string]any{"name": "x"}); !resultError(res) {
		t.Error("nil store should error")
	}
}

func TestCallSkillFileList(t *testing.T) {
	f := &fakeSkillStore{byName: map[string]*skills.Skill{
		"invoice": skillFor("invoice"),
		"empty":   skillFor("empty"),
	}}
	f.files = map[string]skills.Files{
		"invoice": {
			fileRow("references/x.md", "ref", skills.FileKindText),
			fileRow("scripts/a.py", "print(1)", skills.FileKindText),
		},
	}
	r := &Registry{skills: f}
	res, err := r.callSkillFileList(context.Background(), map[string]any{"name": "invoice"})
	if err != nil {
		t.Fatalf("callSkillFileList: %v", err)
	}
	if text := resultText(res); text != "references/x.md (text/plain, 3 bytes)\nscripts/a.py (text/plain, 8 bytes)" {
		t.Errorf("skill_file_list should return sorted manifest: %q", text)
	}
	if res, _ := r.callSkillFileList(context.Background(), map[string]any{"name": "empty"}); resultText(res) != "(no resource files)" {
		t.Errorf("empty skill should report no files: %q", resultText(res))
	}
	if res, _ := r.callSkillFileList(context.Background(), map[string]any{"name": "nope"}); !resultError(res) {
		t.Error("invisible skill should error")
	}
}

func TestCallSkillFileRead(t *testing.T) {
	f := &fakeSkillStore{byName: map[string]*skills.Skill{
		"invoice": skillFor("invoice"),
	}}
	f.files = map[string]skills.Files{
		"invoice": {
			fileRow("scripts/a.py", "print(1)", skills.FileKindText),
			fileRow("assets/logo.png", "PNG", skills.FileKindBinary),
		},
	}
	r := &Registry{skills: f}
	res, err := r.callSkillFileRead(context.Background(), map[string]any{"name": "invoice", "file": "scripts/a.py"})
	if err != nil {
		t.Fatalf("callSkillFileRead: %v", err)
	}
	if !strings.Contains(resultText(res), "print(1)") {
		t.Errorf("skill_file_read result missing content: %q", resultText(res))
	}
	if res, _ := r.callSkillFileRead(context.Background(), map[string]any{"name": "invoice", "file": "assets/logo.png"}); resultError(res) {
		t.Errorf("binary resource should not error: %q", resultText(res))
	} else if text := resultText(res); !strings.Contains(text, "content not inlined") {
		t.Errorf("binary resource should not inline content: %q", text)
	}
	if res, _ := r.callSkillFileRead(context.Background(), map[string]any{"name": "invoice", "file": "nope.py"}); !resultError(res) {
		t.Error("unknown resource should error")
	}
	if res, _ := r.callSkillFileRead(context.Background(), map[string]any{"name": "invoice"}); !resultError(res) {
		t.Error("missing file should error")
	}
}

func TestCallSkillFileReadInvisibleSkill(t *testing.T) {
	r := &Registry{skills: &fakeSkillStore{byName: map[string]*skills.Skill{
		"invoice": skillFor("invoice"),
	}}}
	if res, _ := r.callSkillFileRead(context.Background(), map[string]any{"name": "nope", "file": "a.py"}); !resultError(res) {
		t.Error("invisible skill should error")
	}
}

func resultText(res map[string]any) string {
	if content, ok := res["content"].([]map[string]any); ok && len(content) > 0 {
		if text, ok := content[0]["text"].(string); ok {
			return text
		}
	}
	return ""
}

func resultError(res map[string]any) bool {
	isErr, _ := res["isError"].(bool)
	return isErr
}
