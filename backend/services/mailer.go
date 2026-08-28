package services

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/models"
)

// Mailer sends mail through a configurable SMTP relay.
type Mailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	tlsMode  string
}

// NewMailer builds a relay client from the stored settings. password is the
// decrypted relay password.
func NewMailer(settings models.SMTPSettings, password string) *Mailer {
	return &Mailer{
		host:     settings.Host,
		port:     settings.Port,
		username: settings.Username,
		password: password,
		from:     settings.From,
		tlsMode:  settings.TLSMode,
	}
}

// Send delivers a plain-text message to one recipient.
func (m *Mailer) Send(to, subject, body string) error {
	if m == nil || m.host == "" || m.port <= 0 || m.from == "" {
		return errors.New("SMTP 未配置")
	}
	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	from := m.from
	if idx := strings.Index(m.from, "<"); idx >= 0 {
		if end := strings.Index(m.from, ">"); end > idx {
			from = strings.TrimSpace(m.from[idx+1 : end])
		}
	}

	headers := []struct{ key, value string }{
		{"From", m.from},
		{"To", to},
		{"Subject", encodeHeader(subject)},
		{"MIME-Version", "1.0"},
		{"Content-Type", "text/plain; charset=\"utf-8\""},
		{"Date", time.Now().Format(time.RFC1123Z)},
	}
	var message strings.Builder
	for _, header := range headers {
		message.WriteString(header.key + ": " + header.value + "\r\n")
	}
	message.WriteString("\r\n" + base64.StdEncoding.EncodeToString([]byte(body)))

	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return fmt.Errorf("connect SMTP: %w", err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("greet SMTP: %w", err)
	}
	defer client.Close()

	mode := strings.ToLower(m.tlsMode)
	if mode == "" {
		mode = "starttls"
	}

	if mode == "ssl" {
		return errors.New("tls_mode=ssl 需使用隐式 TLS 客户端，请改用 starttls 模式或 465+ssl 支持（暂以 starttls 提供）")
	}
	if ok, _ := client.Extension("STARTTLS"); mode == "starttls" {
		if !ok {
			return errors.New("SMTP 服务器不支持 STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("start TLS: %w", err)
		}
	}

	if m.username != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
				return fmt.Errorf("SMTP 认证失败: %w", err)
			}
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := writer.Write([]byte(message.String())); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close message: %w", err)
	}
	return client.Quit()
}

// encodeHeader folds a UTF-8 subject into RFC 2047 base64 form.
func encodeHeader(subject string) string {
	return "=?utf-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
}
