package ec

import (
	"crypto/rand"
	"io"
	"math/big"
)

var zero *big.Int

func init() {
	zero = new(big.Int)
}

type ScalarElement struct {
	Int *big.Int
}

func (el *ScalarElement) Accumulate(elems ...*ScalarElement) {
	for _, other := range elems {
		el.Int = el.Int.Add(el.Int, other.Int)
	}

	el.Int = el.Int.Mod(el.Int, CURVE_MOD)
}

func NewScalarElement(el *big.Int) *ScalarElement {
	return &ScalarElement{el}
}

func (el *ScalarElement) ToBytes() []byte {
	return el.Int.Bytes()
}

func (el *ScalarElement) FromBytes(b []byte) {
	el.Int.SetBytes(b)
}

func (el *ScalarElement) Copy() *ScalarElement {
	return &ScalarElement{new(big.Int).Set(el.Int)}
}

func ZeroScalar() *ScalarElement {
	return &ScalarElement{new(big.Int)}
}

func (el *ScalarElement) Inverse() *ScalarElement {
	el.Int.ModInverse(el.Int, CURVE.Params().N)
	return el
}

func (el *ScalarElement) Neg() *ScalarElement {
	if el.Int.Cmp(zero) == 0 {
		return el
	}
	el.Int = el.Int.Sub(CURVE_MOD, el.Int)
	return el
}

// This is just a bitmask with the number of ones starting at 8 then
// incrementing by index. To account for fields with bitsizes that are not a whole
// number of bytes, we mask off the unnecessary bits. h/t agl
var mask = []byte{0xff, 0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f}

func RandomCurveScalar(rand io.Reader) (*ScalarElement, error) {
	N := CURVE.Params().N // base point subgroup order
	bitLen := N.BitLen()
	byteLen := (bitLen + 7) >> 3
	buf := make([]byte, byteLen)

	// When in doubt, do what agl does in elliptic.go. Presumably
	// new(big.Int).SetBytes(b).Mod(N) would introduce bias, so we're sampling.
	for {
		_, err := io.ReadFull(rand, buf)
		if err != nil {
			return nil, err
		}
		// Mask to account for field sizes that are not a whole number of bytes.
		buf[0] &= mask[bitLen%8]
		// Check if scalar is in the correct range.
		if new(big.Int).SetBytes(buf).Cmp(N) >= 0 {
			continue
		}
		break
	}

	return NewScalarElement(new(big.Int).SetBytes(buf)), nil
}

func AdditiveShares(secret *ScalarElement, numShares int) []*ScalarElement {
	signingShares := make([]*ScalarElement, numShares)
	for i := 1; i < len(signingShares); i++ {
		signingShares[i], _ = RandomCurveScalar(rand.Reader)
	}
	signingShares[0] = secret.Copy().Neg()
	signingShares[0].Accumulate(signingShares[1:]...)
	signingShares[0].Neg()
	return signingShares
}
