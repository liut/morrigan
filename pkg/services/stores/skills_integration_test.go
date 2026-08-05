//go:build integration

package stores

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/cupogo/andvari/models/oid"
	auth "github.com/liut/simpauth"

	"github.com/liut/morign/pkg/models/mcps"
	"github.com/liut/morign/pkg/models/skills"
)

func TestIntegration_SkillVisibility(t *testing.T) {
	sto := Sgt()
	pid := os.Getpid()
	userA := auth.User{OID: oid.NewID(oid.OtAccount).String(), UID: "skill-a", Name: "Skill A"}
	userB := auth.User{OID: oid.NewID(oid.OtAccount).String(), UID: "skill-b", Name: "Skill B"}

	names := []string{
		fmt.Sprintf("a-none-%d", pid),
		fmt.Sprintf("a-wecom-%d", pid),
		fmt.Sprintf("b-web-%d", pid),
	}
	skillsList := []skills.SkillBasic{
		{Name: names[0], Description: "A private", Content: "---\nname: " + names[0] + "\ndescription: A private\n---\nbody",
			Channel: skills.ChannelNone, Owner: oid.Cast(userA.OID)},
		{Name: names[1], Description: "A wecom", Content: "---\nname: " + names[1] + "\ndescription: A wecom\n---\nbody",
			Channel: skills.ChannelWecom, Owner: oid.Cast(userA.OID)},
		{Name: names[2], Description: "B web", Content: "---\nname: " + names[2] + "\ndescription: B web\n---\nbody",
			Channel: skills.ChannelWeb, Owner: oid.Cast(userB.OID)},
	}
	for _, in := range skillsList {
		if _, err := sto.Skill().CreateSkill(context.Background(), in); err != nil {
			t.Fatalf("CreateSkill(%s) failed: %v", in.Name, err)
		}
	}
	t.Cleanup(func() {
		for _, in := range skillsList {
			if obj, err := sto.Skill().GetSkill(context.Background(), in.Name); err == nil && obj != nil {
				_ = sto.Skill().DeleteSkill(context.Background(), obj.ID.String())
			}
		}
	})

	ctxA := auth.ContextWithUser(mcps.ContextWithChannel(context.Background(), "wecom"), &userA)
	ctxB := auth.ContextWithUser(mcps.ContextWithChannel(context.Background(), "wecom"), &userB)

	// A 在 wecom 可见：自己的未投放 + 自己的 wecom 投放；看不到 B 的 web 投放
	listA, _, err := sto.Skill().ListVisibleMetadata(ctxA, &SkillSpec{})
	if err != nil {
		t.Fatalf("ListVisibleMetadata A failed: %v", err)
	}
	if got := countSkills(listA, names); got != 2 {
		t.Errorf("A visible count = %d, want 2 (%v)", got, names)
	}
	// 列表不含 content
	for _, sk := range listA {
		if sk.Content != "" {
			t.Errorf("metadata list leaked content for %s", sk.Name)
		}
	}

	// B 在 wecom 可见：A 投放的 wecom + 自己的 web；看不到 A 的未投放（Covers AE4）
	listB, _, err := sto.Skill().ListVisibleMetadata(ctxB, &SkillSpec{})
	if err != nil {
		t.Fatalf("ListVisibleMetadata B failed: %v", err)
	}
	if got := countSkills(listB, names); got != 2 {
		t.Errorf("B visible count = %d, want 2", got)
	}
	for _, sk := range listB {
		if sk.Name == names[0] {
			t.Errorf("B should not see A's unpublished skill %s", names[0])
		}
	}

	// LoadForName 越权按 not found（不泄露存在性）
	if _, err := sto.Skill().LoadForName(ctxB, names[0]); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("B LoadForName(A private) err = %v, want ErrSkillNotFound", err)
	}
	// 投放的 wecom 对频道内其他用户可见
	if _, err := sto.Skill().LoadForName(ctxB, names[1]); err != nil {
		t.Errorf("B LoadForName(A wecom) err = %v, want nil", err)
	}
	// 自己的可见，且含全文
	got, err := sto.Skill().LoadForName(ctxA, names[1])
	if err != nil {
		t.Fatalf("A LoadForName failed: %v", err)
	}
	if got.Content == "" {
		t.Error("LoadForName should return full content")
	}

	// ListVisibleMetadata 强制可见性：即便 spec.VisibleOnly 传 false 也不会绕过
	specB := &SkillSpec{}
	listB2, _, err := sto.Skill().ListVisibleMetadata(ctxB, specB)
	if err != nil {
		t.Fatalf("ListVisibleMetadata B2 failed: %v", err)
	}
	if got := countSkills(listB2, names); got != 2 {
		t.Errorf("B2 visible count = %d, want 2", got)
	}

	// 管理端 ListSkill 不设 VisibleOnly：不套可见性，可见全部（含未投放与跨频道）
	listAll, _, err := sto.Skill().ListSkill(ctxB, &SkillSpec{})
	if err != nil {
		t.Fatalf("ListSkill failed: %v", err)
	}
	if got := countSkills(listAll, names); got != 3 {
		t.Errorf("admin list count = %d, want 3", got)
	}
}

func TestIntegration_SkillTopRecent(t *testing.T) {
	sto := Sgt()
	pid := os.Getpid()
	owner := oid.NewID(oid.OtAccount).String()
	user := auth.User{OID: owner, UID: "skill-top", Name: "Skill Top"}
	ctx := auth.ContextWithUser(context.Background(), &user)

	var created []string
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("top-%d-%d", i, pid)
		created = append(created, name)
		if _, err := sto.Skill().CreateSkill(ctx, skills.SkillBasic{
			Name: name, Description: name, Content: "body",
			Channel: skills.ChannelNone, Owner: oid.Cast(owner),
		}); err != nil {
			t.Fatalf("CreateSkill(%s) failed: %v", name, err)
		}
	}
	t.Cleanup(func() {
		for _, name := range created {
			if obj, err := sto.Skill().GetSkill(context.Background(), name); err == nil && obj != nil {
				_ = sto.Skill().DeleteSkill(context.Background(), obj.ID.String())
			}
		}
	})

	data, err := sto.Skill().TopRecent(ctx, 3)
	if err != nil {
		t.Fatalf("TopRecent failed: %v", err)
	}
	if len(data) != 3 {
		t.Errorf("TopRecent len = %d, want 3", len(data))
	}
	if data[0].Name != created[4] {
		t.Errorf("TopRecent[0] = %s, want newest %s", data[0].Name, created[4])
	}
}

func countSkills(data skills.Skills, names []string) int {
	n := 0
	for _, sk := range data {
		for _, name := range names {
			if sk.Name == name {
				n++
			}
		}
	}
	return n
}
