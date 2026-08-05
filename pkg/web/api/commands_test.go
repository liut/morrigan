package api

import (
	"context"
	"testing"

	"github.com/liut/morign/pkg/models/channel"
)

func TestDetectSkillCommand(t *testing.T) {
	msg := &channel.Message{Content: "/skill invoice 帮我开发票"}
	if cmd := DetectCommand(msg.Content); cmd.Name != "skill" {
		t.Fatalf("DetectCommand = %q, want skill", cmd.Name)
	}
}

func TestHandleSkillCommand(t *testing.T) {
	msg := &channel.Message{Content: "/skill invoice 帮我开发票"}
	handled, err := handleSkillCommand(context.Background(), msg)
	if err != nil {
		t.Fatalf("handleSkillCommand: %v", err)
	}
	if handled {
		t.Error("skill command should continue, not stop")
	}
	if msg.SkillName != "invoice" {
		t.Errorf("SkillName = %q, want invoice", msg.SkillName)
	}
	if msg.Content != "帮我开发票" {
		t.Errorf("Content = %q, want 帮我开发票", msg.Content)
	}
}

func TestHandleSkillCommandInvalid(t *testing.T) {
	msg := &channel.Message{Content: "/skill PDF"}
	if _, err := handleSkillCommand(context.Background(), msg); err != nil {
		t.Fatalf("handleSkillCommand: %v", err)
	}
	if msg.SkillName != "" {
		t.Errorf("invalid name should not set SkillName, got %q", msg.SkillName)
	}
	if msg.Content != "/skill PDF" {
		t.Errorf("invalid command should keep content, got %q", msg.Content)
	}
}

func TestHandleSkillCommandBare(t *testing.T) {
	msg := &channel.Message{Content: "/skill"}
	if _, err := handleSkillCommand(context.Background(), msg); err != nil {
		t.Fatalf("handleSkillCommand: %v", err)
	}
	if msg.SkillName != "" || msg.Content != "/skill" {
		t.Errorf("bare command should be untouched, got %q / %q", msg.SkillName, msg.Content)
	}
}

func TestHandleSkillCommandPrefixCollision(t *testing.T) {
	for _, content := range []string{"/skillset 怎么办", "/skillful 做法", "/skillinvoice"} {
		msg := &channel.Message{Content: content}
		if _, err := handleSkillCommand(context.Background(), msg); err != nil {
			t.Fatalf("handleSkillCommand(%q): %v", content, err)
		}
		if msg.SkillName != "" || msg.Content != content {
			t.Errorf("prefix collision should be untouched, got %q / %q", msg.SkillName, msg.Content)
		}
	}
}

func TestDetectCommandCompleteMatch(t *testing.T) {
	cases := []struct {
		content string
		want    string
	}{
		{"/skill invoice", "skill"},
		{"/skill", "skill"},
		{"  /skill invoice  ", "skill"},
		{"/skill\tinvoice", "skill"},
		{"/reset", "reset"},
		{"\r\n/reset\r\n", "reset"},
		{"/skillset 怎么办", ""},
		{"/skillful 做法", ""},
		{"/resetc", ""},
		{"/clearx", ""},
		{"普通消息 /skill invoice", ""},
	}
	for _, c := range cases {
		if got := DetectCommand(c.content).Name; got != c.want {
			t.Errorf("DetectCommand(%q) = %q, want %q", c.content, got, c.want)
		}
	}
}

func TestHandleSkillCommandWhitespace(t *testing.T) {
	msg := &channel.Message{Content: "  /skill invoice  帮我开发票  \r\n"}
	if _, err := handleSkillCommand(context.Background(), msg); err != nil {
		t.Fatalf("handleSkillCommand: %v", err)
	}
	if msg.SkillName != "invoice" {
		t.Errorf("SkillName = %q, want invoice", msg.SkillName)
	}
	if msg.Content != "帮我开发票" {
		t.Errorf("Content = %q, want 帮我开发票", msg.Content)
	}
}
