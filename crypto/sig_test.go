package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"testing"
)

// test expanded key verification
// benchmark speed difference

func TestExpandedKeys(t *testing.T) {
	spk, ssk := NewSigningKeyPair()
	epk, err := spk.ExpandKey()
	if err != nil {
		t.FailNow()
	}
	m := make([]byte, 1000)
	s := Sign(ssk, m)
	if !VerifyExpanded(epk, m, s) {
		t.Fail()
	}
}

func BenchmarkExpandedKeys(b *testing.B) {
	spk, ssk := NewSigningKeyPair()
	epk, err := spk.ExpandKey()
	if err != nil {
		b.FailNow()
	}
	m := make([]byte, 1000)
	s := Sign(ssk, m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !VerifyExpanded(epk, m, s) {
			b.FailNow()
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	spk, ssk := NewSigningKeyPair()
	m := make([]byte, 1000000)
	s := Sign(ssk, m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !Verify(spk, m, s) {
			b.FailNow()
		}
	}
}

func BenchmarkAES(b *testing.B) {
	sk, pk := NewDHKeyPair()
	shk := sk.SharedKey(&pk)
	m := make([]byte, 1000000)
	out := make([]byte, 1000000)
	block, err := aes.NewCipher(shk[:SymmetricKeySize])
	if err != nil {
		panic("Could not create new aes cipher")
	}
	iv := [NONCE_SIZE]byte{}
	stream := cipher.NewCTR(block, iv[:aes.BlockSize])
	stream.XORKeyStream(out, m)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SecretOpen(out, &iv, shk)
	}
}

// test prehash signing and verification
func TestPrehash(t *testing.T) {
	spk, ssk := NewSigningKeyPair()
	epk, err := spk.ExpandKey()
	if err != nil {
		t.FailNow()
	}
	m := make([]byte, 1000)
	s := PreHashSign(m, ssk)
	h := sha512.New()
	h.Write(m)
	if !PreHashVerify(h, epk, s) {
		t.Fail()
	}
}
