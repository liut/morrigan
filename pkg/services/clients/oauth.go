package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/liut/morign/pkg/settings"
)

// issueTokenReq 向认证服务签发 token 的请求
type issueTokenReq struct {
	// ObjID 为目标 account 的 OID 字符串
	ObjID string `json:"objID"`
}

// issueTokenResp 认证服务 token 签发响应
type issueTokenResp struct {
	// Token 为签发的 token key，可直接用作 Authorization: Bearer <token>
	Token string `json:"token"`
	// ExpiresIn token 有效秒数
	ExpiresIn int32 `json:"expiresIn"`
	// Error 仅在非 200 时返回
	Error string `json:"error,omitempty"`
}

var oauthClient = &http.Client{Timeout: 5 * time.Second}

// IssueToken 从认证服务获取指定 account 的 token，返回 token 和过期秒数
func IssueToken(ctx context.Context, accountID string) (token string, expiresIn int32, err error) {
	body, err := json.Marshal(issueTokenReq{ObjID: accountID})
	if err != nil {
		return "", 0, fmt.Errorf("oauth issue-token marshal: %w", err)
	}
	url := strings.TrimSuffix(settings.Current.OAuthInternalURL, "/") + "/token/issue"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Auth", settings.Current.ServiceAuthKey)

	resp, err := oauthClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("oauth issue-token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", 0, fmt.Errorf("oauth issue-token read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		slog.Info("oauth issue-token fail", "status", resp.StatusCode, "url", url)
		return "", 0, fmt.Errorf("oauth issue-token: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result issueTokenResp
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", 0, fmt.Errorf("oauth issue-token decode: %w", err)
	}
	return result.Token, result.ExpiresIn, nil
}
