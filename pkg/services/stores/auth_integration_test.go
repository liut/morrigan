//go:build integration

package stores

import (
	"context"
	"os"
	"testing"

	"github.com/liut/morign/pkg/settings"
)

func TestIntegration_SaveAndLoadUserWithToken(t *testing.T) {
	// 使用真实的 Redis
	redisURI := getTestRedisURI()
	if redisURI == "" {
		t.Skip("M_TEST_REDIS_URI not set")
	}

	// 强制使用指定的 Redis URI
	origURI := settings.Current.RedisURI
	settings.Current.RedisURI = redisURI
	defer func() { settings.Current.RedisURI = origURI }()

	ctx := context.Background()
	token := "integration-test-token"
	testUser := User{
		OID:  "test-oid-integration",
		UID:  "integrationuser",
		Name: "Integration Test User",
	}

	// 保存用户
	err := SaveUserWithToken(ctx, &testUser, token)
	if err != nil {
		t.Fatalf("SaveUserWithToken failed: %v", err)
	}

	// 加载用户
	loadedUser, err := LoadUserFromToken(ctx, token)
	if err != nil {
		t.Fatalf("LoadUserFromToken failed: %v", err)
	}

	// 验证
	if loadedUser.OID != testUser.OID {
		t.Errorf("expected OID %q, got %q", testUser.OID, loadedUser.OID)
	}
	if loadedUser.UID != testUser.UID {
		t.Errorf("expected UID %q, got %q", testUser.UID, loadedUser.UID)
	}
	if loadedUser.Name != testUser.Name {
		t.Errorf("expected Name %q, got %q", testUser.Name, loadedUser.Name)
	}

	// 清理
	SgtRC().Del(ctx, tokenUserKey(token))
}

func TestIntegration_LoadUserFromToken_NotFound(t *testing.T) {
	redisURI := getTestRedisURI()
	if redisURI == "" {
		t.Skip("M_TEST_REDIS_URI not set")
	}

	origURI := settings.Current.RedisURI
	settings.Current.RedisURI = redisURI
	defer func() { settings.Current.RedisURI = origURI }()

	ctx := context.Background()
	_, err := LoadUserFromToken(ctx, "nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent token")
	}
}

func getTestRedisURI() string {
	return os.Getenv("M_TEST_REDIS_URI")
}
