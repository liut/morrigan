package wecom

import (
	"fmt"

	"github.com/liut/morign/pkg/models/channel"
	"github.com/liut/morign/pkg/services/channels"
)

func init() {
	channels.RegisterChannel("wecom", New)
}

// New creates a WeCom channel adapter.
func New(opts map[string]any) (channel.Channel, error) {
	mode, _ := opts["mode"].(string)
	switch mode {
	case "websocket":
		return newWebSocket(opts)
	case "webhook":
		return newHTTP(opts)
	default:
		return nil, fmt.Errorf("wecom: unsupported mode %q (supported: websocket, webhook)", mode)
	}
}
