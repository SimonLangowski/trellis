// from privacypass/challenge-bypass-server

// An implementation of the widely-used (honest-verifier) NIZK proof of
// discrete logarithm equality originally described in the Chaum and Pedersen
// paper "Wallet Databases with Observers", using Go's standard crypto/elliptic
// package.
//
// This implementation potentially minimizes the amount of data that needs to
// be sent to the verifier by including the intermediate proof values (called
// a, b in the paper) in the Fiat-Shamir hash step and using hash comparison to
// determine proof validity instead of group element equality.
package nizk

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/ec"
)

var (
	ErrInconsistentCurves = errors.New("points are on different curves")
)

type DLEQProof struct {
	R    *ec.ScalarElement // response value
	C    *ec.ScalarElement // hash of intermediate proof values to streamline equality checks
	Hash []byte            // resulting proof hash bytes

}

func NewDecryptionProof(clientPublicKey crypto.DHPublicKey, serverPublicKey crypto.DHPublicKey, sharedKey crypto.DHSharedKey) (*DLEQProof, error) {

	return nil, nil
}

// Given g, h, m, z such that g, m are generators and h = g^x, z = m^x,
// compute a proof that log_g(h) == log_m(z). If (g, h, m, z) are already known
// to the verifier, then (c, r) is sufficient to check the proof.
func NewProof(G, H, M, Z *ec.Point, x *ec.ScalarElement) (*DLEQProof, error) {

	// s is a random element of Z/qZ
	s, err := ec.RandomCurveScalar(crand.Reader)
	if err != nil {
		return nil, err
	}

	// (a, b) = (g^s, m^s)
	A := G.Copy()
	A.ScalarMult(s)
	B := M.Copy()
	B.ScalarMult(s)

	// c = H(g, h, z, a, b)
	// Note: in the paper this is H(m, z, a, b) to constitute a signature over
	// m and prevent existential forgery. What we care about here isn't
	// committing to a particular m but the equality with the specific public
	// key h.
	h := sha256.New()
	h.Write(G.Marshal())
	h.Write(H.Marshal())
	h.Write(M.Marshal())
	h.Write(Z.Marshal())
	h.Write(A.Marshal())
	h.Write(B.Marshal())
	cBytes := h.Sum(nil)

	// Expressing this as r = s - cx instead of r = s + cx saves us an
	// inversion of c when calculating A and B on the verification side.
	c := new(big.Int).SetBytes(cBytes)
	c.Mod(c, ec.CURVE.Params().N) // c = c (mod q)
	r := new(big.Int).Neg(c)      // r = -c
	r.Mul(r, x.Int)               // r = -cx
	r.Add(r, s.Int)               // r = s - cx
	r.Mod(r, ec.CURVE.Params().N) // r = r (mod q)

	proof := &DLEQProof{
		R:    ec.NewScalarElement(r),
		C:    ec.NewScalarElement(c),
		Hash: cBytes,
	}
	return proof, nil
}

func (pr *DLEQProof) Verify(G, H, M, Z *ec.Point) bool {

	// Prover gave us c = H(h, z, a, b)
	// Calculate rG and rM, then C' = H(h, z, rG + cH, rM + cZ).
	// C == C' is equivalent to checking the equalities.

	// a = (g^r)(h^c)
	// A = rG + cH
	cH := H.Copy()
	cH.ScalarMult(pr.C)

	rG := G.Copy()
	rG.ScalarMult(pr.R)

	A := cH.Accumulate(rG)

	// b = (m^r)(z^c)
	// B = rM + cZ
	cZ := Z.Copy()
	cZ.ScalarMult(pr.C)

	rM := M.Copy()
	rM.ScalarMult(pr.R)

	B := cZ.Accumulate(rM)

	// C' = H(g, h, z, a, b) == C
	h := sha256.New()
	h.Write(G.Marshal())
	h.Write(H.Marshal())
	h.Write(M.Marshal())
	h.Write(Z.Marshal())
	h.Write(A.Marshal())
	h.Write(B.Marshal())
	c := h.Sum(nil)

	return hmac.Equal(pr.Hash, c)
}
