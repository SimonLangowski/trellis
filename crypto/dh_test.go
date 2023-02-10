package crypto

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

// ed25519 is faster but doesn't support curve point addition out of the box
func BenchmarkGo(b *testing.B) {
	spk, ssk, _ := box.GenerateKey(rand.Reader)
	opk, osk, _ := box.GenerateKey(rand.Reader)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var s1, s2 [32]byte
		box.Precompute(&s1, opk, ssk)
		box.Precompute(&s2, spk, osk)
		if !bytes.Equal(s1[:], s2[:]) {
			b.Fail()
		}
	}
}

func Benchmark256(b *testing.B) {
	curve := elliptic.P256()
	modulus := curve.Params().N
	secret, _ := rand.Int(rand.Reader, modulus)
	otherSecret, _ := rand.Int(rand.Reader, modulus)
	publicX, publicY := curve.ScalarBaseMult(secret.Bytes())
	otherX, otherY := curve.ScalarBaseMult(otherSecret.Bytes())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1x, s1y := curve.ScalarMult(publicX, publicY, otherSecret.Bytes())
		s2x, s2y := curve.ScalarMult(otherX, otherY, secret.Bytes())
		if s1x.Cmp(s2x) != 0 || s1y.Cmp(s2y) != 0 {
			b.Fail()
		}
	}
}

func BenchmarkEc(b *testing.B) {
	ssk, spk := NewDHKeyPair()
	osk, opk := NewDHKeyPair()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s1 := ssk.SharedKey(&opk)
		s2 := osk.SharedKey(&spk)
		if !bytes.Equal(s1, s2) {
			b.Fail()
		}
	}
}

func BenchmarkSignature(b *testing.B) {
	data := make([]byte, 1000)
	pk, sk := NewSigningKeyPair()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signature := Sign(sk, data)
		if !Verify(pk, data, signature) {
			b.FailNow()
		}
	}
}

// func BenchmarkAuthDecryption(b *testing.B) {
// 	sk, pk := NewDHKeyPair()
// 	shared := sk.SharedKey(&pk)
// 	nonce := Nonce(0, 0, 0)
// 	message := make([]byte, 2000)
// 	cipher := AuthSecretSeal(message, &nonce, shared)
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		AuthSecretOpen(cipher, &nonce, shared)
// 	}
// }

func TestSignedDecryption(t *testing.T) {
	sk, pk := NewDHKeyPair()
	vk, k := NewSigningKeyPair()
	shared := sk.SharedKey(&pk)
	round := 5
	layer := 10
	server := 15
	nonce := Nonce(round, layer, server)
	message := make([]byte, 100)
	rand.Read(message)
	cipher := SignedSecretSeal(message, &nonce, shared, k)

	o := make([]byte, NONCE_SIZE+len(cipher))
	copy(o[NONCE_SIZE:], cipher)
	signed := PackSignedData(round, layer, server, o, NONCE_SIZE)
	signature := ReadSignature(cipher)
	if !Verify(vk, signed, signature) {
		t.Logf("Message not signed correctly")
		t.Fail()
	}
	decrypted := SecretOpen(cipher, &nonce, shared)

	if !bytes.Equal(message, decrypted) {
		t.Logf("Original message not recovered")
		t.Fail()
	}
}

func BenchmarkSignedDecryption(b *testing.B) {
	sk, pk := NewDHKeyPair()
	vk, k := NewSigningKeyPair()
	shared := sk.SharedKey(&pk)
	nonce := Nonce(0, 0, 0)
	message := make([]byte, 2000)
	cipher := SignedSecretSeal(message, &nonce, shared, k)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		o := make([]byte, NONCE_SIZE+len(cipher))
		copy(o[NONCE_SIZE:], cipher)
		signed := PackSignedData(0, 0, 0, o, NONCE_SIZE)
		signature := ReadSignature(cipher)
		if !Verify(vk, signed, signature) {
			b.Logf("Message not signed correctly")
			b.Fail()
		}
		SecretOpen(cipher, &nonce, shared)
	}
}

func TestKeyConversion(t *testing.T) {
	for i := 0; i < 100; i++ {
		cvk, csk := NewSigningKeyPair()
		spk, ssk := NewSigningKeyPair()

		cs, err := csk.ToScalar()
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		cp, err := cvk.ToCurvePoint()
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		ss, err := ssk.ToScalar()
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		sp, err := spk.ToCurvePoint()
		if err != nil {
			t.Log(err)
			t.FailNow()
		}
		sharedKey1 := cs.SharedKey(sp)
		sharedKey2 := ss.SharedKey(cp)
		if !bytes.Equal(sharedKey1, sharedKey2) {
			t.Logf("Shared keys %v and %v", sharedKey1, sharedKey2)
			t.FailNow()
		}
	}
}
