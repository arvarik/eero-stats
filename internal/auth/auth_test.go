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

func TestSaveSession(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Valid Session Save", func(t *testing.T) {
		sessionPath := filepath.Join(tempDir, "new_dir", "session.json")
		userToken := "valid_token_456"

		if err := saveSession(sessionPath, userToken); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify the file was created and contains the correct data
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			t.Fatalf("failed to read created session file: %v", err)
		}

		expectedJSON := "{\n  \"user_token\": \"valid_token_456\"\n}"
		if string(data) != expectedJSON {
			t.Errorf("expected JSON:\n%s\ngot:\n%s", expectedJSON, string(data))
		}
	})

	t.Run("Error Creating Directory", func(t *testing.T) {
		// Create a file where the directory should be
		fileAsDir := filepath.Join(tempDir, "file_as_dir")
		if err := os.WriteFile(fileAsDir, []byte("not a dir"), 0644); err != nil {
			t.Fatalf("failed to create dummy file: %v", err)
		}

		sessionPath := filepath.Join(fileAsDir, "session.json")
		userToken := "token"

		if err := saveSession(sessionPath, userToken); err == nil {
			t.Errorf("expected error when parent path is a file, got nil")
		} else if err.Error()[:26] != "creating session directory" {
			t.Errorf("expected error starting with 'creating session directory', got '%v'", err)
		}
	})

	t.Run("Error Writing File", func(t *testing.T) {
		// Create a directory where the file should be
		dirAsFile := filepath.Join(tempDir, "dir_as_file.json")
		if err := os.MkdirAll(dirAsFile, 0750); err != nil {
			t.Fatalf("failed to create dummy dir: %v", err)
		}

		userToken := "token"

		if err := saveSession(dirAsFile, userToken); err == nil {
			t.Errorf("expected error when target is a directory, got nil")
		}
	})
}
