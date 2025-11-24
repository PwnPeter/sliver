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
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPSender gère l'envoi d'emails via SMTP
type SMTPSender struct {
	config *ServerConfig
}

// NewSMTPSender crée un nouveau sender SMTP
func NewSMTPSender(config *ServerConfig) *SMTPSender {
	return &SMTPSender{
		config: config,
	}
}

// SendEmail envoie un email via SMTP
func (s *SMTPSender) SendEmail(to, subject, body string) error {
	// Construire l'adresse du serveur
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	// Authentification
	auth := smtp.PlainAuth(
		"",
		s.config.SMTPUsername,
		s.config.SMTPPassword,
		s.config.SMTPHost,
	)

	// Construire le message
	message := s.buildMessage(s.config.C2Email, to, subject, body)

	// Envoyer selon le mode TLS
	if s.config.SMTPUseTLS {
		return s.sendWithTLS(addr, auth, to, message)
	}

	// Envoi standard
	return smtp.SendMail(addr, auth, s.config.C2Email, []string{to}, []byte(message))
}

// sendWithTLS envoie un email avec STARTTLS
func (s *SMTPSender) sendWithTLS(addr string, auth smtp.Auth, to string, message string) error {
	// Connexion TLS
	tlsConfig := &tls.Config{
		ServerName: s.config.SMTPHost,
	}

	// Connexion au serveur SMTP
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial failed: %w", err)
	}
	defer client.Close()

	// STARTTLS
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("starttls failed: %w", err)
	}

	// Authentification
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth failed: %w", err)
	}

	// MAIL FROM
	if err := client.Mail(s.config.C2Email); err != nil {
		return fmt.Errorf("smtp mail from failed: %w", err)
	}

	// RCPT TO
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to failed: %w", err)
	}

	// DATA
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data failed: %w", err)
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		writer.Close()
		return fmt.Errorf("smtp write failed: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close failed: %w", err)
	}

	// QUIT
	return client.Quit()
}

// buildMessage construit le message email au format RFC 5322
func (s *SMTPSender) buildMessage(from, to, subject, body string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("From: %s\r\n", from))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", to))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(body)

	return builder.String()
}
