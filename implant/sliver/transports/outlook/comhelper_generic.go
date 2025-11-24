//go:build !windows
// +build !windows

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
	"time"
)

// OutlookCOMHelper stub pour les plateformes non-Windows
type OutlookCOMHelper struct{}

// NewOutlookCOMHelper retourne une erreur sur les plateformes non-Windows
func NewOutlookCOMHelper() (*OutlookCOMHelper, error) {
	return nil, errors.New("outlook COM transport is only available on Windows")
}

// GetFolder stub
func (h *OutlookCOMHelper) GetFolder(folderType int) (interface{}, error) {
	return nil, errors.New("outlook COM transport is only available on Windows")
}

// ReadUnreadEmails stub
func (h *OutlookCOMHelper) ReadUnreadEmails(folder interface{}, senderFilter string) ([]*EmailMessage, error) {
	return nil, errors.New("outlook COM transport is only available on Windows")
}

// SendEmail stub
func (h *OutlookCOMHelper) SendEmail(to, subject, body string) error {
	return errors.New("outlook COM transport is only available on Windows")
}

// MarkAsRead stub
func (h *OutlookCOMHelper) MarkAsRead(item interface{}) error {
	return errors.New("outlook COM transport is only available on Windows")
}

// DeleteEmail stub
func (h *OutlookCOMHelper) DeleteEmail(item interface{}) error {
	return errors.New("outlook COM transport is only available on Windows")
}

// Close stub
func (h *OutlookCOMHelper) Close() {}

// EmailMessage stub
type EmailMessage struct {
	Subject      string
	Body         string
	Sender       string
	ReceivedTime interface{}
	Item         interface{}
}

// GetReceivedTime stub
func (e *EmailMessage) GetReceivedTime() (time.Time, error) {
	return time.Time{}, errors.New("outlook COM transport is only available on Windows")
}
