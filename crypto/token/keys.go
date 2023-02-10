package token

import (
	"github.com/simonlangowski/lightning1/crypto/pairing"
	"github.com/simonlangowski/lightning1/crypto/pairing/kyber_wrap"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

type TokenPublicKey struct {
	X          mcl.G2
	precompute *pairing.Precompute
}

type TokenSigningKey struct {
	X     mcl.G2
	Share mcl.Fr
}

func NewTokenSigningKey(share *mcl.Fr) *TokenSigningKey {
	t := &TokenSigningKey{
		Share: *share,
	}
	mcl.G2Mul(&t.X, &kyber_wrap.G2Generator, share)
	return t
}

func NewTokenPublicKey(key *mcl.G2) *TokenPublicKey {
	return &TokenPublicKey{
		X:          *key,
		precompute: pairing.NewPrecompute(key),
	}
}

func KeyGenShares(numShares int) ([]*TokenSigningKey, *TokenPublicKey, *TokenSigningKey) {
	s := &mcl.Fr{}
	s.Random()
	return MockKeyGen(numShares, s)
}

func MockKeyGen(numShares int, secret *mcl.Fr) ([]*TokenSigningKey, *TokenPublicKey, *TokenSigningKey) {
	masterSigningKey := NewTokenSigningKey(secret)
	shares := pairing.AdditiveShares(secret, numShares)
	partialSigningKeys := make([]*TokenSigningKey, numShares)
	publicKey := NewTokenPublicKey(&masterSigningKey.X)
	for i := range partialSigningKeys {
		partialSigningKeys[i] = NewTokenSigningKey(&shares[i])
	}
	return partialSigningKeys, publicKey, masterSigningKey
}
