package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"

	"mkepub/internal/config"
)

func Send(cfg config.MailConfig, epubPath string) error {
	data, err := os.ReadFile(epubPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	filename := filepath.Base(epubPath)
	body, contentType, err := buildMessage(cfg.From, cfg.To, filename, data)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	auth := smtp.PlainAuth("", cfg.From, cfg.Password, cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, buildRaw(cfg.From, cfg.To, contentType, body))
}

func buildRaw(from, to, contentType string, body []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", to)
	fmt.Fprintf(&buf, "Subject: \r\n")
	fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&buf, "Content-Type: %s\r\n", contentType)
	fmt.Fprintf(&buf, "\r\n")
	buf.Write(body)
	return buf.Bytes()
}

func buildMessage(from, to, filename string, data []byte) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Plain text body (Kindle just needs the attachment)
	th := textproto.MIMEHeader{}
	th.Set("Content-Type", "text/plain; charset=UTF-8")
	pw, err := w.CreatePart(th)
	if err != nil {
		return nil, "", err
	}
	pw.Write([]byte("Sent by mkepub."))

	// EPUB attachment
	ah := textproto.MIMEHeader{}
	ah.Set("Content-Type", "application/epub+zip")
	ah.Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
	ah.Set("Content-Transfer-Encoding", "base64")
	aw, err := w.CreatePart(ah)
	if err != nil {
		return nil, "", err
	}
	enc := base64.NewEncoder(base64.StdEncoding, aw)
	enc.Write(data)
	enc.Close()

	w.Close()
	return buf.Bytes(), "multipart/mixed; boundary=" + w.Boundary(), nil
}
