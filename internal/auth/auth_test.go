package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/arvarik/eero-go/eero"
)

func TestRestoreSession(t *testing.T) {
	client, err := eero.NewClient()
	if err != nil {
		t.Fatalf("failed to create eero client: %v", err)
	}

	tempDir := t.TempDir()

	t.Run("Valid Session", func(t *testing.T) {
		sessionPath := filepath.Join(tempDir, "valid_session.json")
		validJSON := []byte(`{"user_token": "valid_token_123"}`)
		if err := os.WriteFile(sessionPath, validJSON, 0644); err != nil {
			t.Fatalf("failed to write valid session file: %v", err)
		}

		if err := restoreSession(client, sessionPath); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Missing File", func(t *testing.T) {
		sessionPath := filepath.Join(tempDir, "non_existent.json")

		if err := restoreSession(client, sessionPath); err == nil {
			t.Errorf("expected error for missing file, got nil")
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		sessionPath := filepath.Join(tempDir, "invalid_session.json")
		invalidJSON := []byte(`{"user_token": "valid_token_123"`) // Missing closing brace
		if err := os.WriteFile(sessionPath, invalidJSON, 0644); err != nil {
			t.Fatalf("failed to write invalid session file: %v", err)
		}

		if err := restoreSession(client, sessionPath); err == nil {
			t.Errorf("expected error for invalid JSON, got nil")
		}
	})

	t.Run("Empty User Token", func(t *testing.T) {
		sessionPath := filepath.Join(tempDir, "empty_token.json")
		emptyTokenJSON := []byte(`{"user_token": ""}`)
		if err := os.WriteFile(sessionPath, emptyTokenJSON, 0644); err != nil {
			t.Fatalf("failed to write empty token session file: %v", err)
		}

		err := restoreSession(client, sessionPath)
		if err == nil {
			t.Errorf("expected error for empty user_token, got nil")
		} else if err.Error() != "empty user_token in session file" {
			t.Errorf("expected error 'empty user_token in session file', got '%v'", err)
		}
	})
}
