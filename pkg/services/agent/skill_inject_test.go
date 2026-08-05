package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/liut/morign/pkg/models/skills"
	"github.com/liut/morign/pkg/settings"
)

type fakeSkillStore struct {
	byName map[string]*skills.Skill
	recent []skills.Skill
}

func (f *fakeSkillStore) TopRecent(ctx context.Context, limit int) (skills.Skills, error) {
	if limit <= 0 || limit > len(f.recent) {
		limit = len(f.recent)
	}
	return f.recent[:limit], nil
}

func (f *fakeSkillStore) LoadForName(ctx context.Context, name string) (*skills.Skill, error) {
	obj, ok := f.byName[name]
	if !ok {
		return nil, errors.New("skill not found")
	}
	return obj, nil
}

func newFakeStore() *fakeSkillStore {
	return &fakeSkillStore{
		byName: map[string]*skills.Skill{},
	}
}

func TestBuildSkillPromptDirectInjection(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	f := newFakeStore()
	f.byName["a"] = skillOf("a", "A", "content-a")
	f.byName["b"] = skillOf("b", "B", "content-b")

	out, err := BuildSkillPrompt(context.Background(), f, []string{"b", "a", "b", ""})
	if err != nil {
		t.Fatalf("BuildSkillPrompt: %v", err)
	}
	if !strings.Contains(out, "content-a") || !strings.Contains(out, "content-b") {
		t.Errorf("direct injection missing full content: %q", out)
	}
	if strings.Contains(out, "skill_read") {
		t.Errorf("direct injection should not contain tool hint: %q", out)
	}
}

func TestBuildSkillPromptMetadata(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	f := newFakeStore()
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		f.byName[name] = skillOf(name, "desc-"+name, "content-"+name)
	}

	out, err := BuildSkillPrompt(context.Background(), f, []string{"a", "b", "c", "d", "e"})
	if err != nil {
		t.Fatalf("BuildSkillPrompt: %v", err)
	}
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		if !strings.Contains(out, "desc-"+name) {
			t.Errorf("metadata missing %s: %q", name, out)
		}
		if strings.Contains(out, "content-"+name) {
			t.Errorf("metadata leaked content for %s: %q", name, out)
		}
	}
	if !strings.Contains(out, "skill_read") {
		t.Errorf("metadata should hint skill_read: %q", out)
	}
}

func TestBuildSkillPromptDefaults(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	settings.Current.SkillDefaultCount = 2
	f := newFakeStore()
	f.byName["n1"] = skillOf("n1", "D1", "c1")
	f.byName["n2"] = skillOf("n2", "D2", "c2")
	f.recent = []skills.Skill{
		{SkillBasic: skills.SkillBasic{Name: "n1"}},
		{SkillBasic: skills.SkillBasic{Name: "n2"}},
		{SkillBasic: skills.SkillBasic{Name: "n3"}},
	}

	out, err := BuildSkillPrompt(context.Background(), f, nil)
	if err != nil {
		t.Fatalf("BuildSkillPrompt: %v", err)
	}
	if !strings.Contains(out, "c1") || !strings.Contains(out, "c2") {
		t.Errorf("default top-2 should inject full content: %q", out)
	}
}

func TestBuildSkillPromptSkipsInvisible(t *testing.T) {
	settings.Current.SkillDirectThreshold = 3
	f := newFakeStore()
	f.byName["a"] = skillOf("a", "A", "content-a")

	out, err := BuildSkillPrompt(context.Background(), f, []string{"a", "missing"})
	if err != nil {
		t.Fatalf("BuildSkillPrompt: %v", err)
	}
	if !strings.Contains(out, "content-a") {
		t.Errorf("visible skill should be injected: %q", out)
	}
}

func TestBuildSkillPromptEmpty(t *testing.T) {
	out, err := BuildSkillPrompt(context.Background(), newFakeStore(), nil)
	if err != nil {
		t.Fatalf("BuildSkillPrompt: %v", err)
	}
	if out != "" {
		t.Errorf("empty store should return empty prompt, got %q", out)
	}
}

func TestSkillPromptByCommand(t *testing.T) {
	f := newFakeStore()
	f.byName["invoice"] = skillOf("invoice", "D", "invoice-content")
	out, err := SkillPromptByCommand(context.Background(), f, "invoice")
	if err != nil {
		t.Fatalf("SkillPromptByCommand: %v", err)
	}
	if !strings.Contains(out, "invoice-content") {
		t.Errorf("command inject missing content: %q", out)
	}
	if _, err := SkillPromptByCommand(context.Background(), f, "nope"); err == nil {
		t.Error("missing skill should return error")
	}
}

func skillOf(name, desc, content string) *skills.Skill {
	return &skills.Skill{SkillBasic: skills.SkillBasic{Name: name, Description: desc, Content: content}}
}
