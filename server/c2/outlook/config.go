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
	"fmt"
	"time"
)

// ServerConfig configuration pour le serveur C2 Outlook
type ServerConfig struct {
	// SMTP Configuration (pour envoyer les commandes)
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPUseTLS   bool   `json:"smtp_use_tls"`

	// IMAP Configuration (pour recevoir les résultats)
	IMAPHost     string `json:"imap_host"`
	IMAPPort     int    `json:"imap_port"`
	IMAPUsername string `json:"imap_username"`
	IMAPPassword string `json:"imap_password"`
	IMAPUseTLS   bool   `json:"imap_use_tls"`
	IMAPFolder   string `json:"imap_folder"` // Dossier à surveiller (ex: "INBOX")

	// C2 Configuration
	C2Email       string        `json:"c2_email"`        // Adresse email du C2
	EncryptionKey []byte        `json:"encryption_key"`  // Clé AES-256 (32 bytes)
	PollInterval  time.Duration `json:"poll_interval"`   // Intervalle de polling IMAP
	Debug         bool          `json:"debug"`           // Mode debug

	// Gestion des emails
	MarkAsRead  bool `json:"mark_as_read"`  // Marquer les emails comme lus
	DeleteAfter bool `json:"delete_after"`  // Supprimer après traitement
}

// DefaultServerConfig retourne une configuration serveur par défaut
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUseTLS:   true,
		IMAPHost:     "imap.gmail.com",
		IMAPPort:     993,
		IMAPUseTLS:   true,
		IMAPFolder:   "INBOX",
		PollInterval: 30 * time.Second,
		MarkAsRead:   true,
		DeleteAfter:  false,
		Debug:        false,
	}
}

// Validate vérifie que la configuration serveur est valide
func (c *ServerConfig) Validate() error {
	if c.SMTPHost == "" {
		return fmt.Errorf("smtp_host is required")
	}
	if c.SMTPPort < 1 || c.SMTPPort > 65535 {
		return fmt.Errorf("smtp_port must be between 1 and 65535")
	}
	if c.SMTPUsername == "" {
		return fmt.Errorf("smtp_username is required")
	}
	if c.SMTPPassword == "" {
		return fmt.Errorf("smtp_password is required")
	}

	if c.IMAPHost == "" {
		return fmt.Errorf("imap_host is required")
	}
	if c.IMAPPort < 1 || c.IMAPPort > 65535 {
		return fmt.Errorf("imap_port must be between 1 and 65535")
	}
	if c.IMAPUsername == "" {
		return fmt.Errorf("imap_username is required")
	}
	if c.IMAPPassword == "" {
		return fmt.Errorf("imap_password is required")
	}
	if c.IMAPFolder == "" {
		return fmt.Errorf("imap_folder is required")
	}

	if c.C2Email == "" {
		return fmt.Errorf("c2_email is required")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("encryption_key must be 32 bytes (AES-256)")
	}
	if c.PollInterval < 1*time.Second {
		return fmt.Errorf("poll_interval must be at least 1 second")
	}

	return nil
}
