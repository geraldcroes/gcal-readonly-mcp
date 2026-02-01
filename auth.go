package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// IMPORTANT: Read-only scope only!
var oauthScopes = []string{
	calendar.CalendarReadonlyScope,
}

// GetOAuthConfig loads the OAuth configuration from credentials.json
func GetOAuthConfig() (*oauth2.Config, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	config, err := google.ConfigFromJSON(data, oauthScopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return config, nil
}

// LoadToken loads the OAuth token for an account
func LoadToken(accountName string) (*oauth2.Token, error) {
	tokenPath, err := GetTokenPath(accountName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &token, nil
}

// SaveToken saves the OAuth token for an account
func SaveToken(accountName string, token *oauth2.Token) error {
	tokenPath, err := GetTokenPath(accountName)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Write with restricted permissions (owner only)
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

// PerformOAuthFlow performs the OAuth flow for a new account
func PerformOAuthFlow(accountName string) (string, error) {
	config, err := GetOAuthConfig()
	if err != nil {
		return "", err
	}

	// Set redirect URL for local callback
	config.RedirectURL = "http://localhost:8089/callback"

	// Channels for communication
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)
	serverReady := make(chan bool, 1)

	// Create a new mux for this server (don't use DefaultServeMux)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code provided", http.StatusBadRequest)
			errChan <- fmt.Errorf("no code in callback")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Authentication Successful</title>
</head>
<body style="font-family: sans-serif; text-align: center; padding-top: 50px;">
<h1>Authentication successful!</h1>
<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
		codeChan <- code
	})

	server := &http.Server{
		Addr:    ":8089",
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		// Signal that we're about to start listening
		serverReady <- true
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for server to be ready
	<-serverReady
	time.Sleep(100 * time.Millisecond) // Small delay to ensure server is listening

	// Generate auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Printf("\n=== OAuth Authentication for account '%s' ===\n", accountName)
	fmt.Printf("\n1. Opening browser for authentication...\n")
	fmt.Printf("\n2. If the callback doesn't work, copy the authorization code from the URL\n")
	fmt.Printf("   (the 'code' parameter) and paste it below.\n")
	fmt.Printf("\nAuth URL (if browser doesn't open):\n%s\n\n", authURL)

	// Try to open browser
	openBrowser(authURL)

	// Start a goroutine to read manual input
	go func() {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Paste authorization code here (or wait for automatic callback): ")
		input, err := reader.ReadString('\n')
		if err == nil {
			input = strings.TrimSpace(input)
			if input != "" {
				codeChan <- input
			}
		}
	}()

	// Wait for code (from callback or manual input) or error
	var code string
	select {
	case code = <-codeChan:
		fmt.Println("\n✅ Authorization code received!")
	case err := <-errChan:
		server.Close()
		return "", err
	case <-time.After(5 * time.Minute):
		server.Close()
		return "", fmt.Errorf("timeout waiting for authorization")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Exchange code for token
	ctx = context.Background()
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save token
	if err := SaveToken(accountName, token); err != nil {
		return "", err
	}

	// Get user email
	client := config.Client(ctx, token)
	srv, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create calendar service: %w", err)
	}

	// Get primary calendar to extract email
	cal, err := srv.CalendarList.Get("primary").Do()
	if err != nil {
		return "", fmt.Errorf("failed to get primary calendar: %w", err)
	}

	fmt.Printf("\n✅ Account '%s' configured successfully!\n", accountName)
	fmt.Printf("   Email: %s\n\n", cal.Id)

	return cal.Id, nil
}

// GetCalendarService returns a Calendar service for the specified account
func GetCalendarService(ctx context.Context, accountName string) (*calendar.Service, error) {
	config, err := GetOAuthConfig()
	if err != nil {
		return nil, err
	}

	token, err := LoadToken(accountName)
	if err != nil {
		return nil, fmt.Errorf("failed to load token for account '%s': %w", accountName, err)
	}

	client := config.Client(ctx, token)
	return calendar.NewService(ctx, option.WithHTTPClient(client))
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) {
	// Using 'open' command on macOS
	cmd := exec.Command("open", url)
	cmd.Start()
}
