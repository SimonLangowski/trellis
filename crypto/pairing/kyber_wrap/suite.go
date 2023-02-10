package kyber_wrap

import (
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"go.dedis.ch/kyber/v3"
)

type BLS12Suite struct {
	BLS12_381Group
	kyber.HashFactory
	kyber.XOFFactory
	kyber.Random
}

// implements kyber.Group interface
type BLS12_381Group struct {
}

func NewBLS12Suite(hf kyber.HashFactory, xof kyber.XOFFactory, rnd kyber.Random) *BLS12Suite {
	return &BLS12Suite{
		BLS12_381Group: BLS12_381Group{},
		HashFactory:    hf,
		XOFFactory:     xof,
		Random:         rnd,
	}
}

func (g *BLS12_381Group) String() string {
	return "BLS12-381 G2 group"
}

func (g *BLS12_381Group) ScalarLen() int {
	return mcl.FR_LEN
}

func (g *BLS12_381Group) Scalar() kyber.Scalar {
	s := &Scalar{}
	return s.Zero()
}

func (g *BLS12_381Group) PointLen() int {
	return mcl.G2_LEN
}

func (g *BLS12_381Group) Point() kyber.Point {
	p := &Point{}
	return p.Null()
}
