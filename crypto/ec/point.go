package ec

import (
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"math/big"
)

var (
	ErrInvalidPoint     = errors.New("marshaled point was invalid")
	ErrNoPointFound     = errors.New("hash_to_curve failed to find a point")
	ErrPointOffCurve    = errors.New("point is not on curve")
	ErrUnspecifiedCurve = errors.New("must specify an elliptic curve")
	ErrCommSanityCheck  = errors.New("commitment does not match key")
)

type Point struct {
	X, Y *big.Int
}

func NewPoint(curve elliptic.Curve, x, y *big.Int) (*Point, error) {
	if curve == nil {
		return nil, ErrUnspecifiedCurve
	}
	if !curve.IsOnCurve(x, y) {
		return nil, ErrPointOffCurve
	}
	return &Point{X: x, Y: y}, nil
}

func RandomCurvePoint() (*Point, error) {
	scal, err := RandomCurveScalar(rand.Reader)
	if err != nil {
		return nil, err
	}

	Rx, Ry := CURVE.ScalarBaseMult(scal.ToBytes())

	return &Point{Rx, Ry}, nil
}

func NewBasePoint() *Point {
	return &Point{X: CURVE.Params().Gx, Y: CURVE.Params().Gy}
}

func ZeroPoint() *Point {
	return &Point{X: new(big.Int), Y: new(big.Int)}
}

func (p *Point) ScalarMult(elems ...*ScalarElement) *Point {
	for _, el := range elems {
		x, y := CURVE.ScalarMult(p.X, p.Y, el.ToBytes())
		p.X = x
		p.Y = y
	}
	return p
}

func (p *Point) Accumulate(points ...*Point) *Point {
	for _, other := range points {
		x, y := CURVE.Add(p.X, p.Y, other.X, other.Y)
		p.X = x
		p.Y = y
	}

	return p
}

func (p *Point) IsOnCurve() bool {
	return CURVE.IsOnCurve(p.X, p.Y)
}

func (p *Point) Copy() *Point {
	q, _ := NewPoint(CURVE, new(big.Int).Set(p.X), new(big.Int).Set(p.Y))
	return q
}

func (p *Point) Equals(other *Point) bool {
	return p.X.Cmp(other.X) == 0 && p.Y.Cmp(other.Y) == 0
}
