package checkpoint

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
)

var TOKEN_MESSAGE_LENGTH = token.TOKEN_SIZE + crypto.VERIFICATION_KEY_SIZE

const RESPONSE_LENGTH = crypto.KEY_SIZE + crypto.POINT_SIZE

func (c *CheckpointInfo) Len() int {
	return TOKEN_MESSAGE_LENGTH
}

func (c *CheckpointInfo) InterpretFrom(b []byte) error {
	if len(b) != c.Len() {
		return errors.LengthInvalidError()
	}
	pos := 0
	c.AnonymousVerificationKey.InterpretFrom(b[pos : pos+crypto.VERIFICATION_KEY_SIZE])
	pos += crypto.VERIFICATION_KEY_SIZE
	return c.Token.InterpretFrom(b[pos:])
}

func (c *CheckpointInfo) PackTo(b []byte) {
	if len(b) != c.Len() {
		panic(errors.LengthInvalidError())
	}
	pos := 0
	c.AnonymousVerificationKey.PackTo(b[pos : pos+crypto.VERIFICATION_KEY_SIZE])
	pos += crypto.VERIFICATION_KEY_SIZE
	c.Token.PackTo(b[pos:])
}

func (c *CheckpointInfo) Marshal() []byte {
	b := make([]byte, c.Len())
	c.PackTo(b)
	return b
}

func (c *CheckpointResponse) Len() int {
	return RESPONSE_LENGTH
}

func (c *CheckpointResponse) InterpretFrom(b []byte) error {
	if len(b) != c.Len() {
		return errors.LengthInvalidError()
	}
	copy(c.PublicKey[:], b[:crypto.KEY_SIZE])
	return c.PartialKey.InterpretFrom(b[crypto.KEY_SIZE:])
}

func (c *CheckpointResponse) PackTo(b []byte) {
	if len(b) != c.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:crypto.KEY_SIZE], c.PublicKey[:])
	c.PartialKey.PackTo(b[crypto.KEY_SIZE:])
}
