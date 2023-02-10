package kyber_wrap

import (
	"crypto/cipher"

	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/random"
)

// Implement the scalar interface for the group for mcl.Fr

type Scalar struct {
	fr mcl.Fr
}

func (s *Scalar) Equal(s2 kyber.Scalar) bool {
	return s.fr.IsEqual(&s2.(*Scalar).fr)
}

func (s *Scalar) Set(a kyber.Scalar) kyber.Scalar {
	s.fr = a.(*Scalar).fr
	return s
}

func (s *Scalar) Clone() kyber.Scalar {
	s2 := &Scalar{}
	s2.fr = s.fr
	return s2
}

func (s *Scalar) SetInt64(v int64) kyber.Scalar {
	s.fr.SetInt64(v)
	return s
}

func (s *Scalar) Zero() kyber.Scalar {
	s.fr.Clear()
	return s
}

func (s *Scalar) Add(a, b kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	e2 := b.(*Scalar)
	mcl.FrAdd(&s.fr, &e1.fr, &e2.fr)
	return s
}

func (s *Scalar) Sub(a, b kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	e2 := b.(*Scalar)
	mcl.FrSub(&s.fr, &e1.fr, &e2.fr)
	return s
}

func (s *Scalar) Neg(a kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	mcl.FrNeg(&s.fr, &e1.fr)
	return s
}

func (s *Scalar) One() kyber.Scalar {
	s.fr.SetInt64(1)
	return s
}

func (s *Scalar) Mul(a, b kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	e2 := b.(*Scalar)
	mcl.FrMul(&s.fr, &e1.fr, &e2.fr)
	return s
}

func (s *Scalar) Div(a, b kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	e2 := b.(*Scalar)
	mcl.FrDiv(&s.fr, &e1.fr, &e2.fr)
	return s
}

func (s *Scalar) Inv(a kyber.Scalar) kyber.Scalar {
	e1 := a.(*Scalar)
	mcl.FrInv(&s.fr, &e1.fr)
	return s
}

func (s *Scalar) Pick(rand cipher.Stream) kyber.Scalar {
	b := random.Int(CURVE_MOD, rand)
	s.fr.SetBigEndianMod(b.Bytes())
	return s
}

func (s *Scalar) SetBytes(b []byte) kyber.Scalar {
	s.fr.SetBigEndianMod(b)
	return s
}
