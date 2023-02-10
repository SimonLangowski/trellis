package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/sha512"

	"github.com/oasisprotocol/curve25519-voi/primitives/ed25519"

	"filippo.io/edwards25519"
	"github.com/simonlangowski/lightning1/errors"
)

type VerificationKey ed25519.PublicKey
type SigningKey ed25519.PrivateKey
type Signature []byte

const SIGNATURE_SIZE = ed25519.SignatureSize
const VERIFICATION_KEY_SIZE = ed25519.PublicKeySize

/*
We use ed25519 for signing

- Each server signs all their messages under a ed25519 public key
- Each user signs their message for a token and the submission of the onion-encrypted ciphertext under a ed25519 public key

- Each user's blame message under a uniquely chosen key for each layer

*/

type ExpandedVerificationKey ed25519.ExpandedPublicKey

func SignData(key SigningKey, data []byte) []byte {
	return Sign(key, data)
}

func VerifyMessage(key VerificationKey, m []byte, s Signature) bool {
	return Verify(key, m, s)
}

func Sign(key SigningKey, message []byte) Signature {
	s := ed25519.Sign(ed25519.PrivateKey(key), message)
	errors.DebugPrint("Signature %v on %v with %v", s, message, key)
	return s
}

func BatchVerifier(size int) *ed25519.BatchVerifier {
	return ed25519.NewBatchVerifierWithCapacity(size)
}

func Verify(k VerificationKey, m []byte, s Signature) bool {
	return ed25519.Verify(ed25519.PublicKey(k), m, s)
}

func (k *VerificationKey) ExpandKey() (*ExpandedVerificationKey, error) {
	e, err := ed25519.NewExpandedPublicKey(ed25519.PublicKey(*k))
	return (*ExpandedVerificationKey)(e), err
}

func VerifyExpanded(v *ExpandedVerificationKey, m []byte, s Signature) bool {
	return ed25519.VerifyExpanded((*ed25519.ExpandedPublicKey)(v), m, s)
}

func NewSigningKeyPair() (VerificationKey, SigningKey) {
	psk, ssk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return VerificationKey(psk), SigningKey(ssk)
}

func (p *VerificationKey) Len() int {
	return VERIFICATION_KEY_SIZE
}
func (p *VerificationKey) PackTo(b []byte) {
	if len(b) != p.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:], (*p)[:])
}
func (p *VerificationKey) InterpretFrom(b []byte) error {
	if len(b) != p.Len() {
		return errors.LengthInvalidError()
	}
	*p = (VerificationKey)(b)
	return nil
}

func (p *VerificationKey) Bytes() []byte {
	return *p
}

func (p *VerificationKey) Copy() VerificationKey {
	b := make([]byte, VERIFICATION_KEY_SIZE)
	copy(b, p.Bytes())
	return VerificationKey(b)
}

func (p *VerificationKey) PublicKey() ed25519.PublicKey {
	return ed25519.PublicKey(*p)
}

func (p *SigningKey) ToScalar() (*DHPrivateKey, error) {
	privateKey := (*ed25519.PrivateKey)(p)
	h := sha512.Sum512(privateKey.Seed())
	s, err := edwards25519.NewScalar().SetBytesWithClamping(h[:32])
	return &DHPrivateKey{s}, err
}

func (p *SigningKey) PublicKey() crypto.PublicKey {
	privateKey := (*ed25519.PrivateKey)(p)
	return privateKey.Public()
}

func (p *VerificationKey) ToCurvePoint() (*DHPublicKey, error) {
	pk := (*ed25519.PublicKey)(p)
	pt, err := edwards25519.NewIdentityPoint().SetBytes(*pk)
	return &DHPublicKey{pt}, err
}

func (d *VerificationKey) LookupKey() LookupKey {
	k := LookupKey{}
	copy(k[:], []byte(*d))
	return k
}

func (s *Signature) InterpretFrom(b []byte) error {
	if len(b) != SIGNATURE_SIZE {
		return errors.LengthInvalidError()
	}
	b2 := make([]byte, SIGNATURE_SIZE)
	copy(b2, b)
	*s = b2
	return nil
}
