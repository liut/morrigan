package feishu

import (
	"fmt"

	"github.com/liut/morign/pkg/models/channel"
	"github.com/liut/morign/pkg/services/channels"
)

func init() {
	channels.RegisterChannel("feishu", New)
}

// New creates a Feishu channel adapter.
func New(opts map[string]any) (channel.Channel, error) {
	mode, _ := opts["mode"].(string)
	switch mode {
	case "websocket":
		return newWebSocket(opts)
	case "webhook":
		return newWebhook(opts)
	default:
		return nil, fmt.Errorf("feishu: unsupported mode %q (supported: websocket, webhook)", mode)
	}
}
