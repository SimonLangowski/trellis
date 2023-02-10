package kyber_wrap

import (
	"crypto/cipher"
	"math/big"

	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/random"
)

// Implement the point interface of Group for the mcl.G2 point

var CURVE_MOD *big.Int
var One mcl.Fp2
var G2Generator mcl.G2

func init() {
	curveOrder := mcl.GetCurveOrder()
	CURVE_MOD, _ = new(big.Int).SetString(curveOrder, 10)
	One.D[0].SetInt64(1)
	One.D[1].SetInt64(0)
	if !One.IsOne() {
		panic("Error setting one")
	}
	mcl.MapToG2(&G2Generator, &One)
}

type Point struct {
	g2 mcl.G2
}

func (p *Point) Equal(P2 kyber.Point) bool {
	return p.g2.IsEqual(&(P2.(*Point).g2))
}

func (p *Point) Set(P2 kyber.Point) kyber.Point {
	p.g2 = P2.(*Point).g2
	return p
}

func (p *Point) Clone() kyber.Point {
	p2 := &Point{}
	p2.g2 = p.g2
	return p2
}

func (p *Point) Null() kyber.Point {
	p2 := &Point{}
	p2.g2.Clear()
	return p2
}

func (p *Point) Base() kyber.Point {
	p.g2 = G2Generator
	return p
}

func (p *Point) Add(P1, P2 kyber.Point) kyber.Point {
	E1 := P1.(*Point)
	E2 := P2.(*Point)
	mcl.G2Add(&p.g2, &E1.g2, &E2.g2)
	return p
}

func (p *Point) Sub(P1, P2 kyber.Point) kyber.Point {
	E1 := P1.(*Point)
	E2 := P2.(*Point)
	mcl.G2Sub(&p.g2, &E1.g2, &E2.g2)
	return p
}

func (p *Point) Neg(A kyber.Point) kyber.Point {
	mcl.G2Neg(&p.g2, &A.(*Point).g2)
	return p
}

func (p *Point) Mul(s kyber.Scalar, A kyber.Point) kyber.Point {
	sc := s.(*Scalar)
	if A == nil {
		A = p.Base()
	}
	a := A.(*Point)
	mcl.G2Mul(&p.g2, &a.g2, &sc.fr)
	return p
}

// I hope I don't need these, so we'll just say we embed 0
func (p *Point) EmbedLen() int {
	return 0
}

func (p *Point) Embed(data []byte, r cipher.Stream) kyber.Point {
	return p
}

func (p *Point) Data() ([]byte, error) {
	return []byte{}, nil
}

func (p *Point) Pick(rand cipher.Stream) kyber.Point {
	b := [32]byte{}
	random.Bytes(b[:], rand)
	p.g2.HashAndMapTo(b[:])
	return p
}
