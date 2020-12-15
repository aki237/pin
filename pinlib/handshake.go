package pinlib

import (
	"fmt"
	"io"
)

type MessageType byte

const (
	IPRequest MessageType = 1 // no payload
	Accept    MessageType = 3 // no payload
	Deny      MessageType = 5 // no payload

	IPResponse        MessageType = 2 // 9 bytes => ip (4) + prefix (1) + gateway ip (4)
	NoIPError         MessageType = 4 // no payload
	AcknowledgeAccept MessageType = 6 // motd (till \0 character)

	last = 7
)

// HandshakeMessage contains fields required for
// exchanging handshake details between the client and the server
type HandshakeMessage struct {
	Type    MessageType
	Payload [128]byte
}

// HandshakeMessage converts the handshake message to raw bytes
func (hm HandshakeMessage) ToBytes() []byte {
	return append([]byte{byte(hm.Type)}, hm.Payload[:]...)
}

// HandshakeMessage converts the raw bytes to handshake message
func (hm *HandshakeMessage) FromBytes(p []byte) error {
	if len(p) != 129 {
		return fmt.Errorf("message size mismatch: %d != 129", len(p))
	}

	if p[0] > last || p[0] < 1 {
		return fmt.Errorf("unable to decode message type: %b", p[0])
	}
	hm.Type = MessageType(p[0])
	copy(hm.Payload[:], p[1:])

	return nil
}

func getMessage(rd io.Reader) (*HandshakeMessage, error) {
	buf := make([]byte, 129)
	_, err := rd.Read(buf)
	if err != nil {
		return nil, err
	}

	hm := &HandshakeMessage{}
	return hm, hm.FromBytes(buf)
}
