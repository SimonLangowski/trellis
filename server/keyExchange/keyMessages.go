package keyExchange

import (
	"encoding/binary"
	"math/big"

	"github.com/simonlangowski/lightning1/crypto/ec"
	"github.com/simonlangowski/lightning1/errors"
)

func SigningCurveScalarAdd(dest *big.Int, src *big.Int) *big.Int {
	dest = dest.Add(src, dest)
	dest = dest.Mod(dest, ec.CURVE_MOD)
	return dest
}

type PublicKeyMessage struct {
	SigningKey ec.Point
}

type PrivateKeyShareMessage struct {
	SecretSigningShare ec.ScalarElement
}

type InformationRequest struct {
	Round       uint32
	NumLayers   uint32
	SigningKeys []ec.Point
}

func (p *PublicKeyMessage) Len() int {
	return ec.CurveElementSize
}

func (p *PublicKeyMessage) PackTo(b []byte) {
	if len(b) != p.Len() {
		panic(errors.LengthInvalidError())
	}
	p.SigningKey.PackTo(b[:])
}

// first initialize arrays to correct size
func (p *PublicKeyMessage) InterpretFrom(b []byte) error {
	if len(b) != p.Len() {
		return errors.LengthInvalidError()
	}
	p.SigningKey.InterpretFrom(b[:])
	return nil
}

func NewPublicKeysMessage() *PublicKeyMessage {
	p := &PublicKeyMessage{}
	p.SigningKey.X = new(big.Int)
	p.SigningKey.Y = new(big.Int)
	return p
}

func (p *PrivateKeyShareMessage) Len() int {
	return ec.ScalarElementSize
}

func (p *PrivateKeyShareMessage) PackTo(b []byte) {
	p.SecretSigningShare.PackTo(b[:])
}

func (p *PrivateKeyShareMessage) InterpretFrom(b []byte) error {
	return p.SecretSigningShare.InterpretFrom(b[:])
}

func NewPrivateKeySharesMessage() *PrivateKeyShareMessage {
	p := &PrivateKeyShareMessage{}
	p.SecretSigningShare = *ec.NewScalarElement(new(big.Int))
	return p
}

func NewInformationRequest(b []byte) *InformationRequest {
	i := &InformationRequest{}
	pos := 0
	i.Round = binary.LittleEndian.Uint32(b[:4])
	pos += 4
	i.NumLayers = binary.LittleEndian.Uint32(b[pos : pos+4])
	pos += 4
	i.SigningKeys = make([]ec.Point, i.NumLayers)
	for j := range i.SigningKeys {
		i.SigningKeys[j].X = new(big.Int)
		i.SigningKeys[j].Y = new(big.Int)
		i.SigningKeys[j].InterpretFrom(b[pos : pos+ec.CurveElementSize])
		pos += ec.CurveElementSize
	}
	return i
}

func (i *InformationRequest) Len() int {
	return 8 + len(i.SigningKeys)*ec.CurveElementSize
}

func (i *InformationRequest) PackTo(b []byte) {
	if len(b) != i.Len() {
		panic(errors.LengthInvalidError())
	}
	pos := 0
	binary.LittleEndian.PutUint32(b[:4], i.Round)
	pos += 4
	binary.LittleEndian.PutUint32(b[pos:pos+4], i.NumLayers)
	pos += 4
	for j := range i.SigningKeys {
		i.SigningKeys[j].PackTo(b[pos : pos+ec.CurveElementSize])
		pos += ec.CurveElementSize
	}
}
