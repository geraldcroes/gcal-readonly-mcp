// gcal-readonly-mcp: A read-only Google Calendar MCP server
// Supports multiple Google accounts (personal, work, etc.)
// Only uses calendar.readonly scope - cannot modify any data
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ServerName    = "gcal-readonly-mcp"
	ServerVersion = "1.0.0"
)

func main() {
	// Parse command line flags
	addAccount := flag.String("add-account", "", "Add a new Google account (provide account name, e.g., 'personal' or 'work')")
	removeAccount := flag.String("remove-account", "", "Remove a configured Google account")
	listAccounts := flag.Bool("list-accounts", false, "List configured accounts")
	flag.Parse()

	// Handle account management commands
	if *removeAccount != "" {
		if err := RemoveAccount(*removeAccount); err != nil {
			log.Fatalf("Failed to remove account: %v", err)
		}
		fmt.Printf("Account '%s' removed successfully!\n", *removeAccount)
		os.Exit(0)
	}

	if *addAccount != "" {
		if err := AddAccount(*addAccount); err != nil {
			log.Fatalf("Failed to add account: %v", err)
		}
		fmt.Printf("Account '%s' added successfully!\n", *addAccount)
		os.Exit(0)
	}

	if *listAccounts {
		accounts, err := ListConfiguredAccounts()
		if err != nil {
			log.Fatalf("Failed to list accounts: %v", err)
		}
		if len(accounts) == 0 {
			fmt.Println("No accounts configured. Use --add-account <name> to add one.")
		} else {
			fmt.Println("Configured accounts:")
			for _, acc := range accounts {
				fmt.Printf("  - %s\n", acc)
			}
		}
		os.Exit(0)
	}

	// Start MCP server
	ctx := context.Background()
	server := NewCalendarServer()

	log.Printf("Starting %s v%s", ServerName, ServerVersion)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
