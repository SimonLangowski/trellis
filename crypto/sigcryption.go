package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"

	"github.com/simonlangowski/lightning1/errors"
)

// derived from box.SecretSeal and box.SecretOpen
// perform encrypt then sign non-repudiable encryption

const NONCE_SIZE = 24
const SignedMetadataSize = 12
const SymmetricKeySize = 16
const Overhead = SIGNATURE_SIZE

// ID of receiving party must be signed
func Nonce(round, layer, destId int) [NONCE_SIZE]byte {
	var nonce [NONCE_SIZE]byte
	binary.LittleEndian.PutUint64(nonce[:8], uint64(round))
	binary.LittleEndian.PutUint64(nonce[8:16], uint64(layer))
	binary.LittleEndian.PutUint64(nonce[16:], uint64(destId))
	return nonce
}

// return the signature from a signed message
func ReadSignature(box []byte) Signature {
	return box[len(box)-Overhead:]
}

// write the additional signed metadata before the message
func PackSignedData(round, layer, server int, raw []byte, offset int) []byte {
	if offset < NONCE_SIZE {
		panic(errors.UnimplementedError())
	}
	binary.LittleEndian.PutUint64(raw[offset-24:offset-16], uint64(round))
	binary.LittleEndian.PutUint64(raw[offset-16:offset-8], uint64(layer))
	binary.LittleEndian.PutUint64(raw[offset-8:offset], uint64(server))
	return raw[offset-24 : len(raw)-Overhead]
}

// AES encryption with the symmetric key
// Encrypt-then-sign non repudiable encryption
func SignedSecretSeal(message []byte, nonce *[NONCE_SIZE]byte, key DHSharedKey, signingKey SigningKey) []byte {
	// keys := hkdf.New(sha256.New, key, (*nonce)[:], nil)
	// aesKey := make([]byte, SymmetricKeySize)

	// n, err := keys.Read(aesKey)
	// if err != nil {
	// 	panic("Could not generate aes key")
	// } else if n != len(aesKey) {
	// 	panic("Could not read enough bytes for aes key")
	// }

	out := make([]byte, len(message)+Overhead+NONCE_SIZE)

	iv := (*nonce)[:aes.BlockSize]

	block, err := aes.NewCipher(key[:SymmetricKeySize])
	if err != nil {
		panic("Could not create new aes cipher")
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(out[NONCE_SIZE:len(out)-Overhead], message)

	// Sign nonce and ciphertext!
	copy(out[:NONCE_SIZE], nonce[:])
	s := Sign(signingKey, out[:len(out)-Overhead])
	copy(out[len(out)-Overhead:], s)
	// Nonce here can be filled in by the verifying machine, and not included
	errors.DebugPrint("Encrypted %v %v: %v to %v", nonce, key, message, out[NONCE_SIZE:len(out)-Overhead])
	return out[NONCE_SIZE:]
}

func SecretOpen(box []byte, nonce *[24]byte, key DHSharedKey) []byte {
	// keys := hkdf.New(sha256.New, key, (*nonce)[:], nil)
	// aesKey := [SymmetricKeySize]byte{}

	// n, err := keys.Read(aesKey[:])
	// if err != nil {
	// 	panic("Could not generate aes key")
	// } else if n != len(aesKey) {
	// 	panic("Could not read enough bytes for aes key")
	// }

	iv := (*nonce)[:aes.BlockSize]
	out := make([]byte, len(box)-Overhead)

	block, err := aes.NewCipher(key[:SymmetricKeySize])
	if err != nil {
		panic("Could not create new aes cipher")
	}

	stream := cipher.NewCTR(block, iv[:])
	stream.XORKeyStream(out, box[:len(box)-Overhead])

	errors.DebugPrint("Decrypted %v %v: %v from %v", nonce, key, out, box[:len(box)-Overhead])
	return out
}
