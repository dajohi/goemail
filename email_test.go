// Copyright (c) 2014-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package goemail

import (
	"bytes"
	"net/mail"
	"testing"
)

func TestNewMessageType(t *testing.T) {
	type args struct {
		from        string
		subject     string
		body        string
		contentType string
	}
	tests := []struct {
		name       string
		args       args
		wantName   string
		wantFrom   string
		wantNilMsg bool
	}{
		{
			name: "plain",
			args: args{
				from:        "boring@email.com",
				subject:     "boring plain email",
				body:        "nothing to see here",
				contentType: "text/plain",
			},
			wantName:   "",
			wantFrom:   "boring@email.com",
			wantNilMsg: false,
		},
		{
			name: "both",
			args: args{
				from:        "Boring Guy <boring@email.com>",
				subject:     "boring plain email",
				body:        "nothing to see here",
				contentType: "text/plain",
			},
			wantName:   "Boring Guy",
			wantFrom:   "boring@email.com",
			wantNilMsg: false,
		},
		{
			name: "invalid",
			args: args{
				from:        "boring-email.com",
				subject:     "boring plain email",
				body:        "nothing to see here",
				contentType: "text/plain",
			},
			wantName:   "",
			wantFrom:   "",
			wantNilMsg: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := NewMessageType(tt.args.from, tt.args.subject, tt.args.body, tt.args.contentType)
			if (msg == nil) != tt.wantNilMsg {
				t.Errorf("Message was not nil. Should have been invalid.")
			}
			if msg == nil {
				return
			}
			gotFrom := msg.From()
			if gotFrom != tt.wantFrom {
				t.Errorf("Incorrect From. Got %s, expected %s",
					gotFrom, tt.wantFrom)
			}
			gotName := msg.Name()
			if gotName != tt.wantName {
				t.Errorf("Incorrect Name. Got %s, expected %s",
					gotName, tt.wantName)
			}
			t.Logf(`Input: "%s". Name: "%s". From: "%s".`, tt.args.from,
				gotName, gotFrom)
		})
	}
}

func TestIsValidAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		isOk bool
	}{
		{"ok plain", "boring@email.com", true},
		{"ok named", "Boring Guy <boring@email.com>", true},
		{"bad", "boring-email.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidAddress(tt.addr); got != tt.isOk {
				t.Errorf("IsValidAddress() = %v, want %v", got, tt.isOk)
			}
		})
	}
}

func TestMessage_Body(t *testing.T) {
	body := `This is the body of the email.
	
Yay!`
	mailMsg := NewMessage("blah@example.com", "boring email", body)
	mailMsg.AddTo("someone@else.com")
	mailMsg.SetName("blah parson")

	tests := []struct {
		name string
		msg  *Message
	}{
		{"good", mailMsg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.msg.Body()
			t.Log(string(body))

			stdMsg, err := mail.ReadMessage(bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}

			t.Log(stdMsg.Header.Get("Content-Type"))
		})
	}
}
