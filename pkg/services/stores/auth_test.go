package stores

import (
	"testing"
)

// TestTokenUserKeyFormat tests the key format
func TestTokenUserKeyFormat(t *testing.T) {
	key := tokenUserKey("my-token")
	expected := "tk-o-user-my-token"
	if key != expected {
		t.Errorf("expected key %q, got %q", expected, key)
	}
}

// TestUserEncodeDecode tests User encoding and decoding
func TestUserEncodeDecode(t *testing.T) {
	user := User{
		OID:  "test-oid",
		UID:  "testuser",
		Name: "Test User",
	}

	encoded, err := user.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	var loaded User
	if err := loaded.Decode(encoded); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if loaded.OID != user.OID {
		t.Errorf("expected OID %q, got %q", user.OID, loaded.OID)
	}
	if loaded.UID != user.UID {
		t.Errorf("expected UID %q, got %q", user.UID, loaded.UID)
	}
	if loaded.Name != user.Name {
		t.Errorf("expected Name %q, got %q", user.Name, loaded.Name)
	}
}
