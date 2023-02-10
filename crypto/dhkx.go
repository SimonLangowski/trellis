package crypto

import (
	"crypto/rand"

	"filippo.io/edwards25519"
	"github.com/simonlangowski/lightning1/errors"
)

const KEY_SIZE = VERIFICATION_KEY_SIZE
const POINT_SIZE = 32
const SCALAR_SIZE = 32
const PK_SIZE = POINT_SIZE

type DHPublicKey struct {
	*edwards25519.Point
}
type DHPrivateKey struct {
	*edwards25519.Scalar
}
type DHSharedKey []byte

// This is a hint to help finding the public key for a message
type LookupKey [KEY_SIZE]byte

func (d *DHPrivateKey) SharedKey(pk *DHPublicKey) DHSharedKey {
	sharedKey := d.Mul(pk)
	return DHSharedKey(sharedKey.Bytes())
}

func (d *DHPrivateKey) Mul(pk *DHPublicKey) DHPublicKey {
	p := pk.Point
	s := d.Scalar
	sharedKey := edwards25519.NewIdentityPoint().ScalarMult(s, p)
	return DHPublicKey{sharedKey}
}

func NewDHKeyPair() (DHPrivateKey, DHPublicKey) {
	secret := RandomCurveScalar()
	public := secret.PublicKey()
	return *secret, *public
}

func RandomCurveScalar() *DHPrivateKey {
	s := edwards25519.NewScalar()
	randomBytes := make([]byte, 64)
	_, err := rand.Reader.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	s, err = s.SetUniformBytes(randomBytes)
	if err != nil {
		panic(err)
	}
	return &DHPrivateKey{Scalar: s}
}

// return the public key for a private key
func (d *DHPrivateKey) PublicKey() *DHPublicKey {
	s := d.Scalar
	p := edwards25519.NewIdentityPoint()
	p = p.ScalarBaseMult(s)
	return &DHPublicKey{p}
}

func (d *DHPublicKey) Len() int {
	return POINT_SIZE
}

func (d *DHPublicKey) PackTo(b []byte) {
	if len(b) != POINT_SIZE {
		panic(errors.LengthInvalidError)
	}
	copy(b, d.Bytes())
}

func (d *DHPublicKey) InterpretFrom(b []byte) error {
	if len(b) != POINT_SIZE {
		return errors.LengthInvalidError()
	}
	d.Point = &edwards25519.Point{}
	_, err := d.Point.SetBytes(b)
	return err
}

func (d *DHPrivateKey) InterpretFrom(b []byte) error {
	if len(b) != SCALAR_SIZE {
		return errors.LengthInvalidError()
	}
	d.Scalar = edwards25519.NewScalar()
	_, err := d.Scalar.SetCanonicalBytes(b)
	return err
}

// set d = d + o and return d
func (d *DHPublicKey) Accumulate(o *DHPublicKey) *DHPublicKey {
	d.Point = d.Point.Add(d.Point, o.Point)
	return d
}

func ZeroPoint() *DHPublicKey {
	return &DHPublicKey{
		Point: edwards25519.NewIdentityPoint(),
	}
}

func (d *DHPublicKey) AsShared() DHSharedKey {
	return d.Bytes()
}

func (d *DHPublicKey) Equals(o *DHPublicKey) bool {
	return d.Point.Equal(o.Point) == 1
}

func AdditiveShares(secret *DHPrivateKey, numShares int) []*DHPrivateKey {
	signingShares := make([]*DHPrivateKey, numShares)
	signingShares[0] = secret.Copy().Neg()
	for i := 1; i < len(signingShares); i++ {
		signingShares[i] = RandomCurveScalar()
		signingShares[0] = signingShares[0].Accumulate(signingShares[i])
	}
	signingShares[0].Neg()
	return signingShares
}

func (d *DHPrivateKey) Accumulate(o *DHPrivateKey) *DHPrivateKey {
	d.Scalar = d.Scalar.Add(d.Scalar, o.Scalar)
	return d
}

func (d *DHPrivateKey) Copy() *DHPrivateKey {
	return &DHPrivateKey{
		Scalar: edwards25519.NewScalar().Set(d.Scalar),
	}
}

func (d *DHPrivateKey) Neg() *DHPrivateKey {
	d.Scalar = d.Scalar.Negate(d.Scalar)
	return d
}
