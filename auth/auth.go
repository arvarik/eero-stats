package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"eero-stats/config"

	"github.com/arvarik/eero-go/eero"
)

var SessionFile = "/app/data/.eero_session.json"

func init() {
	// Fallback to local path if not running inside the Docker container
	if _, err := os.Stat("/app/data"); os.IsNotExist(err) {
		SessionFile = "data/app/.eero_session.json"
	}
}

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

	// 1. Check if session exists and is valid
	if err := restoreSession(client); err == nil {
		slog.Info("Restored cached session. Validating...")
		if _, err := client.Account.Get(ctx); err == nil {
			slog.Info("Eero session is valid")
			return client, nil
		}
		slog.Warn("Cached session expired or invalid, re-authenticating")
	} else {
		slog.Info("No valid cached session found, starting interactive login")
	}

	// 2. Interactive CLI flow
	if err := interactiveLogin(ctx, client, cfg.EeroLogin); err != nil {
		return nil, fmt.Errorf("interactive login failed: %w", err)
	}

	return client, nil
}

func restoreSession(client *eero.Client) error {
	data, err := os.ReadFile(SessionFile)
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

	// Hydrate the HTTP client
	return client.SetSessionCookie(sess.UserToken)
}

func interactiveLogin(ctx context.Context, client *eero.Client, loginID string) error {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Login challenge
	identifier := loginID
	if identifier == "" {
		fmt.Print("Enter your eero email or phone: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		identifier = strings.TrimSpace(input)
	}

	loginResp, err := client.Auth.Login(ctx, identifier)
	if err != nil {
		return fmt.Errorf("initiating login: %w", err)
	}

	fmt.Printf("Verification code sent to %s. Enter verification code: ", identifier)

	// Step 2: Verify code
	code, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading verification code: %w", err)
	}
	code = strings.TrimSpace(code)

	if err := client.Auth.Verify(ctx, code); err != nil {
		return fmt.Errorf("verifying code: %w", err)
	}
	slog.Info("Authenticated successfully!")

	// 3. Save the session
	sess := sessionData{UserToken: loginResp.UserToken}
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		slog.Warn("Could not marshal session data", "error", err)
		return nil
	}

	if err := os.WriteFile(SessionFile, data, 0600); err != nil {
		slog.Warn("Could not cache session to disk", "error", err, "file", SessionFile)
	} else {
		slog.Info("Session cached securely", "file", SessionFile)
	}

	return nil
}
