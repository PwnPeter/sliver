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

import "time"

// Outlook folder type constants
const (
	OlFolderDeletedItems = 3
	OlFolderOutbox       = 4
	OlFolderSentMail     = 5
	OlFolderInbox        = 6
	OlFolderCalendar     = 9
	OlFolderContacts     = 10
	OlFolderJournal      = 11
	OlFolderNotes        = 12
	OlFolderTasks        = 13
	OlFolderDrafts       = 16
	OlFolderJunk         = 23
)

// OutlookConfig configuration du transport Outlook
type OutlookConfig struct {
	Enabled       bool          `json:"enabled"`
	C2Email       string        `json:"c2_email"`       // Adresse email du C2
	PollInterval  time.Duration `json:"poll_interval"`  // Délai entre checks
	FolderType    int           `json:"folder_type"`    // 6=Inbox, 16=Drafts
	EncryptionKey []byte        `json:"encryption_key"` // Clé AES-256 (32 bytes)
	DeleteAfter   bool          `json:"delete_after"`   // Supprimer les emails après lecture
	Jitter        int           `json:"jitter"`         // Jitter en % (0-100)
	SessionID     string        `json:"session_id"`     // Identifiant unique de la session
	MaxRetries    int           `json:"max_retries"`    // Nombre max de tentatives en cas d'erreur
	Debug         bool          `json:"debug"`          // Mode debug pour logs verbeux
}

// DefaultConfig retourne une config par défaut
func DefaultConfig() *OutlookConfig {
	return &OutlookConfig{
		Enabled:      false,
		PollInterval: 60 * time.Second,
		FolderType:   OlFolderDrafts, // Drafts par défaut (moins suspect)
		DeleteAfter:  false,
		Jitter:       30, // 30% de jitter
		MaxRetries:   3,
		Debug:        false,
	}
}

// Validate vérifie que la configuration est valide
func (c *OutlookConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.C2Email == "" {
		return &ConfigError{Field: "c2_email", Message: "C2 email address is required"}
	}

	if c.PollInterval < 1*time.Second {
		return &ConfigError{Field: "poll_interval", Message: "Poll interval must be at least 1 second"}
	}

	if len(c.EncryptionKey) != 32 {
		return &ConfigError{Field: "encryption_key", Message: "Encryption key must be 32 bytes (AES-256)"}
	}

	if c.Jitter < 0 || c.Jitter > 100 {
		return &ConfigError{Field: "jitter", Message: "Jitter must be between 0 and 100"}
	}

	if c.SessionID == "" {
		return &ConfigError{Field: "session_id", Message: "Session ID is required"}
	}

	return nil
}

// ConfigError représente une erreur de configuration
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "outlook config error [" + e.Field + "]: " + e.Message
}
