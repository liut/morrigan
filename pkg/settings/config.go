package settings

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// consts
const (
	Name = "Morign"
)

// Config ...
type Config struct {
	Name         string   `ignored:"true"`
	Version      string   `ignored:"true"`
	PgStoreDSN   string   `envconfig:"PG_STORE_DSN" default:"postgres://morrigan@localhost/morrigan?sslmode=disable"`
	PgTSConfig   string   `envconfig:"PG_TS_CONFIG"`
	PgQueryDebug bool     `envconfig:"PG_QUERY_DEBUG"`
	DbAutoInit   bool     `envconfig:"DB_AUTO_INIT"`
	SentryDSN    string   `envconfig:"SENTRY_DSN" `
	HTTPListen   string   `envconfig:"HTTP_LISTEN" default:":5001" required:"true"`
	APIPrefix    string   `envconfig:"API_PREFIX" default:"/api" desc:"API path prefix"`
	RedisURI     string   `envconfig:"redis_uri" default:"redis://localhost:6379/1" required:"true"`
	AllowOrigins []string `envconfig:"allow_origins" default:"*" desc:"cors"` // CORS: 允许的 Origin 调用来源
	AuthRequired bool     `envconfig:"Auth_Required"`
	AuthSecret   string   `envconfig:"Auth_Secret" desc:"for chatgpt-web session only"`
	CookieName   string   `envconfig:"Cookie_Name" default:"oaic" desc:"for oauth client"`
	CookiePath   string   `envconfig:"Cookie_Path" default:"/" desc:"for oauth client"`
	CookieDomain string   `envconfig:"Cookie_Domain" desc:"for oauth client"`
	CookieMaxAge int      `envconfig:"Cookie_MaxAge" desc:"seconds of cookie maxAge"`
	OAuthPathMCP string   `envconfig:"OAuth_Path_MCP" desc:"OAuth SP as A MCP Server"`

	WebAppPath string `envconfig:"Web_App_Path" default:"/" desc:"web app path for oauth redirect"`

	PresetFile  string `envconfig:"preset_file" desc:"custom welcome and messages"`
	QAEmbedding bool   `envconfig:"QA_Embedding" desc:"enable embed QA into prompt"`
	QAChatLog   bool   `envconfig:"QA_chat_log"`

	AskRateLimit string `envconfig:"Ask_Rate_Limit" default:"20-H"`

	DateInContext bool `envconfig:"date_in_context"`

	KeeperRole string   `envconfig:"Keeper_Role" default:"keeper" desc:"role required for write tools"`
	KeeperUIDs []string `envconfig:"Keeper_UIDs" desc:"uid list that bypasses role check"`

	// 相似度阈值 建议范围 0.39 - 0.65
	VectorThreshold float32 `envconfig:"Vector_Threshold" default:"0.39"`
	// 相似度匹配数量
	VectorLimit int `envconfig:"Vector_Limit" default:"5"`

	Embedding Provider
	Interact  Provider
	Summarize Provider
}

type Provider struct {
	APIKey string `envconfig:"Api_Key" required:"true"`
	URL    string `envconfig:"url" required:"true"`
	Model  string `envconfig:"MODEL" required:"true"`
	Type   string `envconfig:"type" default:"openai" desc:"provider type: openai, anthropic, openrouter, ollama"`
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
