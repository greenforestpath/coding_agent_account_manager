package refresh

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/health"
)

func TestRefreshGeminiToken(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		err := r.ParseForm()
		if err != nil {
			t.Fatal(err)
		}

		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("expected grant_type refresh_token, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "test-refresh" {
			t.Errorf("unexpected refresh token: %s", r.Form.Get("refresh_token"))
		}
		if r.Form.Get("client_id") != "test-client" {
			t.Errorf("unexpected client id: %s", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "test-secret" {
			t.Errorf("unexpected client secret: %s", r.Form.Get("client_secret"))
		}

		resp := GoogleTokenResponse{
			AccessToken: "new-access-token",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Override URL
	oldURL := GeminiTokenURL
	GeminiTokenURL = server.URL
	defer func() { GeminiTokenURL = oldURL }()

	// Test
	resp, err := RefreshGeminiToken(context.Background(), "test-client", "test-secret", "test-refresh")
	if err != nil {
		t.Fatalf("RefreshGeminiToken failed: %v", err)
	}

	if resp.AccessToken != "new-access-token" {
		t.Errorf("expected access token new-access-token, got %s", resp.AccessToken)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("expected expires in 3600, got %d", resp.ExpiresIn)
	}
}

func TestReadADC(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "adc.json")

	adc := ADC{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RefreshToken: "test-refresh",
		Type:         "authorized_user",
	}
	data, _ := json.Marshal(adc)
	os.WriteFile(path, data, 0600)

	readAdc, err := ReadADC(path)
	if err != nil {
		t.Fatalf("ReadADC failed: %v", err)
	}

	if readAdc.ClientID != "test-client" {
		t.Errorf("expected client id test-client, got %s", readAdc.ClientID)
	}
}

func TestUpdateGeminiHealth(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "health.json")
	store := health.NewStorage(storePath)

	resp := &GoogleTokenResponse{
		ExpiresIn: 3600,
	}

	if err := UpdateGeminiHealth(store, "gemini", "default", resp); err != nil {
		t.Fatalf("UpdateGeminiHealth failed: %v", err)
	}

	// Verify
	h, _ := store.GetProfile("gemini", "default")
	if h == nil {
		t.Fatal("health profile not created")
	}

	if h.TokenExpiresAt.IsZero() {
		t.Error("expiry not set")
	}
	// Check if expiry is roughly 1 hour from now
	if time.Until(h.TokenExpiresAt) < 59*time.Minute {
		t.Error("expiry too soon")
	}
}

func TestUpdateGeminiAuth(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "settings.json")

	original := map[string]any{
		"access_token":  "old-access",
		"refresh_token": "old-refresh",
		"expiry":        "2020-01-01T00:00:00Z",
		"other_field":   "preserve-me",
	}
	raw, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	resp := &GoogleTokenResponse{
		AccessToken: "new-access",
		ExpiresIn:   3600,
		TokenType:   "Bearer",
	}

	if err := UpdateGeminiAuth(path, resp); err != nil {
		t.Fatalf("UpdateGeminiAuth() error = %v", err)
	}

	updatedRaw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var updated map[string]any
	if err := json.Unmarshal(updatedRaw, &updated); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got := updated["access_token"]; got != "new-access" {
		t.Fatalf("access_token = %v, want %v", got, "new-access")
	}
	if got := updated["other_field"]; got != "preserve-me" {
		t.Fatalf("other_field = %v, want %v", got, "preserve-me")
	}
	if got := updated["expiry"]; got == "" {
		t.Fatalf("expiry is empty")
	}
}
