package settings

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// consts
const (
	Name = "Morrigan"
)

// Config ...
type Config struct {
	Name         string   `ignored:"true"`
	Version      string   `ignored:"true"`
	PgStoreDSN   string   `envconfig:"PG_STORE_DSN" default:"postgres://scaffold@localhost/scaffold?sslmode=disable"`
	PgTSConfig   string   `envconfig:"PG_TS_CONFIG"`
	PgQueryDebug bool     `envconfig:"PG_QUERY_DEBUG"`
	DbAutoInit   bool     `envconfig:"DB_AUTO_INIT"`
	SentryDSN    string   `envconfig:"SENTRY_DSN" `
	HTTPListen   string   `envconfig:"HTTP_LISTEN" default:":5001" required:"true"`
	RedisURI     string   `envconfig:"redis_uri" default:"redis://localhost:6379/1" required:"true"`
	AllowOrigins []string `envconfig:"allow_origins" default:"*" desc:"cors"` // CORS: 允许的 Origin 调用来源
	AuthRequired bool     `envconfig:"Auth_Required"`
	AuthSecret   string   `envconfig:"Auth_Secret" desc:"for chatgpt-web session only"`
	CookieName   string   `envconfig:"Cookie_Name" default:"oaic" desc:"for oauth client"`
	CookiePath   string   `envconfig:"Cookie_Path" default:"/" desc:"for oauth client"`
	CookieDomain string   `envconfig:"Cookie_Domain" desc:"for oauth client"`
	CookieMaxAge int      `envconfig:"Cookie_MaxAge" desc:"for oauth client"`

	OpenAIAPIKey string `envconfig:"openAi_Api_Key" required:"true"`
	PresetFile   string `envconfig:"preset_file" desc:"custom welcome and messages"`
}

var (
	// Current 当前配置
	Current = new(Config)
)

func init() {
	if err := envconfig.Process(Name, Current); err != nil {
		log.Printf("envconfig process fail: %s", err)
	}

	Current.Name = Name
	Current.Version = version
}

// Usage 打印配置帮助
func Usage() error {
	log.Printf("ver: %s", Current.Version)
	return envconfig.Usage(Current.Name, Current)
}

// AllowAllOrigins ...
func AllowAllOrigins() bool {
	return 0 == len(Current.AllowOrigins) ||
		1 == len(Current.AllowOrigins) && Current.AllowOrigins[0] == "*"
}
