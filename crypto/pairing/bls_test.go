package pairing

import (
	"testing"

	"github.com/simonlangowski/lightning1/crypto/pairing/kyber_wrap"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

// port of https://github.com/herumi/mcl/blob/v1.52/sample/bls_sig.cpp
func TestBLS(t *testing.T) {
	Q := kyber_wrap.G2Generator

	var secret mcl.Fr
	var public mcl.G2
	KeyGen(&secret, &public, &Q)

	message := []byte("Hello")

	var signature mcl.G1
	Sign(&signature, &secret, message)

	if !Verify(&signature, &Q, &public, message) {
		t.Fail()
	}
}

func BenchmarkBLSSign(b *testing.B) {
	Q := kyber_wrap.G2Generator

	var secret mcl.Fr
	var public mcl.G2
	KeyGen(&secret, &public, &Q)

	message := []byte("Hello")

	var signature mcl.G1
	for i := 0; i < b.N; i++ {
		Sign(&signature, &secret, message)
	}
}

func BenchmarkBLSVerify(b *testing.B) {
	Q := kyber_wrap.G2Generator

	var secret mcl.Fr
	var public mcl.G2
	KeyGen(&secret, &public, &Q)

	message := []byte("Hello")

	var signature mcl.G1
	Sign(&signature, &secret, message)

	for i := 0; i < b.N; i++ {
		if !Verify(&signature, &Q, &public, message) {
			b.FailNow()
		}
	}
}
