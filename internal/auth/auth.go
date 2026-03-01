// Package auth handles authentication with the Eero cloud API. It attempts to
// restore a previously cached session token from disk and falls back to an
// interactive 2FA login flow via standard input when no valid session exists.
package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/arvarik/eero-stats/internal/config"

	"github.com/arvarik/eero-go/eero"
)

type sessionData struct {
	UserToken string `json:"user_token"`
}

// Init initializes the Eero client, attempting to load a cached session
// or falling back to interactive 2FA login via standard input.
func Init(ctx context.Context, cfg *config.Config) (*eero.Client, error) {
	client, err := eero.NewClient()
	if err != nil {
		return nil, fmt.Errorf("creating eero client: %w", err)
	}

	// 1. Check if session exists and is valid.
	if err := restoreSession(client, cfg.EeroSessionPath); err == nil {
		slog.Info("Restored cached session. Validating...")
		if _, err := client.Account.Get(ctx); err == nil {
			slog.Info("Eero session is valid")
			return client, nil
		}
		slog.Warn("Cached session expired or invalid, re-authenticating")
	} else {
		slog.Info("No valid cached session found, starting interactive login")
	}

	// 2. Interactive CLI flow.
	if err := interactiveLogin(ctx, client, cfg.EeroLogin, cfg.EeroSessionPath); err != nil {
		return nil, fmt.Errorf("interactive login failed: %w", err)
	}

	return client, nil
}

func restoreSession(client *eero.Client, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var sess sessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return err
	}
	if sess.UserToken == "" {
		return errors.New("empty user_token in session file")
	}

	// Hydrate the HTTP client with the cached token.
	return client.SetSessionCookie(sess.UserToken)
}

func interactiveLogin(ctx context.Context, client *eero.Client, loginID, path string) error {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Login challenge.
	loginResp, err := client.Auth.Login(ctx, loginID)
	if err != nil {
		return fmt.Errorf("initiating login: %w", err)
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Printf("  Verification code sent to %s\n", loginID)
	fmt.Println("========================================")
	fmt.Print("Enter verification code: ")

	// Step 2: Verify code.
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading verification code: %w", err)
	}
	code = strings.TrimSpace(code)

	if err := client.Auth.Verify(ctx, code); err != nil {
		return fmt.Errorf("verifying code: %w", err)
	}
	slog.Info("Authenticated successfully!")

	// 3. Save the session to disk for future restarts.
	if err := saveSession(path, loginResp.UserToken); err != nil {
		slog.Warn("Could not cache session to disk", "error", err, "file", path)
	} else {
		slog.Info("Session cached securely", "file", path)
	}

	return nil
}

// saveSession persists the user token to disk, creating the parent directory
// if it doesn't already exist.
func saveSession(path, userToken string) error {
	sess := sessionData{UserToken: userToken}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling session: %w", err)
	}

	// Ensure the parent directory exists (e.g., first run before Docker
	// volume is fully initialized).
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("creating session directory %s: %w", dir, err)
	}

	return os.WriteFile(path, data, 0600)
}
