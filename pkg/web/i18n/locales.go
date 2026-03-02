package i18n

import (
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Printer ...
type Printer = message.Printer

// nolint
var (
	enUS   = language.AmericanEnglish
	zhHans = language.MustParse("zh-Hans")
	zhHant = language.MustParse("zh-Hant")

	matcher language.Matcher
)

func init() {
	matcher = language.NewMatcher(message.DefaultCatalog.Languages())
}

// GetTag ...
func GetTag(r *http.Request) language.Tag {
	var lang string
	if s := r.FormValue("lang"); s != "" {
		lang = s
	} else if c, err := r.Cookie("lang"); err == nil {
		lang = c.String()
	}
	accept := r.Header.Get("Accept-Language")
	tag, _ := language.MatchStrings(matcher, lang, accept, "zh-hans")
	// tag := message.MatchLanguage(lang, accept, "zh-Hans")
	return tag
}

// GetPrinter ...
func GetPrinter(r *http.Request) *message.Printer {
	return message.NewPrinter(GetTag(r))
}

// Field field validate failed
type Field string

// Field name
func (f Field) Field() string {
	return string(f)
}

func (f Field) GetMessage(r *http.Request) string {
	return GetPrinter(r).Sprintf("Error:Field validation for '%s' failed ", f)
}

// FieldError ...
type FieldError interface {
	Field() string
}

func GetDecodeValueError(r *http.Request, value, codec string) string {
	return GetPrinter(r).Sprintf("Error: can not decode '%s' with %s", value, codec)
}

type StringError string

func (se StringError) GetMessage(r *http.Request) string {
	return GetPrinter(r).Sprint(se)
}
