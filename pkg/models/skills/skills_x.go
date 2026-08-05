package skills

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var (
	// 开放标准约束：小写字母数字连字符，1-64 字符
	nameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

	ErrInvalidName     = errors.New("invalid skill name")
	ErrDescriptionLong = errors.New("description too long")
	ErrFrontmatterMiss = errors.New("missing skill frontmatter")
	ErrNameMismatch    = errors.New("frontmatter name mismatch")
)

// MaxDescriptionLen 描述长度上限，与 DB 的 varchar(124) 一致（按字符计）
const MaxDescriptionLen = 124

// ValidName 校验开放标准要求的 name 格式
func ValidName(name string) bool {
	return len(name) > 0 && len(name) <= 64 && nameRe.MatchString(name)
}

// ValidDescription 校验描述长度（按字符计，varchar(124) 语义）
func ValidDescription(s string) bool {
	return utf8.RuneCountInString(s) <= MaxDescriptionLen
}

// Matches 判断位掩码是否包含指定频道名（web/wecom/feishu），None（0）表示未投放，
// 不匹配任何频道（可见性由 owner 规则承担）。空频道名按 web 处理（HTTP 请求无频道上下文）。
func (c Channel) Matches(channel string) bool {
	if c == ChannelNone {
		return false
	}
	var bit Channel
	switch channel {
	case "wecom":
		bit = ChannelWecom
	case "feishu":
		bit = ChannelFeishu
	default:
		bit = ChannelWeb
	}
	return c&bit != 0
}

// Frontmatter 解析 SKILL.md 开头的 YAML frontmatter，返回 name 与 description。
// 无 frontmatter 时返回 ok=false。
func Frontmatter(content string) (name, desc string, ok bool) {
	rest, ok := cutFrontmatter(content)
	if !ok {
		return "", "", false
	}
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(rest), &fm); err != nil {
		return "", "", false
	}
	return strings.TrimSpace(fm.Name), strings.TrimSpace(fm.Description), true
}

// ValidateFrontmatter 校验内容中的 frontmatter：存在且 name/description 非空，
// name 与记录名一致且符合格式。
func ValidateFrontmatter(content, name, desc string) error {
	fmName, fmDesc, ok := Frontmatter(content)
	if !ok {
		return ErrFrontmatterMiss
	}
	if fmName == "" || fmDesc == "" {
		return ErrFrontmatterMiss
	}
	if fmName != name {
		return ErrNameMismatch
	}
	return nil
}

func cutFrontmatter(content string) (string, bool) {
	s := strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return "", false
	}
	lines := strings.SplitN(s, "\n", 2)
	if len(lines) < 2 {
		return "", false
	}
	rest := lines[1]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", false
	}
	return rest[:idx], true
}
