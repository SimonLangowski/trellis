package token

import (
	"crypto/sha256"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"github.com/simonlangowski/lightning1/errors"
)

var TOKEN_SIZE = mcl.G1_LEN

type SignedToken struct {
	T mcl.G1
}

type TokenIssuanceInformation struct {
	key      *TokenPublicKey
	hash     mcl.G1
	blinding mcl.Fr
}

func (t *TokenPublicKey) Prepare(message []byte) (*mcl.G1, *TokenIssuanceInformation) {
	blindedHash := &mcl.G1{}
	info := &TokenIssuanceInformation{key: t}
	t.hashToCurvePoint(message, &info.hash)
	info.blinding.Random()
	t.blind(blindedHash, &info.hash, &info.blinding)
	return blindedHash, info
}

func (t *TokenSigningKey) BlindSign(out *mcl.G1, blindedHash *mcl.G1) error {
	// prevent subgroup leaking bits of key
	// https://eprint.iacr.org/2015/247.pdf
	// This is a scalar multiplication by the prime field order
	// See https://eprint.iacr.org/2019/814.pdf for faster methods
	if !config.SkipToken && !blindedHash.IsValidOrder() {
		return errors.BadElementError()
	}
	// Technicaly this scalar multiplication reuses the same base as the valid order check, so one could reuse the doublings
	mcl.G1Mul(out, blindedHash, &t.Share)
	return nil
}

func (info *TokenIssuanceInformation) Create(partials []mcl.G1) (*SignedToken, error) {
	token := &SignedToken{}
	t := info.key
	t.combine(partials, &token.T)
	t.unblind(&token.T, &info.blinding)
	if !t.verify(&token.T, &info.hash) {
		// Can check pairings of each message with public key shares to blame server
		return nil, errors.TokenInvalid()
	} else {
		info.key = nil
		return token, nil
	}
}

func (t *TokenPublicKey) VerifyMessage(token *SignedToken, message []byte) bool {
	var hash mcl.G1
	t.hashToCurvePoint(message, &hash)
	return t.verify(&token.T, &hash)
}

func (t *TokenPublicKey) hashToCurvePoint(message []byte, out *mcl.G1) {
	out.HashAndMapTo(message)
}

func (t *TokenPublicKey) blind(out *mcl.G1, m *mcl.G1, r *mcl.Fr) {
	mcl.G1Mul(out, m, r)
}

func (t *TokenPublicKey) combine(partials []mcl.G1, final *mcl.G1) {
	final.Clear()
	for i := range partials {
		mcl.G1Add(final, final, &partials[i])
	}
}

func (t *TokenPublicKey) unblind(m *mcl.G1, r *mcl.Fr) {
	mcl.FrInv(r, r)
	mcl.G1Mul(m, m, r)
}

func (t *TokenPublicKey) verify(signature *mcl.G1, hash *mcl.G1) bool {
	// Assume we hashed hash to G1 correctly, so it is in G1
	// Assume we checked the keys already
	// check signature order - See https://datatracker.ietf.org/doc/html/draft-boneh-bls-signature-00#section-3.2
	if !signature.IsValidOrder() {
		return false
	}
	// e(sign, Q) = e(hash, sQ)

	/*
		var e1, e2 mcl.GT
		mcl.Pairing(&e1, signature, &G2Generator) // e1 = e(sign, Q)
		mcl.Pairing(&e2, hash, &t.X)              // e2 = e(Hm, sQ)
		return e1.IsEqual(&e2)
	*/
	/*
		var e1, e2 mcl.GT
		G2GeneratorPrecompute.Pairing(&e1, signature)
		t.precompute.Pairing(&e2, hash)
		return e1.IsEqual(&e2)
	*/
	// https://hackmd.io/@benjaminion/bls12-381#Final-exponentiation
	return t.precompute.PrecomputedPairingCheck(signature, hash)
}

const HASH_SIZE = sha256.Size

func (t *SignedToken) Hash() [HASH_SIZE]byte {
	b := t.T.Serialize()
	return sha256.Sum256(b)
}
