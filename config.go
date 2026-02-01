package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the configuration for all accounts
type Config struct {
	Accounts map[string]AccountConfig `json:"accounts"`
}

// AccountConfig holds configuration for a single Google account
type AccountConfig struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "gcal-readonly-mcp"), nil
}

// GetTokenPath returns the token file path for an account
func GetTokenPath(accountName string) (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "tokens", accountName+".json"), nil
}

// GetCredentialsPath returns the OAuth credentials file path
func GetCredentialsPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "credentials.json"), nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Accounts: make(map[string]AccountConfig)}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Accounts == nil {
		config.Accounts = make(map[string]AccountConfig)
	}

	return &config, nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Ensure tokens directory exists
	tokensDir := filepath.Join(configDir, "tokens")
	if err := os.MkdirAll(tokensDir, 0700); err != nil {
		return fmt.Errorf("failed to create tokens directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := filepath.Join(configDir, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ListConfiguredAccounts returns the list of configured account names
func ListConfiguredAccounts() ([]string, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	accounts := make([]string, 0, len(config.Accounts))
	for name := range config.Accounts {
		accounts = append(accounts, name)
	}
	return accounts, nil
}

// RemoveAccount removes an account and its token
func RemoveAccount(name string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Check if account exists
	if _, exists := config.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' not found", name)
	}

	// Delete token file
	tokenPath, err := GetTokenPath(name)
	if err != nil {
		return err
	}
	os.Remove(tokenPath) // Ignore error if file doesn't exist

	// Remove from config
	delete(config.Accounts, name)

	return SaveConfig(config)
}

// AddAccount adds a new account and triggers OAuth flow
func AddAccount(name string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	// Check if account already exists
	if _, exists := config.Accounts[name]; exists {
		return fmt.Errorf("account '%s' already exists", name)
	}

	// Check for credentials file
	credPath, err := GetCredentialsPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		configDir, _ := GetConfigDir()
		return fmt.Errorf("credentials.json not found. Please place your Google OAuth credentials at: %s", filepath.Join(configDir, "credentials.json"))
	}

	// Perform OAuth flow
	email, err := PerformOAuthFlow(name)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	// Save account config
	config.Accounts[name] = AccountConfig{
		Name:  name,
		Email: email,
	}

	return SaveConfig(config)
}
