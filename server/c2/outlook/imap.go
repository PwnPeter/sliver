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
	"io"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// IMAPReceiver gère la réception d'emails via IMAP
type IMAPReceiver struct {
	config *ServerConfig
	client *client.Client
}

// NewIMAPReceiver crée un nouveau receiver IMAP
func NewIMAPReceiver(config *ServerConfig) *IMAPReceiver {
	return &IMAPReceiver{
		config: config,
	}
}

// Connect se connecte au serveur IMAP
func (r *IMAPReceiver) Connect() error {
	var err error
	addr := fmt.Sprintf("%s:%d", r.config.IMAPHost, r.config.IMAPPort)

	if r.config.IMAPUseTLS {
		r.client, err = client.DialTLS(addr, nil)
	} else {
		r.client, err = client.Dial(addr)
	}

	if err != nil {
		return fmt.Errorf("imap dial failed: %w", err)
	}

	// Authentification
	if err := r.client.Login(r.config.IMAPUsername, r.config.IMAPPassword); err != nil {
		return fmt.Errorf("imap login failed: %w", err)
	}

	if r.config.Debug {
		log.Printf("[Outlook C2] Connected to IMAP server: %s", addr)
	}

	return nil
}

// Disconnect se déconnecte du serveur IMAP
func (r *IMAPReceiver) Disconnect() error {
	if r.client != nil {
		return r.client.Logout()
	}
	return nil
}

// Email représente un email reçu
type Email struct {
	UID     uint32
	Subject string
	From    string
	Body    string
	Date    string
}

// FetchUnreadEmails récupère les emails non lus du dossier configuré
func (r *IMAPReceiver) FetchUnreadEmails() ([]*Email, error) {
	if r.client == nil {
		return nil, fmt.Errorf("not connected to IMAP server")
	}

	// Sélectionner le dossier
	mbox, err := r.client.Select(r.config.IMAPFolder, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %w", r.config.IMAPFolder, err)
	}

	if mbox.Messages == 0 {
		return []*Email{}, nil
	}

	// Rechercher les messages non lus
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	uids, err := r.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("imap search failed: %w", err)
	}

	if len(uids) == 0 {
		return []*Email{}, nil
	}

	if r.config.Debug {
		log.Printf("[Outlook C2] Found %d unread emails", len(uids))
	}

	// Récupérer les messages
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}

	go func() {
		done <- r.client.UidFetch(seqSet, items, messages)
	}()

	var emails []*Email
	for msg := range messages {
		email, err := r.parseMessage(msg, section)
		if err != nil {
			if r.config.Debug {
				log.Printf("[Outlook C2] Error parsing message: %v", err)
			}
			continue
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("imap fetch failed: %w", err)
	}

	return emails, nil
}

// parseMessage parse un message IMAP en Email
func (r *IMAPReceiver) parseMessage(msg *imap.Message, section *imap.BodySectionName) (*Email, error) {
	if msg.Envelope == nil {
		return nil, fmt.Errorf("message has no envelope")
	}

	// Extraire le corps
	body := ""
	if literal := msg.GetBody(section); literal != nil {
		bodyBytes, err := io.ReadAll(literal)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
		body = string(bodyBytes)
	}

	// Extraire l'expéditeur
	from := ""
	if len(msg.Envelope.From) > 0 {
		from = msg.Envelope.From[0].Address()
	}

	return &Email{
		UID:     msg.Uid,
		Subject: msg.Envelope.Subject,
		From:    from,
		Body:    body,
		Date:    msg.Envelope.Date.String(),
	}, nil
}

// MarkAsRead marque un email comme lu
func (r *IMAPReceiver) MarkAsRead(uid uint32) error {
	if r.client == nil {
		return fmt.Errorf("not connected to IMAP server")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}

	if err := r.client.UidStore(seqSet, item, flags, nil); err != nil {
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	return nil
}

// DeleteEmail supprime un email
func (r *IMAPReceiver) DeleteEmail(uid uint32) error {
	if r.client == nil {
		return fmt.Errorf("not connected to IMAP server")
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uid)

	// Marquer comme supprimé
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}

	if err := r.client.UidStore(seqSet, item, flags, nil); err != nil {
		return fmt.Errorf("failed to mark for deletion: %w", err)
	}

	// Expunge pour supprimer définitivement
	if err := r.client.Expunge(nil); err != nil {
		return fmt.Errorf("failed to expunge: %w", err)
	}

	return nil
}
