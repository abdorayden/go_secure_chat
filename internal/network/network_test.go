package network

import (
	"bufio"
	"bytes"
	"testing"

	appcrypto "chat_sec/internal/crypto"
	"chat_sec/internal/protocol"
)

func TestReadEncryptedPacketRejectsMalformedPayload(t *testing.T) {
	key, err := appcrypto.GenerateAESKey(32)
	if err != nil {
		t.Fatalf("GenerateAESKey: %v", err)
	}

	var buf bytes.Buffer
	if err := protocol.Encode(&buf, protocol.Packet{
		Type:    protocol.TypeSystem,
		Status:  "transport",
		Payload: "not-base64",
	}); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	if _, err := readEncryptedPacket(bufio.NewReader(&buf), key); err == nil {
		t.Fatal("expected error for malformed encrypted payload")
	}
}
