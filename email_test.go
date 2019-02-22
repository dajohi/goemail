// Copyright (c) 2014-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package goemail

import (
	"bytes"
	"net/mail"
	"testing"
)

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
