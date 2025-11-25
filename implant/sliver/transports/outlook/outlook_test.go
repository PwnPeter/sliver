package outlook

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"strings"
	"testing"
	"time"
)

func TestEncodeDecodeMessage(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	msg := &C2Message{
		Type:      "cmd",
		TaskID:    "test-123",
		SessionID: "session-456",
		Payload:   "d2hvYW1p", // "whoami" en base64
	}

	subject, body, err := EncodeMessage(msg, key)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	if subject == "" {
		t.Error("Subject should not be empty")
	}

	if body == "" {
		t.Error("Body should not be empty")
	}

	// Vérifier que le body contient les marqueurs C2
	if !containsC2Data(body) {
		t.Error("Body should contain C2DATA markers")
	}

	decoded, err := DecodeMessage(body, key)
	if err != nil {
		t.Fatalf("DecodeMessage failed: %v", err)
	}

	if decoded.TaskID != msg.TaskID {
		t.Errorf("TaskID mismatch: got %s, want %s", decoded.TaskID, msg.TaskID)
	}

	if decoded.SessionID != msg.SessionID {
		t.Errorf("SessionID mismatch: got %s, want %s", decoded.SessionID, msg.SessionID)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, msg.Type)
	}

	if decoded.Payload != msg.Payload {
		t.Errorf("Payload mismatch: got %s, want %s", decoded.Payload, msg.Payload)
	}

	// Vérifier que le timestamp est récent
	age := time.Now().Unix() - decoded.Timestamp
	if age > 10 || age < 0 {
		t.Errorf("Timestamp age should be recent, got %d seconds", age)
	}
}

func TestEncodeDecodeWithWrongKey(t *testing.T) {
	key1, _ := GenerateEncryptionKey()
	key2, _ := GenerateEncryptionKey()

	msg := &C2Message{
		Type:      "cmd",
		TaskID:    "test-123",
		SessionID: "session-456",
		Payload:   "d2hvYW1p",
	}

	_, body, err := EncodeMessage(msg, key1)
	if err != nil {
		t.Fatalf("EncodeMessage failed: %v", err)
	}

	// Tenter de décoder avec une mauvaise clé
	_, err = DecodeMessage(body, key2)
	if err == nil {
		t.Error("DecodeMessage should fail with wrong key")
	}
}

func TestKeyGeneration(t *testing.T) {
	key1, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("Key length should be 32 bytes, got %d", len(key1))
	}

	key2, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	if string(key1) == string(key2) {
		t.Error("Keys should be unique")
	}
}

func TestKeyHexConversion(t *testing.T) {
	key, err := GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("GenerateEncryptionKey failed: %v", err)
	}

	hexKey := KeyToHex(key)
	if hexKey == "" {
		t.Error("Hex key should not be empty")
	}

	if len(hexKey) != 64 { // 32 bytes = 64 hex characters
		t.Errorf("Hex key length should be 64, got %d", len(hexKey))
	}

	// Convertir de retour
	key2, err := HexToKey(hexKey)
	if err != nil {
		t.Fatalf("HexToKey failed: %v", err)
	}

	if string(key) != string(key2) {
		t.Error("Key should match after hex conversion")
	}
}

func TestConfigValidation(t *testing.T) {
	config := DefaultConfig()

	if config.PollInterval != 60*time.Second {
		t.Errorf("Default poll interval should be 60s, got %v", config.PollInterval)
	}

	if config.FolderType != OlFolderDrafts {
		t.Errorf("Default folder should be Drafts (%d), got %d", OlFolderDrafts, config.FolderType)
	}

	if config.Jitter != 30 {
		t.Errorf("Default jitter should be 30, got %d", config.Jitter)
	}

	// Tester la validation avec une config invalide
	config.Enabled = true
	err := config.Validate()
	if err == nil {
		t.Error("Validation should fail with incomplete config")
	}

	// Compléter la config
	key, _ := GenerateEncryptionKey()
	config.C2Email = "c2@example.com"
	config.EncryptionKey = key
	config.SessionID = "session-123"

	err = config.Validate()
	if err != nil {
		t.Errorf("Validation should succeed with complete config: %v", err)
	}
}

func TestConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		modifyFn  func(*OutlookConfig)
		expectErr bool
	}{
		{
			name:      "Disabled config should pass",
			modifyFn:  func(c *OutlookConfig) { c.Enabled = false },
			expectErr: false,
		},
		{
			name: "Missing C2 email",
			modifyFn: func(c *OutlookConfig) {
				c.Enabled = true
				c.C2Email = ""
			},
			expectErr: true,
		},
		{
			name: "Invalid poll interval",
			modifyFn: func(c *OutlookConfig) {
				c.Enabled = true
				c.C2Email = "test@example.com"
				c.PollInterval = 0
			},
			expectErr: true,
		},
		{
			name: "Invalid encryption key length",
			modifyFn: func(c *OutlookConfig) {
				c.Enabled = true
				c.C2Email = "test@example.com"
				c.PollInterval = 60 * time.Second
				c.EncryptionKey = []byte("short")
			},
			expectErr: true,
		},
		{
			name: "Invalid jitter",
			modifyFn: func(c *OutlookConfig) {
				c.Enabled = true
				c.C2Email = "test@example.com"
				c.PollInterval = 60 * time.Second
				key, _ := GenerateEncryptionKey()
				c.EncryptionKey = key
				c.Jitter = 150
			},
			expectErr: true,
		},
		{
			name: "Missing session ID",
			modifyFn: func(c *OutlookConfig) {
				c.Enabled = true
				c.C2Email = "test@example.com"
				c.PollInterval = 60 * time.Second
				key, _ := GenerateEncryptionKey()
				c.EncryptionKey = key
				c.Jitter = 30
				c.SessionID = ""
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			tt.modifyFn(config)

			err := config.Validate()
			if tt.expectErr && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestFakeContentGeneration(t *testing.T) {
	// Tester la génération de sujet
	subject1 := generateFakeSubject()
	if subject1 == "" {
		t.Error("Generated subject should not be empty")
	}

	subject2 := generateFakeSubject()
	if subject2 == "" {
		t.Error("Generated subject should not be empty")
	}

	// Tester la génération de corps d'email
	body1 := generateFakeEmailBody()
	if body1 == "" {
		t.Error("Generated body should not be empty")
	}

	body2 := generateFakeEmailBody()
	if body2 == "" {
		t.Error("Generated body should not be empty")
	}

	// Les sujets et corps peuvent être différents (randomisation)
	// mais on vérifie juste qu'ils sont générés correctement
}

func TestDecodeMessageWithInvalidData(t *testing.T) {
	key, _ := GenerateEncryptionKey()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "No C2 markers",
			body: "This is a normal email without any C2 data",
		},
		{
			name: "Invalid base64",
			body: "<!--C2DATA-->\nThis is not valid base64!!!\n<!--/C2DATA-->",
		},
		{
			name: "Empty C2 data",
			body: "<!--C2DATA-->\n\n<!--/C2DATA-->",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodeMessage(tt.body, key)
			if err == nil {
				t.Error("DecodeMessage should fail with invalid data")
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key, _ := GenerateEncryptionKey()
	plaintext := []byte("This is a test message")

	ciphertext, err := encryptAES(plaintext, key)
	if err != nil {
		t.Fatalf("encryptAES failed: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("Ciphertext should not be empty")
	}

	decrypted, err := decryptAES(ciphertext, key)
	if err != nil {
		t.Fatalf("decryptAES failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

// Helper function
func containsC2Data(body string) bool {
	return strings.Contains(body, "<!--C2DATA-->") &&
		strings.Contains(body, "<!--/C2DATA-->")
}
