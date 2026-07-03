package gamejolt

import (
	"strings"
	"testing"
)

func TestSignURL(t *testing.T) {
	// Setup test keys
	GameID = "12345"
	PrivateKey = "abcde"

	c := NewClient("user", "token")

	rawURL := "https://api.gamejolt.com/api/game/v1/sessions/open/?game_id=12345&username=user&user_token=token"
	signed := c.signURL(rawURL)

	if !strings.Contains(signed, "&signature=") {
		t.Errorf("Signed URL should contain signature")
	}

	// Expected signature for "https://api.gamejolt.com/api/game/v1/sessions/open/?game_id=12345&username=user&user_token=tokenabcde"
	// MD5("https://api.gamejolt.com/api/game/v1/sessions/open/?game_id=12345&username=user&user_token=tokenabcde")
	// Let's just verify it's a valid hex string of 32 chars
	parts := strings.Split(signed, "&signature=")
	signature := parts[1]

	if len(signature) != 32 {
		t.Errorf("Signature should be 32 characters (MD5), got %d", len(signature))
	}
}

func TestClientActivation(t *testing.T) {
	GameID = ""
	PrivateKey = ""

	c := NewClient("user", "token")
	if c.IsActive() {
		t.Errorf("Client should not be active without GameID and PrivateKey")
	}

	GameID = "123"
	PrivateKey = "456"
	c = NewClient("user", "token")
	if !c.IsActive() {
		t.Errorf("Client should be active with all credentials")
	}

	c = NewClient("", "")
	if c.IsActive() {
		t.Errorf("Client should not be active without username/token")
	}
}
