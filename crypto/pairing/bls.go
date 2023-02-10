package pairing

import "github.com/simonlangowski/lightning1/crypto/pairing/mcl"

// Port of https://github.com/herumi/mcl/blob/v1.52/sample/bls_sig.cpp

func KeyGen(secret *mcl.Fr, public, base *mcl.G2) {
	secret.Random()
	mcl.G2Mul(public, base, secret) // pub = sQ
}

func Sign(signature *mcl.G1, secret *mcl.Fr, message []byte) {
	var Hm mcl.G1
	Hm.HashAndMapTo(message)
	mcl.G1Mul(signature, &Hm, secret) // sign = s H(m)
}

func Verify(signature *mcl.G1, base, public *mcl.G2, message []byte) bool {
	var e1, e2 mcl.GT
	var Hm mcl.G1
	Hm.HashAndMapTo(message)
	mcl.Pairing(&e1, signature, base) // e1 = e(sign, Q)
	mcl.Pairing(&e2, &Hm, public)     // e2 = e(Hm, sQ)
	return e1.IsEqual(&e2)
}
