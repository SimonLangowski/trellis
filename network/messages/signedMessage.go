package messages

import (
	"encoding/binary"

	"github.com/simonlangowski/lightning1/crypto"
)

type Metadata struct {
	Type        NetworkMessage_MessageType
	Round       int
	Layer       int
	Sender      int
	Dest        int
	Group       int32
	NumMessages uint32
}

type SignedMessage struct {
	Metadata
	Raw       []byte
	Data      []byte
	Signature []byte
}

// we also sign the chunk number and chunk length, but these can vary between chunks
const Metadata_size = 4 * 7 // 7 uint32 of metadata

// Create a message with fields for metadata
func NewSignedMessage(dataLen, round, layer, sender, group, dest, numMessages int, t NetworkMessage_MessageType) *SignedMessage {
	m := &SignedMessage{
		Raw: make([]byte, dataLen+Metadata_size+crypto.SIGNATURE_SIZE),
		Metadata: Metadata{
			Type:        t,
			Round:       round,
			Layer:       layer,
			Sender:      sender,
			Dest:        dest,
			Group:       int32(group),
			NumMessages: uint32(numMessages),
		},
	}
	m.Data = m.Raw[Metadata_size : dataLen+Metadata_size]
	return m
}

func (m *Metadata) InterpretFrom(b []byte) {
	m.NumMessages = binary.LittleEndian.Uint32(b[0:4])
	m.Type = NetworkMessage_MessageType(binary.LittleEndian.Uint32(b[4:8]))
	m.Round = int(binary.LittleEndian.Uint32(b[8:12]))
	m.Layer = int(binary.LittleEndian.Uint32(b[12:16]))
	m.Sender = int(binary.LittleEndian.Uint32(b[16:20]))
	m.Dest = int(binary.LittleEndian.Uint32(b[20:24]))
	m.Group = int32(binary.LittleEndian.Uint32(b[24:28]))
}

// Parse a message's metadata fields
func ParseSignedMessage(raw *NetworkMessage) *SignedMessage {
	if len(raw.Data) < Metadata_size {
		return nil
	}
	m := &SignedMessage{
		Raw:       raw.Data,
		Data:      raw.Data[Metadata_size:],
		Signature: raw.Signature,
	}
	m.Metadata.InterpretFrom(m.Raw)
	return m
}

func (s *Metadata) PackTo(b []byte) {
	binary.LittleEndian.PutUint32(b[0:4], s.NumMessages)
	binary.LittleEndian.PutUint32(b[4:8], uint32(s.Type))
	binary.LittleEndian.PutUint32(b[8:12], uint32(s.Round))
	binary.LittleEndian.PutUint32(b[12:16], uint32(s.Layer))
	binary.LittleEndian.PutUint32(b[16:20], uint32(s.Sender))
	binary.LittleEndian.PutUint32(b[20:24], uint32(s.Dest))
	binary.LittleEndian.PutUint32(b[24:28], uint32(s.Group))
}

// Get the byte array that is signed by the signature (including the metadata)
func (s *SignedMessage) GetSignedData() []byte {
	s.Metadata.PackTo(s.Raw)
	return s.Raw[:Metadata_size+len(s.Data)]
}

func (s *SignedMessage) AsArray() []byte {
	copy(s.Raw[len(s.Data)+Metadata_size:], s.Signature)
	return s.Raw[:len(s.Data)+Metadata_size+crypto.SIGNATURE_SIZE]
}

func (s *SignedMessage) AsNetworkMessage() *NetworkMessage {
	return &NetworkMessage{
		MessageType: s.Type,
		Data:        s.Raw[:Metadata_size+len(s.Data)],
		Signature:   s.Signature,
	}
}
