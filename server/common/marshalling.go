package common

import (
	"encoding/binary"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
)

func (l *LightningEnvelope) Len() int {
	return crypto.KEY_SIZE + len(l.SignedCiphertext)
}

func (l *LightningEnvelope) InterpretFrom(b []byte) error {
	// length has to be checked in advance because it is a function of the output message size
	copy(l.Key[:], b[:crypto.KEY_SIZE])
	l.SignedCiphertext = b[crypto.KEY_SIZE:]
	l.raw = b
	return nil
}

func (l *LightningEnvelope) PackTo(b []byte) {
	if len(b) != l.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:crypto.KEY_SIZE], l.Key[:])
	copy(b[crypto.KEY_SIZE:], l.SignedCiphertext)
}

func (l *LightningEnvelope) Marshal() []byte {
	b := make([]byte, l.Len())
	l.PackTo(b)
	return b
}

func (l *LightningEnvelope) GetSignedData(round, layer, server int) []byte {
	// note that lookup key was copied out, so we can overwrite the data
	return crypto.PackSignedData(round, layer, server, l.raw, crypto.KEY_SIZE)
}

func (l *LightningEnvelope) GetSignature() crypto.Signature {
	return crypto.ReadSignature(l.SignedCiphertext)
}

func (p *PathEstablishmentEnvelope) Len() int {
	return crypto.POINT_SIZE + token.TOKEN_SIZE + len(p.SignedCiphertext)
}

func (p *PathEstablishmentEnvelope) InterpretFrom(b []byte) error {
	pos := 0
	err := p.InKey.InterpretFrom(b[:crypto.POINT_SIZE])
	pos += crypto.POINT_SIZE
	p.InToken.InterpretFrom(b[pos : pos+token.TOKEN_SIZE])
	if err != nil {
		return err
	}
	pos += token.TOKEN_SIZE
	p.SignedCiphertext = b[pos:]
	p.raw = b
	return err
}

func (p *PathEstablishmentEnvelope) PackTo(b []byte) {
	pos := 0
	p.InKey.PackTo(b[:crypto.POINT_SIZE])
	pos += crypto.POINT_SIZE
	p.InToken.PackTo(b[pos : pos+token.TOKEN_SIZE])
	pos += token.TOKEN_SIZE
	copy(b[pos:], p.SignedCiphertext)
}

func (p *PathEstablishmentEnvelope) Marshal() []byte {
	b := make([]byte, p.Len())
	p.PackTo(b)
	return b
}

func (p *PathEstablishmentEnvelope) GetSignedData(round, layer, server int) []byte {
	// note that InToken was parsed into a curve element
	return crypto.PackSignedData(round, layer, server, p.raw, crypto.POINT_SIZE+token.TOKEN_SIZE)
}

func (p *PathEstablishmentEnvelope) ReadSignature() crypto.Signature {
	return crypto.ReadSignature(p.SignedCiphertext)
}

const FINAL_MESSAGE_BASE_LENGTH = crypto.SIGNATURE_SIZE

// pack the bytes that are signed by a token
func TokenContent(key crypto.VerificationKey, round, layer, server int) []byte {
	length := 4*3 + crypto.POINT_SIZE
	b := make([]byte, length)
	binary.LittleEndian.PutUint32(b[0:4], uint32(round))
	binary.LittleEndian.PutUint32(b[4:8], uint32(layer))
	binary.LittleEndian.PutUint32(b[8:12], uint32(server))
	key.PackTo(b[12 : 12+crypto.POINT_SIZE])
	return b
}

func (p *PathEstablishmentInfo) Marshal() []byte {
	l := token.TOKEN_SIZE + crypto.POINT_SIZE + len(p.BoomerangEnvelope) + len(p.NextEnvelope)
	b := make([]byte, l)
	pos := 0
	p.OutKey.PackTo(b[:crypto.POINT_SIZE])
	pos += crypto.POINT_SIZE
	p.OutToken.PackTo(b[pos : pos+token.TOKEN_SIZE])
	pos += token.TOKEN_SIZE
	copy(b[pos:], p.BoomerangEnvelope)
	pos += len(p.BoomerangEnvelope)
	copy(b[pos:], p.NextEnvelope)
	return b
}

func (p *PathEstablishmentInfo) InterpretFrom(b []byte, boomerangLength int) error {
	// length has to be checked in advance because it is a function of the number of layers
	p.raw = b
	pos := 0
	err := p.OutKey.InterpretFrom(b[:crypto.POINT_SIZE])
	if err != nil {
		return err
	}
	pos += crypto.POINT_SIZE
	err = p.OutToken.InterpretFrom(b[pos : pos+token.TOKEN_SIZE])
	if err != nil {
		return err
	}
	pos += token.TOKEN_SIZE
	p.BoomerangEnvelope = b[pos : pos+boomerangLength]
	pos += boomerangLength
	p.NextEnvelope = b[pos:]
	return nil
}

func (p *PathEstablishmentInfo) GetSignedData(round, layer, server int) []byte {
	// note that InToken was parsed into a curve element
	return crypto.PackSignedData(round, layer, server, p.raw, crypto.POINT_SIZE+token.TOKEN_SIZE)
}

func (p *PathEstablishmentInfo) GetSignature() crypto.Signature {
	return crypto.ReadSignature(p.BoomerangEnvelope)
}

func (l *FinalLightningMessage) Len() int {
	return crypto.VERIFICATION_KEY_SIZE + crypto.SIGNATURE_SIZE + len(l.Message)
}

func (l *FinalLightningMessage) PackTo(b []byte) {
	if len(b) != l.Len() {
		panic(errors.LengthInvalidError())
	}
	pos := 0
	l.AnonymousVerificationKey.PackTo(b[pos : pos+crypto.VERIFICATION_KEY_SIZE])
	pos += crypto.VERIFICATION_KEY_SIZE
	copy(b[pos:pos+crypto.SIGNATURE_SIZE], l.Signature)
	pos += crypto.SIGNATURE_SIZE
	copy(b[pos:], l.Message)
}

func (l *FinalLightningMessage) InterpretFrom(b []byte) error {
	if len(b) < FINAL_MESSAGE_BASE_LENGTH {
		return errors.LengthInvalidError()
	}
	pos := 0
	err := l.AnonymousVerificationKey.InterpretFrom(b[pos : pos+crypto.VERIFICATION_KEY_SIZE])
	pos += crypto.VERIFICATION_KEY_SIZE
	if err != nil {
		return err
	}
	l.Signature = b[pos : pos+crypto.SIGNATURE_SIZE]
	pos += crypto.SIGNATURE_SIZE
	l.Message = b[pos:]
	return nil
}

// Marshal fields for signing
func (l *FinalLightningMessage) MarshalSigned() []byte {
	return l.Message
}

// Marshal for inclusion in next message
func (l *FinalLightningMessage) MarshalI() []byte {
	b := make([]byte, crypto.SIGNATURE_SIZE+len(l.Message))
	pos := 0
	copy(b[pos:pos+crypto.SIGNATURE_SIZE], l.Signature)
	pos += crypto.SIGNATURE_SIZE
	copy(b[pos:], l.Message)
	return b
}

func (l *FinalLightningMessage) Marshal() []byte {
	b := make([]byte, l.Len())
	l.PackTo(b)
	return b
}
