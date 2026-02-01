# gcal-readonly-mcp

A **read-only** Google Calendar MCP server written in Go. Supports multiple Google accounts.

## Security

- **Read-only scope only**: Uses `calendar.readonly` scope - cannot create, modify, or delete any events
- **Local token storage**: OAuth tokens stored locally in `~/.config/gcal-readonly-mcp/tokens/`
- **No third-party services**: Communicates only with Google Calendar API
- **No telemetry**: Zero analytics or data collection

## Features

- Multi-account support (personal, work, etc.)
- List calendars from all configured accounts
- List events with date filtering and search
- Check free/busy availability across accounts

## Setup

### 1. Google Cloud Console Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project (or select existing)
3. Enable the **Google Calendar API**
4. Create OAuth 2.0 credentials:
   - Application type: **Desktop app**
   - Download the JSON file
5. Place the credentials at: `~/.config/gcal-readonly-mcp/credentials.json`

### 2. Add Google Accounts

```bash
# Add your personal account
./gcal-readonly-mcp --add-account personal

# Add your work account
./gcal-readonly-mcp --add-account work

# List configured accounts
./gcal-readonly-mcp --list-accounts
```

Each `--add-account` command opens a browser for OAuth authentication.

### 3. Configure Claude Code

Add to your `~/.claude/settings.json`:

```json
{
  "mcpServers": {
    "gcal-readonly": {
      "command": "/Users/geraldcroes/workspace/geraldcroes/gcal-readonly-mcp/gcal-readonly-mcp"
    }
  }
}
```

## Available Tools

| Tool | Description |
|------|-------------|
| `list_accounts` | List all configured Google accounts |
| `list_calendars` | List calendars (all accounts or specific) |
| `list_events` | List events with date/query filters |
| `get_event` | Get detailed event information |
| `check_availability` | Check free/busy status |

## Example Queries

Once configured, you can ask:

- "What's on my calendar today?"
- "List my events for next week"
- "Am I free Thursday afternoon?"
- "Show my work calendar events"

## File Structure

```
~/.config/gcal-readonly-mcp/
├── config.json           # Account configuration
├── credentials.json      # Google OAuth credentials (you provide)
└── tokens/
    ├── personal.json     # Token for 'personal' account
    └── work.json         # Token for 'work' account
```

## License

MIT
