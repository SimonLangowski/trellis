package nizk

import (
	"bytes"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/suites"
)

// This needs to be over the ed25519 curve
// proof of ownership of DHPublicKey

// If abstract, also used to prove commit of secret is g^s in VSS

// See Figure 2.5 of
// https://core.ac.uk/download/pdf/144147582.pdf

type DLProof struct {
	Value kyber.Point
	V     kyber.Scalar
	Hash  []byte
}

// assuming with respect to base generator for CURVE
// Value = G^(witness)
func NewDL(suite suites.Suite, witness kyber.Scalar, value kyber.Point) *DLProof {
	r := suite.Scalar().Pick(suite.RandomStream())
	// hash = Hash(Value || g^r)
	h := suite.Hash()
	commit := suite.Point().Mul(r, nil)
	b, _ := value.MarshalBinary()
	h.Write(b)
	b, _ = commit.MarshalBinary()
	h.Write(b)
	hash := h.Sum(nil)
	c := suite.Scalar().Pick(suite.XOF(hash))
	// v = -cx + r
	v := c.Mul(c, witness)
	v = v.Add(v, r)
	return &DLProof{
		Value: value,
		V:     v,
		Hash:  hash,
	}
}

func (p *DLProof) Verify(suite suites.Suite) bool {
	c := suite.Scalar().Pick(suite.XOF(p.Hash))
	// hash = Hash(Value || g^v * Value^c)
	commit := suite.Point().Mul(p.V, nil)
	commit = commit.Add(commit, suite.Point().Mul(c, p.Value))
	h := suite.Hash()
	b, _ := p.Value.MarshalBinary()
	h.Write(b)
	b, _ = commit.MarshalBinary()
	h.Write(b)
	hash := h.Sum(nil)
	return bytes.Equal(hash, p.Hash)
}
