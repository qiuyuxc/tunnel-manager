package services

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
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

// Send delivers a multipart/alternative message (plain + HTML) to one
// recipient. Bodies travel quoted-printable encoded, which keeps the wire
// format pure ASCII and immune to charset guessing.
func (m *Mailer) Send(to, subject, plain, htmlBody string) error {
	if m == nil || m.host == "" || m.port <= 0 || m.from == "" {
		return errors.New("SMTP 未配置")
	}
	fromHeader, fromAddr := m.fromHeader()

	var message bytes.Buffer
	mw := multipart.NewWriter(&message)

	headers := []struct{ key, value string }{
		{"From", fromHeader},
		{"To", to},
		{"Subject", encodeHeader(subject)},
		{"Date", time.Now().Format(time.RFC1123Z)},
		{"MIME-Version", "1.0"},
		{"Content-Type", "multipart/alternative; boundary=" + mw.Boundary()},
	}
	for _, header := range headers {
		message.WriteString(header.key + ": " + header.value + "\r\n")
	}
	message.WriteString("\r\n")

	textPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("build text part: %w", err)
	}
	qw := quotedprintable.NewWriter(textPart)
	if _, err := qw.Write([]byte(plain)); err != nil {
		return fmt.Errorf("write text part: %w", err)
	}
	if err := qw.Close(); err != nil {
		return fmt.Errorf("close text part: %w", err)
	}

	htmlPart, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=utf-8"},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("build html part: %w", err)
	}
	hw := quotedprintable.NewWriter(htmlPart)
	if _, err := hw.Write([]byte(htmlBody)); err != nil {
		return fmt.Errorf("write html part: %w", err)
	}
	if err := hw.Close(); err != nil {
		return fmt.Errorf("close html part: %w", err)
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("finalize message: %w", err)
	}

	client, err := m.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	if m.username != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
				return fmt.Errorf("SMTP 认证失败: %w", err)
			}
		}
	}

	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	dataWriter, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := dataWriter.Write(message.Bytes()); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := dataWriter.Close(); err != nil {
		return fmt.Errorf("close message: %w", err)
	}
	return client.Quit()
}

// fromHeader builds the From header, Q-encoding a non-ASCII display name.
func (m *Mailer) fromHeader() (string, string) {
	from := strings.TrimSpace(m.from)
	idx := strings.Index(from, "<")
	if idx <= 0 {
		return from, from
	}
	end := strings.Index(from, ">")
	if end <= idx {
		return from, from
	}
	name := strings.TrimSpace(from[:idx])
	addr := from[idx+1 : end]
	if !isASCII(name) {
		name = mime.QEncoding.Encode("utf-8", name)
	}
	return name + " <" + addr + ">", addr
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// connect dials the relay. Encrypted mode tries implicit TLS first (port
// 465 style) and falls back to STARTTLS (port 587 style); plain mode skips
// TLS entirely.
func (m *Mailer) connect() (*smtp.Client, error) {
	addr := net.JoinHostPort(m.host, strconv.Itoa(m.port))
	encrypted := m.tlsMode != "plain"

	if encrypted {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: m.host})
		if err == nil {
			client, err := smtp.NewClient(conn, m.host)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("greet SMTP: %w", err)
			}
			return client, nil
		}
		// Not implicit TLS on this port: fall through and try STARTTLS.
	}

	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		if encrypted {
			return nil, fmt.Errorf("connect SMTP（已尝试 TLS 与 STARTTLS）: %w", err)
		}
		return nil, fmt.Errorf("connect SMTP: %w", err)
	}
	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("greet SMTP: %w", err)
	}

	if encrypted {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
				return nil, fmt.Errorf("start TLS: %w", err)
			}
		} else {
			return nil, errors.New("SMTP 服务器不支持 STARTTLS")
		}
	}
	return client, nil
}

// encodeHeader folds a UTF-8 header into RFC 2047 base64 form.
func encodeHeader(subject string) string {
	return "=?utf-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
}
