//go:build integration

// Integration Tests for stores package
//
// Prerequisites (本地测试前必须执行):
//
//  1. 创建测试数据库:
//     psql -U morrigan -c "CREATE DATABASE morrigan_test;"
//
//  2. 安装 vector 扩展（需要管理员权限）:
//     psql -U postgres -d morrigan_test -c "CREATE EXTENSION IF NOT EXISTS vector;"
//     psql -U postgres -d morrigan_test -c "ALTER EXTENSION vector SET SCHEMA public;"
//
//  3. 运行集成测试:
//     go test -tags=integration -v ./pkg/services/stores/...
//
// 环境变量:
//
//	TEST_DB_DSN      - 完整连接字符串 (优先级最高)
//	TEST_DB_HOST     - 数据库主机 (默认 localhost)
//	TEST_DB_PORT     - 数据库端口 (默认 5432)
//	TEST_DB_USER     - 数据库用户 (默认 morrigan)
//	TEST_DB_PASSWORD - 数据库密码
//	TEST_DB_NAME     - 数据库名 (默认 morrigan_test)
package stores

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cupogo/andvari/models/oid"
	"github.com/liut/morrigan/pkg/models/corpus"
	"github.com/liut/morrigan/pkg/settings"
)

// getTestDBDSN 构建测试数据库连接字符串
func getTestDBDSN() string {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn != "" {
		return dsn
	}

	host := getEnvOrDefault("TEST_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_DB_PORT", "5432")
	user := getEnvOrDefault("TEST_DB_USER", "morrigan")
	password := getEnvOrDefault("TEST_DB_PASSWORD", "")
	dbname := getEnvOrDefault("TEST_DB_NAME", "morrigan_test")

	if password != "" {
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=disable", user, host, port, dbname)
}

// getEnvOrDefault 获取环境变量或默认值
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// TestMain 测试入口
func TestMain(m *testing.M) {
	// 替换数据库名为测试库
	dsn := getTestDBDSN()
	settings.Current.PgStoreDSN = dsn
	fmt.Printf("Using test DSN: %s\n", dsn)

	// 初始化数据库（InitSchemas + RunMigrations）
	if err := InitDB(); err != nil {
		fmt.Printf("warning: InitDB failed: %v\n", err)
	}

	ret := m.Run()
	os.Exit(ret)
}

// 测试标题，用于避免测试数据冲突
func testDocTitle() string {
	return fmt.Sprintf("test-doc-%d", os.Getpid())
}

func TestIntegration_DocumentCRUD(t *testing.T) {
	// 检查是否有 Embedding API_KEY（CreateDocument 需要 embedding）
	if settings.Current.Embedding.APIKey == "" {
		t.Skip("Embedding.APIKey not set, skipping document CRUD test (requires embedding)")
	}

	sto := Sgt()
	ctx := context.Background()

	title := testDocTitle()

	// Create
	doc, err := sto.Cob().CreateDocument(ctx, corpus.DocumentBasic{
		Title:   title,
		Heading: "Test Heading",
		Content: "Test Content",
	})
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	if doc == nil {
		t.Fatal("CreateDocument returned nil")
	}

	// Get
	found, err := sto.Cob().GetDocument(ctx, doc.ID.String())
	if err != nil {
		t.Fatalf("GetDocument failed: %v", err)
	}
	if found == nil {
		t.Fatal("GetDocument returned nil")
	}

	// Delete
	if err := sto.Cob().DeleteDocument(ctx, doc.ID.String()); err != nil {
		t.Logf("cleanup failed: %v", err)
	}
}

func TestIntegration_ListDocuments(t *testing.T) {
	sto := Sgt()
	ctx := context.Background()

	spec := &CobDocumentSpec{}
	data, total, err := sto.Cob().ListDocument(ctx, spec)
	if err != nil {
		t.Fatalf("ListDocument failed: %v", err)
	}

	t.Logf("Total documents: %d, returned: %d", total, len(data))
}

func TestIntegration_DocVectorCRUD(t *testing.T) {
	// 检查是否有 Embedding API_KEY
	if settings.Current.Embedding.APIKey == "" {
		t.Skip("Embedding.APIKey not set, skipping vector test (requires embedding)")
	}

	sto := Sgt()
	ctx := context.Background()

	docTitle := testDocTitle()
	doc, err := sto.Cob().CreateDocument(ctx, corpus.DocumentBasic{
		Title:   docTitle,
		Heading: "Vector Test",
		Content: "Content for vector test",
	})
	if err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	docID := doc.ID.String()

	// 清理
	defer sto.Cob().DeleteDocument(ctx, docID)

	// Create DocVector
	vec, err := sto.Cob().CreateDocVector(ctx, corpus.DocVectorBasic{
		DocID:   doc.ID,
		Subject: "test-subject",
		Vector:  corpus.Vector{0.1, 0.2, 0.3},
	})
	if err != nil {
		t.Fatalf("CreateDocVector failed: %v", err)
	}
	if vec == nil {
		t.Fatal("CreateDocVector returned nil")
	}

	// Get DocVector
	found, err := sto.Cob().GetDocVector(ctx, vec.ID.String())
	if err != nil {
		t.Fatalf("GetDocVector failed: %v", err)
	}
	if found == nil {
		t.Fatal("GetDocVector returned nil")
	}

	// Delete DocVector
	err = sto.Cob().DeleteDocVector(ctx, vec.ID.String())
	if err != nil {
		t.Fatalf("DeleteDocVector failed: %v", err)
	}
}

func TestIntegration_ChatLogCRUD(t *testing.T) {
	sto := Sgt()
	ctx := context.Background()

	// Create ChatLog
	chatID := oid.OID(os.Getpid())
	log, err := sto.Cob().CreateChatLog(ctx, corpus.ChatLogBasic{
		ChatID:   chatID,
		Question: "What is Go?",
		Answer:   "Go is a programming language.",
	})
	if err != nil {
		t.Fatalf("CreateChatLog failed: %v", err)
	}
	if log == nil {
		t.Fatal("CreateChatLog returned nil")
	}

	// Get ChatLog
	found, err := sto.Cob().GetChatLog(ctx, log.ID.String())
	if err != nil {
		t.Fatalf("GetChatLog failed: %v", err)
	}
	if found == nil {
		t.Fatal("GetChatLog returned nil")
	}
	if found.Question != "What is Go?" {
		t.Errorf("expected question 'What is Go?', got %q", found.Question)
	}

	// Delete ChatLog
	err = sto.Cob().DeleteChatLog(ctx, log.ID.String())
	if err != nil {
		t.Fatalf("DeleteChatLog failed: %v", err)
	}
}
