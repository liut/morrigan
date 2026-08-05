package skills

import (
	"strings"
	"testing"
)

func TestValidName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"invoice", true},
		{"pdf-processing", true},
		{"a1-b2", true},
		{"", false},
		{"PDF", false},
		{"-pdf", false},
		{"pdf-", false},
		{"pdf--x", false},
		{"pdf processing", false},
		{"pdf_processing", false},
		{string(make([]byte, 65)), false},
	}
	for _, c := range cases {
		if got := ValidName(c.name); got != c.want {
			t.Errorf("ValidName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestValidDescription(t *testing.T) {
	ok := strings.Repeat("a", MaxDescriptionLen)
	if !ValidDescription(ok) {
		t.Errorf("length %d should be valid", MaxDescriptionLen)
	}
	if ValidDescription(ok + "a") {
		t.Error("length over limit should be invalid")
	}
	// 中文字符按字符计，不按字节
	cn := strings.Repeat("技", MaxDescriptionLen)
	if !ValidDescription(cn) {
		t.Error("multi-byte chars should count by rune")
	}
	if ValidDescription(cn + "技") {
		t.Error("multi-byte over limit should be invalid")
	}
}

func TestChannelMatches(t *testing.T) {
	if ChannelNone.Matches("wecom") {
		t.Error("ChannelNone should not match any channel")
	}
	if ChannelNone.Matches("") {
		t.Error("ChannelNone should not match empty channel")
	}
	if !ChannelWecom.Matches("wecom") {
		t.Error("ChannelWecom should match wecom")
	}
	if ChannelWecom.Matches("feishu") {
		t.Error("ChannelWecom should not match feishu")
	}
	if ChannelWecom.Matches("") {
		t.Error("ChannelWecom should not match web")
	}
	if !ChannelWeb.Matches("") {
		t.Error("empty channel should map to web")
	}
	if (ChannelWeb | ChannelFeishu).Matches("feishu") == false {
		t.Error("combined mask should match feishu")
	}
	if (ChannelWeb | ChannelFeishu).Matches("wecom") {
		t.Error("combined mask should not match wecom")
	}
}

func TestFrontmatter(t *testing.T) {
	content := "---\nname: invoice\ndescription: 处理发票\n---\n\n正文"
	name, desc, ok := Frontmatter(content)
	if !ok || name != "invoice" || desc != "处理发票" {
		t.Fatalf("Frontmatter = (%q, %q, %v)", name, desc, ok)
	}
	if _, _, ok := Frontmatter("plain markdown"); ok {
		t.Error("plain markdown should not have frontmatter")
	}
}

func TestValidateFrontmatter(t *testing.T) {
	good := "---\nname: invoice\ndescription: 处理发票\n---\n\n正文"
	if err := ValidateFrontmatter(good, "invoice", "处理发票"); err != nil {
		t.Errorf("ValidateFrontmatter(good) = %v", err)
	}
	if err := ValidateFrontmatter("plain", "invoice", "d"); err != ErrFrontmatterMiss {
		t.Errorf("missing frontmatter err = %v", err)
	}
	if err := ValidateFrontmatter("---\nname: other\ndescription: d\n---", "invoice", "d"); err != ErrNameMismatch {
		t.Errorf("mismatch err = %v", err)
	}
}
