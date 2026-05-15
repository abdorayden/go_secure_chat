package protocol

import (
	"bytes"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	packet := Packet{
		Type:       TypeMessage,
		Username:   "alice",
		MessageID:  "msg-1",
		Ciphertext: "ciphertext",
		EncryptedKeys: map[string]string{
			"alice": "wrapped",
		},
	}

	var buf bytes.Buffer
	if err := Encode(&buf, packet); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	got, err := Decode(&buf)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got.Type != packet.Type || got.MessageID != packet.MessageID {
		t.Fatalf("decoded packet mismatch: %#v", got)
	}
}

func TestMalformedPacket(t *testing.T) {
	packet := Packet{Type: TypeMessage}
	var buf bytes.Buffer
	if err := Encode(&buf, packet); err == nil {
		t.Fatal("expected validation error")
	}
}
