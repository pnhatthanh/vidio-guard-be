package pkg

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"

	"github.com/pnhatthanh/vidio-guard-be/internal/config"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
	SendHTML(ctx context.Context, to, subject, htmlBody, plainBody string) error
	Enabled() bool
}

type smtpMailer struct {
	cfg config.SMTPConfig
}

func NewMailer(cfg config.SMTPConfig) Mailer {
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) Enabled() bool {
	return strings.TrimSpace(m.cfg.Host) != ""
}

func (m *smtpMailer) Send(ctx context.Context, to, subject, body string) error {
	return m.sendMessage(ctx, to, subject, buildPlainEmail(to, subject, body))
}

func (m *smtpMailer) SendHTML(ctx context.Context, to, subject, htmlBody, plainBody string) error {
	return m.sendMessage(ctx, to, subject, buildAlternativeEmail(to, subject, plainBody, htmlBody))
}

func (m *smtpMailer) sendMessage(ctx context.Context, to, subject string, msg []byte) error {
	_ = ctx
	if !m.Enabled() {
		return fmt.Errorf("smtp is not configured")
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("recipient is required")
	}

	from := strings.TrimSpace(m.cfg.From)
	if from == "" {
		from = strings.TrimSpace(m.cfg.User)
	}

	fullMsg := append(buildEmailHeaders(from, to, subject), msg...)
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)

	if m.cfg.UseTLS && m.cfg.Port == 465 {
		return m.sendTLS(addr, from, to, fullMsg)
	}
	return m.sendStartTLS(addr, from, to, fullMsg)
}

func (m *smtpMailer) sendStartTLS(addr, from, to string, msg []byte) error {
	host := m.cfg.Host
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if err := c.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if m.cfg.User != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}

func (m *smtpMailer) sendTLS(addr, from, to string, msg []byte) error {
	host := m.cfg.Host
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if m.cfg.User != "" {
		auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Password, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return c.Quit()
}

func buildEmailHeaders(from, to, subject string) []byte {
	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodedSubject,
		"MIME-Version: 1.0",
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n")
}

func buildPlainEmail(to, subject, body string) []byte {
	_ = to
	_ = subject
	part := []string{
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"",
	}
	var buf bytes.Buffer
	buf.WriteString(strings.Join(part, "\r\n"))
	w := quotedprintable.NewWriter(&buf)
	_, _ = w.Write([]byte(body))
	_ = w.Close()
	return buf.Bytes()
}

func buildAlternativeEmail(to, subject, plainBody, htmlBody string) []byte {
	_ = to
	_ = subject
	boundary := "vidio-guard-boundary"

	var buf bytes.Buffer
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")

	writePart := func(contentType, body string) {
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: " + contentType + "\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		w := quotedprintable.NewWriter(&buf)
		_, _ = w.Write([]byte(body))
		_ = w.Close()
		buf.WriteString("\r\n")
	}

	writePart("text/plain; charset=UTF-8", plainBody)
	writePart("text/html; charset=UTF-8", htmlBody)
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes()
}
