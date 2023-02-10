package ec

import (
	"crypto/elliptic"
	"math/big"

	"github.com/simonlangowski/lightning1/errors"
)

var byteLen int
var ScalarElementSize int
var CurveElementSize int
var CURVE_MOD *big.Int

func init() {
	byteLen = (CURVE.Params().BitSize + 7) >> 3
	CurveElementSize = byteLen + 1
	CURVE_MOD = CURVE.Params().N
	ScalarElementSize = len(CURVE_MOD.Bytes())
}

func (p *Point) Len() int {
	return CurveElementSize
}

func (p *Point) PackTo(b []byte) {
	if len(b) != p.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:], p.Marshal()[:])
}

func (p *Point) InterpretFrom(b []byte) error {
	if len(b) != CurveElementSize {
		return errors.LengthInvalidError()
	}
	return p.Unmarshal(b)
}

func (s *ScalarElement) Len() int {
	return ScalarElementSize
}

func (s *ScalarElement) PackTo(b []byte) {
	if len(b) != ScalarElementSize {
		panic(errors.LengthInvalidError())
	}
	s.Int.FillBytes(b)
}

func (s *ScalarElement) Marshal() []byte {
	b := make([]byte, s.Len())
	s.PackTo(b)
	return b
}

func (s *ScalarElement) InterpretFrom(b []byte) error {
	if len(b) != s.Len() {
		return errors.LengthInvalidError()
	}
	s.Int = new(big.Int).SetBytes(b)
	if s.Int.Cmp(CURVE_MOD) >= 0 {
		return errors.BadElementError()
	}
	return nil
}

// Marshal calls through to elliptic.Marshal using the Curve field of the
// receiving Point. This produces a compressed marshaling as specified in
// SEC1 2.3.3.
func (p *Point) Marshal() []byte {
	return elliptic.MarshalCompressed(CURVE, p.X, p.Y)
}

// Unmarshal interprets SEC1 2.3.4 compressed points in addition to the raw
// points supported by elliptic.Unmarshal. It assumes a NIST curve, and
// specifically that a = -3. It's faster when p = 3 mod 4 because of how
// ModSqrt works.
func (p *Point) Unmarshal(data []byte) error {

	curve := CURVE

	fieldOrder := curve.Params().P
	// Compressed point
	x := new(big.Int).SetBytes(data[1 : 1+byteLen])
	if x.Cmp(fieldOrder) != -1 {
		// x in [0, p-1]
		return ErrInvalidPoint
	}
	if data[0] == 0x02 || data[0] == 0x03 {
		sign := data[0] & 1 // "mod 2"

		// Recall y² = x³ - 3x + b
		// obviously, the Lsh trick is only valid when a = -3
		x3 := new(big.Int).Mul(x, x)          // x^2
		x3.Mul(x3, x)                         // x(x^2)
		threeTimesX := new(big.Int).Lsh(x, 1) // x << 1 == x*2
		threeTimesX.Add(threeTimesX, x)       // (x << 1) + x == x*3
		x3.Sub(x3, threeTimesX)               // x^3 - 3x
		x3.Add(x3, curve.Params().B)          // x^3 - 3x + b
		y := x3.ModSqrt(x3, fieldOrder)       // sqrt(x^3 - 3x + b) (mod p)
		if y == nil {
			// if no square root exists, either marshaling error
			// or an invalid curve point
			return ErrInvalidPoint
		}
		if sign != isOdd(y) {
			y.Sub(fieldOrder, y)
		}
		if !curve.IsOnCurve(x, y) {
			x = nil
			y = nil
			return ErrInvalidPoint
		}
		p.X, p.Y = x, y
		return nil
	}
	return ErrInvalidPoint
}

func isOdd(x *big.Int) byte {
	return byte(x.Bit(0) & 1)
}

// BatchUnmarshalPoints takes a slice of P-256 curve points in the form specified
// in section 4.3.6 of ANSI X9.62 (see Go crypto/elliptic) and returns a slice
// of crypto.Point instances.
func BatchUnmarshalPoints(data [][]byte) ([]*Point, error) {
	decoded := make([]*Point, len(data))
	for i := 0; i < len(data); i++ {
		p := &Point{X: nil, Y: nil}
		err := p.Unmarshal(data[i])
		if err != nil {
			return nil, err
		}
		decoded[i] = p
	}
	return decoded, nil
}

// BatchMarshalPoints encodes a slice of crypto.Point objects in the form
// specified in section 4.3.6 of ANSI X9.62.
func BatchMarshalPoints(points []*Point) ([][]byte, error) {
	data := make([][]byte, len(points))
	for i := 0; i < len(points); i++ {
		data[i] = points[i].Marshal()
	}
	return data, nil
}
