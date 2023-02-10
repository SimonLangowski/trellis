package token

import (
	"github.com/simonlangowski/lightning1/crypto/pairing"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

func (t *TokenPublicKey) Len() int {
	return mcl.G2_LEN
}

func (t *TokenPublicKey) PackTo(b []byte) {
	t.X.PackTo(b)
}

func (t *TokenPublicKey) InterpretFrom(b []byte) error {
	err := t.X.InterpretFrom(b)
	if err != nil {
		return err
	}
	t.precompute = pairing.NewPrecompute(&t.X)
	return nil
}

func (t *SignedToken) Len() int {
	return mcl.G1_LEN
}

func (t *SignedToken) PackTo(b []byte) {
	t.T.PackTo(b)
}

func (t *SignedToken) InterpretFrom(b []byte) error {
	return t.T.InterpretFrom(b)
}
