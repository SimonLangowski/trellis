package token

import (
	"os"
	"runtime/pprof"
	"testing"

	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

func TestToken(t *testing.T) {
	numSigners := 5
	partialSigningKeys, publicKey, _ := KeyGenShares(numSigners)
	message := []byte("Hi")
	blindedHash, info := publicKey.Prepare(message)
	blindedHashes := make([]mcl.G1, numSigners)
	for i := range partialSigningKeys {
		err := partialSigningKeys[i].BlindSign(&blindedHashes[i], blindedHash)
		if err != nil {
			t.Logf("%v", err)
			t.FailNow()
		}
	}

	token, err := info.Create(blindedHashes)
	if err != nil {
		t.Logf("%v", err)
		t.FailNow()
	}

	if !publicKey.VerifyMessage(token, message) {
		t.FailNow()
	}
}

func TestProfile(t *testing.T) {
	f, _ := os.Create("token.pprof")
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	for q := 0; q < 10; q++ {
		numSigners := 10 // anytrust size
		partialSigningKeys, publicKey, _ := KeyGenShares(numSigners)

		message := []byte("Hi")
		blindedHash, info := publicKey.Prepare(message)
		blindedHashes := make([]mcl.G1, numSigners)
		for i := range partialSigningKeys {
			err := partialSigningKeys[i].BlindSign(&blindedHashes[i], blindedHash)
			if err != nil {
				t.Logf("%v", err)
				t.FailNow()
			}
		}

		token, err := info.Create(blindedHashes)
		if err != nil {
			t.Logf("%v", err)
			t.FailNow()
		}

		for range partialSigningKeys {
			// each anytrust group member checks
			if !publicKey.VerifyMessage(token, message) {
				t.FailNow()
			}
		}
	}
}

func BenchmarkTokenSign(b *testing.B) {
	numSigners := 1
	partialSigningKeys, publicKey, _ := KeyGenShares(numSigners)

	message := []byte("Hi")
	blindedHash, _ := publicKey.Prepare(message)
	blindedHashes := make([]mcl.G1, numSigners)
	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		for i := range partialSigningKeys {
			err := partialSigningKeys[i].BlindSign(&blindedHashes[i], blindedHash)
			if err != nil {
				b.Logf("%v", err)
				b.FailNow()
			}
		}
	}
}

func BenchmarkTokenVerify(b *testing.B) {
	_, publicKey, signingKey := KeyGenShares(1)
	message := []byte("Hi")
	blindedHash, info := publicKey.Prepare(message)
	signingKey.BlindSign(blindedHash, blindedHash)
	blindedHashes := []mcl.G1{*blindedHash}
	token, err := info.Create(blindedHashes)
	if err != nil {
		b.FailNow()
	}
	b.ResetTimer()
	for j := 0; j < b.N; j++ {
		publicKey.VerifyMessage(token, message)
	}
}

func BenchmarkKeyGen(b *testing.B) {
	for j := 0; j < b.N; j++ {
		KeyGenShares(64)
	}
}
