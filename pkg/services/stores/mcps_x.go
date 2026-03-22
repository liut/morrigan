package stores

import (
	"context"

	"github.com/liut/morign/pkg/models/mcps"
)

func PatchMCPServer(obj *mcps.Server) {
	if obj.HeaderCate.HasAuthorization() {
		obj.HeaderFunc = func(ctx context.Context) map[string]string {
			if tk := OAuthTokenFromContext(ctx); len(tk) > 0 {
				return map[string]string{"Authorization": "Bearer " + tk}
			}
			return nil
		}
	} else if obj.HeaderCate.HasOwnerSession() {
		obj.HeaderFunc = func(ctx context.Context) map[string]string {
			csid := ConvoIDFromContext(ctx)
			if user, ok := UserFromContext(ctx); ok && len(csid) > 0 {
				logger().Debugw("got scarf", "uid", user.OID, "csid", csid)
				return map[string]string{
					"X-Owner-Id":   user.OID,
					"X-Session-Id": csid,
				}
			}

			return nil
		}
	}
}

func (s *mcpStore) afterLoadServer(ctx context.Context, obj *mcps.Server) error {
	PatchMCPServer(obj)
	return nil
}

func (s *mcpStore) afterListServer(ctx context.Context, spec *MCPServerSpec, data mcps.Servers) error {
	for i := range data {
		_ = s.afterLoadServer(ctx, &data[i])
	}
	return nil
}
