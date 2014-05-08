package goemail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"strings"
)

var (
	ErrInvalidScheme = errors.New("invalid scheme")
	ErrNoRecipients  = errors.New("no recipients specified")
)

type Message struct {
	from            string
	to              []string
	cc              []string
	bcc             []string
	subject         string
	body            string
	bodyContentType string
	attachments     map[string][]byte
}

type SMTP struct {
	scheme   string
	server   string
	auth     *smtp.Auth
	hostname string
}

func newMessage(from, subject, body, contenttype string) *Message {
	m := Message{
		from:            from,
		subject:         subject,
		body:            body,
		bodyContentType: contenttype,
		attachments:     make(map[string][]byte),
	}
	return &m
}

// NewMessage creates a new text/plain email.
func NewMessage(from, subject, body string) *Message {
	return newMessage(from, subject, body, "text/plain")
}

// NewHTMLMessage creates a new text/html email.
func NewHTMLMessage(from, subject, body string) *Message {
	return newMessage(from, subject, body, "text/html")
}

func (m *Message) AddAttachment(filename string, attachment []byte) {
	m.attachments[filename] = attachment
}

func (m *Message) AddAttachmentFromFile(filename string) error {
	a, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	m.attachments[filename] = a
	return nil
}

func (m *Message) AddCC(emailAddr string) {
	m.cc = append(m.cc, emailAddr)
}

func (m *Message) AddBCC(emailAddr string) {
	m.bcc = append(m.bcc, emailAddr)
}

// AddTo adds an email address to the To recipients
func (m *Message) AddTo(emailAddr string) {
	m.to = append(m.to, emailAddr)
}

// Body returns the formatted message body.
func (m *Message) Body() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("From: " + m.from + "\n")
	buf.WriteString("To: " + strings.Join(m.to, ",") + "\n")
	if len(m.cc) > 0 {
		buf.WriteString("Cc: " + strings.Join(m.cc, ",") + "\n")
	}
	buf.WriteString("Subject: " + m.subject + "\n")
	buf.WriteString("MIME-Version: 1.0\n")

	boundary := "mnwKuycHoXCwn9S5UY6avz8ZGJPEeUdMPS"

	if len(m.attachments) > 0 {
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\n")
		buf.WriteString("--" + boundary + "\n")
	}

	buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\n", m.bodyContentType))
	buf.WriteString(m.body)

	if len(m.attachments) > 0 {
		for k, v := range m.attachments {
			buf.WriteString("\n\n--" + boundary + "\n")
			buf.WriteString("Content-Type: application/octet-stream\n")
			buf.WriteString("Content-Transfer-Encoding: base64\n")
			buf.WriteString("Content-Disposition: attachment; filename=\"" + k + "\"\n\n")

			b64 := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
			base64.StdEncoding.Encode(b64, v)
			buf.Write(b64)
			buf.WriteString("\n--" + boundary)
		}

		buf.WriteString("--")
	}

	return buf.Bytes()
}

// From returns the sender's email address
func (m *Message) From() string {
	return m.from
}

// Recipients returns an array of all the recipients, which includes
// To, CC, and BCC
func (m *Message) Recipients() []string {
	rcpts := make([]string, 0, len(m.to)+len(m.cc)+len(m.bcc))
	rcpts = append(rcpts, m.to...)
	rcpts = append(rcpts, m.cc...)
	rcpts = append(rcpts, m.bcc...)
	return rcpts
}

// NewSMTP is called with smtp[s]://[username:[password]]@server:[port]
func NewSMTP(rawUrl string) (*SMTP, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	if url.Scheme != "smtp" && url.Scheme != "smtps" {
		return nil, ErrInvalidScheme
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	mysmtp := &SMTP{
		scheme:   url.Scheme,
		hostname: hostname,
	}

	_, _, err = net.SplitHostPort(url.Host)
	if err != nil {
		mysmtp.server = url.Host + ":25"
	} else {
		mysmtp.server = url.Host
	}

	if url.User != nil {
		p, _ := url.User.Password()
		a := smtp.CRAMMD5Auth(url.User.Username(), p)
		mysmtp.auth = &a
	}
	return mysmtp, nil
}

func (s *SMTP) Send(msg *Message) error {
	var conn net.Conn
	var err error

	recipients := msg.Recipients()
	if len(recipients) < 1 {
		return ErrNoRecipients
	}

	if s.scheme == "smtps" {
		tlscfg := tls.Config{
			InsecureSkipVerify: true,
		}
		if conn, err = tls.Dial("tcp", s.server, &tlscfg); err != nil {
			return err
		}
	} else {
		if conn, err = net.Dial("tcp", s.server); err != nil {
			return err
		}
	}

	client, err := smtp.NewClient(conn, s.server)
	if err != nil {
		return err
	}

	// Send HELO/EHLO
	if err = client.Hello(s.hostname); err != nil {
		return err
	}

	// Check if STARTTLS is supported if not smtps.
	if s.scheme != "smtps" {
		hasStartTLS, _ := client.Extension("STARTTLS")
		if hasStartTLS {
			tlscfg := tls.Config{
				InsecureSkipVerify: true,
			}
			if err = client.StartTLS(&tlscfg); err != nil {
				return err
			}
		}
	}

	// Send authentication, if specified
	if s.auth != nil {
		if err = client.Auth(*s.auth); err != nil {
			return err
		}
	}

	// MAIL FROM
	if err = client.Mail(msg.From()); err != nil {
		return err
	}

	// RCPT TO
	for _, rcpt := range msg.Recipients() {
		if err = client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	// DATA
	dataBuf, err := client.Data()
	if err != nil {
		return err
	}
	dataBuf.Write(msg.Body())
	dataBuf.Close()

	if err = client.Quit(); err != nil {
		return err
	}

	return nil
}
