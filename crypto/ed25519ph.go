package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/sha512"
	"hash"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"
	"github.com/simonlangowski/lightning1/errors"
)

// Implementation of the ed25519ph (pre-hash), or HashEdDSA algorithm
// see https://www.rfc-editor.org/rfc/rfc8032#section-4 for discussion
// Sign the hash of the message rather than the message

// Since each message is also signed by the user this signature
// Is only to blame the server and can be checked at the end of a layer

// const prefixWithFlag = "SigEd25519 no Ed25519 collisions\x01"

// func PreHashAndSign(m []byte, key SigningKey) Signature {
// 	h := sha512.New()
// 	h.Write(m)
// 	return PreHashSign(h, key)
// }

// func PreHashSign(h hash.Hash, key SigningKey) Signature {
// 	b := make([]byte, len(prefixWithFlag))
// 	copy(b, []byte(prefixWithFlag))
// 	b = h.Sum(b)
// 	return SignData(key, b)
// }

// func PreHashVerify(h hash.Hash, key VerificationKey, s Signature) bool {
// 	b := make([]byte, len(prefixWithFlag))
// 	copy(b, []byte(prefixWithFlag))
// 	b = h.Sum(b)
// 	return VerifyMessage(key, b, s)
// }

func PreHashSign(m []byte, key SigningKey) Signature {
	h := sha512.Sum512(m)
	s, err := ed25519.PrivateKey(key).Sign(rand.Reader, h[:], crypto.SHA512)
	if err != nil {
		panic(err)
	}
	errors.DebugPrint("Signature %v on %v %v with %v", s, m[:NONCE_SIZE], h, key)
	return s
}

func PreHashVerify(h hash.Hash, vk *ExpandedVerificationKey, s Signature) bool {
	op := &ed25519.Options{
		Hash: crypto.SHA512,
	}
	return ed25519.VerifyExpandedWithOptions((*ed25519.ExpandedPublicKey)(vk), h.Sum(nil), s, op)
}
