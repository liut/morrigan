package stores

import (
	"context"
	"strings"
	"testing"

	"github.com/cupogo/andvari/models/oid"
	auth "github.com/liut/simpauth"

	"github.com/liut/morign/pkg/models/mcps"
	"github.com/liut/morign/pkg/models/skills"
)

func TestSkillVisibleCond(t *testing.T) {
	cases := []struct {
		name    string
		channel string
		user    bool
		wantBit skills.Channel
		wantOwn bool
	}{
		{"web no user", "", false, skills.ChannelWeb, false},
		{"wecom no user", "wecom", false, skills.ChannelWecom, false},
		{"feishu with user", "feishu", true, skills.ChannelFeishu, true},
		{"web with user", "", true, skills.ChannelWeb, true},
	}
	for _, c := range cases {
		ctx := context.Background()
		if c.channel != "" {
			ctx = mcps.ContextWithChannel(ctx, c.channel)
		}
		if c.user {
			ctx = auth.ContextWithUser(ctx, &auth.User{
				OID:  oid.NewID(oid.OtAccount).String(),
				UID:  "u-" + c.name,
				Name: c.name,
			})
		}
		cond, args := skillVisibleCond(ctx)
		if strings.Contains(cond, "channel = 0") {
			t.Errorf("%s: cond should not treat channel=0 as public: %q", c.name, cond)
		}
		if !strings.Contains(cond, "channel & ? != 0") {
			t.Errorf("%s: cond missing channel bit clause: %q", c.name, cond)
		}
		hasOwn := strings.Contains(cond, "owner = ?")
		if hasOwn != c.wantOwn {
			t.Errorf("%s: owner clause = %v, want %v", c.name, hasOwn, c.wantOwn)
		}
		if len(args) != 1+boolToInt(c.wantOwn) {
			t.Errorf("%s: args len = %d", c.name, len(args))
		}
		if got, ok := args[0].(skills.Channel); !ok || got != c.wantBit {
			t.Errorf("%s: channel bit = %v, want %v", c.name, args[0], c.wantBit)
		}
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestSkillVisible(t *testing.T) {
	ownerA := oid.NewID(oid.OtAccount).String()
	ctxA := auth.ContextWithUser(mcps.ContextWithChannel(context.Background(), "wecom"), &auth.User{
		OID: ownerA, UID: "a", Name: "A",
	})
	ctxB := auth.ContextWithUser(mcps.ContextWithChannel(context.Background(), "wecom"), &auth.User{
		OID: oid.NewID(oid.OtAccount).String(), UID: "b", Name: "B",
	})
	ctxAnon := mcps.ContextWithChannel(context.Background(), "wecom")

	cases := []struct {
		name    string
		ctx     context.Context
		channel skills.Channel
		want    bool
	}{
		{"owner visible unpublished", ctxA, skills.ChannelNone, true},
		{"other cannot see unpublished", ctxB, skills.ChannelNone, false},
		{"other sees published", ctxB, skills.ChannelWecom, true},
		{"anonymous cannot see unpublished", ctxAnon, skills.ChannelNone, false},
		{"anonymous sees published", ctxAnon, skills.ChannelWecom, true},
	}
	for _, c := range cases {
		obj := &skills.Skill{SkillBasic: skills.SkillBasic{Channel: c.channel, Owner: oid.Cast(ownerA)}}
		if got := skillVisible(c.ctx, obj); got != c.want {
			t.Errorf("%s: skillVisible = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDBBeforeCreateSkill(t *testing.T) {
	user := &auth.User{OID: oid.NewID(oid.OtAccount).String(), UID: "u", Name: "u"}
	ctx := auth.ContextWithUser(context.Background(), user)

	// owner 为空 + 有用户：补上当前用户
	obj := &skills.Skill{}
	if err := dbBeforeCreateSkill(ctx, nil, obj); err != nil {
		t.Fatalf("dbBeforeCreateSkill: %v", err)
	}
	if obj.Owner != oid.Cast(user.OID) {
		t.Errorf("owner = %v, want %v", obj.Owner, user.OID)
	}

	// owner 为空 + 无用户：保持零值
	obj2 := &skills.Skill{}
	if err := dbBeforeCreateSkill(context.Background(), nil, obj2); err != nil {
		t.Fatalf("dbBeforeCreateSkill no user: %v", err)
	}
	if !obj2.Owner.IsZero() {
		t.Errorf("owner should stay zero without user, got %v", obj2.Owner)
	}

	// owner 已指定 + 有用户：不被覆盖（keeper 代建 / 导入可指定 owner）
	initial := oid.NewID(oid.OtAccount)
	obj3 := &skills.Skill{SkillBasic: skills.SkillBasic{Owner: initial}}
	if err := dbBeforeCreateSkill(ctx, nil, obj3); err != nil {
		t.Fatalf("dbBeforeCreateSkill with owner: %v", err)
	}
	if obj3.Owner != initial {
		t.Errorf("owner should be preserved when set, got %v", obj3.Owner)
	}
}
