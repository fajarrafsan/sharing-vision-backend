package mail

import (
	"context"
	"io"
	"mime/quotedprintable"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestCompose(t *testing.T) {
	msg := Compose("Warta <noreply@warta.id>", ResetPassword("budi@warta.test", "Budi", "https://warta.id/reset-password?token=abc", "1 jam"), time.Unix(0, 0))
	text := string(msg)

	for _, want := range []string{
		"From: Warta <noreply@warta.id>\r\n",
		"To: budi@warta.test\r\n",
		"Subject: Atur ulang password Warta\r\n",
		"@warta.id>\r\n",
		"Content-Transfer-Encoding: quoted-printable\r\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("header %q tidak ada:\n%s", want, text)
		}
	}

	body, _ := io.ReadAll(quotedprintable.NewReader(strings.NewReader(text[strings.Index(text, "\r\n\r\n")+4:])))
	if !strings.Contains(string(body), "https://warta.id/reset-password?token=abc") {
		t.Fatalf("tautan harus utuh setelah decode:\n%s", body)
	}
}

// fakeSMTP adalah server SMTP minimal yang merekam pesan yang diterima.
func fakeSMTP(t *testing.T) (addr string, received chan string) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	received = make(chan string, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tp := textproto.NewConn(conn)
		_ = tp.PrintfLine("220 fake")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			switch cmd := strings.ToUpper(strings.Fields(line + " x")[0]); cmd {
			case "EHLO", "HELO":
				_ = tp.PrintfLine("250 fake")
			case "DATA":
				_ = tp.PrintfLine("354 go")
				data, _ := tp.ReadDotBytes()
				received <- string(data)
				_ = tp.PrintfLine("250 ok")
			case "QUIT":
				_ = tp.PrintfLine("221 bye")
				return
			default:
				_ = tp.PrintfLine("250 ok")
			}
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), received
}

func TestSMTPMailerSends(t *testing.T) {
	addr, received := fakeSMTP(t)
	host, port, _ := net.SplitHostPort(addr)

	m := NewSMTPMailer(SMTPConfig{Host: host, Port: port, From: "Warta <noreply@warta.id>"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Send(ctx, VerifyEmail("budi@warta.test", "Budi", "https://warta.id/verify-email?token=xyz", "24 jam")); err != nil {
		t.Fatal(err)
	}

	select {
	case data := <-received:
		if !strings.Contains(data, "Subject: Verifikasi email akun Warta") {
			t.Fatalf("pesan diterima:\n%s", data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server tidak menerima pesan")
	}
}
