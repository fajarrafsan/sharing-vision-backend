// Package mail mengirim email transaksional: verifikasi alamat dan reset
// password.
package mail

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
}

type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// LogMailer tidak mengirim apa pun, hanya mencatat isi email ke log. Dipakai
// di development ketika SMTP belum diatur, supaya tautan verifikasi dan reset
// bisa disalin dari log.
type LogMailer struct{}

func (LogMailer) Send(ctx context.Context, m Message) error {
	slog.InfoContext(ctx, "email (tidak dikirim, SMTP belum diatur)", "to", m.To, "subject", m.Subject, "body", m.Text)
	return nil
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// SMTPMailer mengirim lewat server SMTP. Port 465 memakai TLS langsung; port
// lain memakai STARTTLS bila server mendukungnya.
type SMTPMailer struct {
	cfg SMTPConfig
}

func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{cfg: cfg}
}

func (s *SMTPMailer) Send(ctx context.Context, m Message) error {
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	dialer := &net.Dialer{Timeout: 15 * time.Second}

	var conn net.Conn
	var err error
	if s.cfg.Port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: s.cfg.Host})
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return err
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok && s.cfg.Port != "465" {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return err
		}
	}
	if s.cfg.Username != "" {
		// smtp.PlainAuth menolak mengirim password tanpa TLS kecuali ke localhost.
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return err
		}
	}

	from := addressOnly(s.cfg.From)
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(m.To); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(Compose(s.cfg.From, m, time.Now())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func addressOnly(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimSuffix(strings.TrimSpace(from[i+1:]), ">")
	}
	return strings.TrimSpace(from)
}

// Compose membentuk pesan MIME teks biasa berenkode quoted-printable.
func Compose(from string, m Message, now time.Time) []byte {
	var buf bytes.Buffer
	id := make([]byte, 12)
	_, _ = rand.Read(id)
	domain := "warta.local"
	if at := strings.LastIndex(addressOnly(from), "@"); at >= 0 {
		domain = addressOnly(from)[at+1:]
	}

	headers := []string{
		"From: " + from,
		"To: " + m.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", m.Subject),
		"Date: " + now.Format(time.RFC1123Z),
		fmt.Sprintf("Message-ID: <%s@%s>", hex.EncodeToString(id), domain),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: quoted-printable",
	}
	buf.WriteString(strings.Join(headers, "\r\n"))
	buf.WriteString("\r\n\r\n")

	qp := quotedprintable.NewWriter(&buf)
	_, _ = qp.Write([]byte(strings.ReplaceAll(m.Text, "\n", "\r\n")))
	_ = qp.Close()
	return buf.Bytes()
}

// Async mengirim di latar belakang supaya permintaan HTTP tidak menunggu
// server SMTP. Kegagalan dicatat ke log.
type Async struct {
	Mailer  Mailer
	Timeout time.Duration
}

func (a Async) Send(_ context.Context, m Message) error {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), a.Timeout)
		defer cancel()
		if err := a.Mailer.Send(ctx, m); err != nil {
			slog.Error("gagal mengirim email", "to", m.To, "subject", m.Subject, "error", err)
		}
	}()
	return nil
}
