package ec

// from privacypass/challenge-bypass-server

import (
	"crypto/sha256"
	"math/big"
)

const seed = "1.2.840.10045.3.1.7 point generation seed"

var curveFieldByteLength int

func init() {
	curveFieldByteLength = (CURVE.Params().BitSize + 7) >> 3
}

// P256SHA256SWU calculates the Simplified SWU encoding by Brier et al.
// given in "Efficient Indifferentiable Hashing into Ordinary Elliptic Curves".
// It assumes that curve is one of the NIST curves; thus a=-3 and p=3 mod 4.
// Compatible with Privacy Pass > v1.0.

func HashToCurvePoint(data []byte) *Point {
	t := HashToBaseField(data)
	P := SimplifiedSWU(t)
	return P
}

// Hashes bytes to a big.Int that will be interpreted as a field element
func HashToBaseField(data []byte) *big.Int {
	byteLen := curveFieldByteLength
	h := sha256.New()
	h.Write([]byte(seed))
	h.Write(data)
	sum := h.Sum(nil)
	t := new(big.Int).SetBytes(sum[:byteLen])
	t.Mod(t, CURVE_MOD)
	return t
}

func SimplifiedSWU(t *big.Int) *Point {
	var u, t0, y2, bDivA, g, pPlus1Div4, x, y big.Int
	e := CURVE.Params()
	p := e.P
	A := big.NewInt(-3)
	B := e.B
	// bDivA = -B/A
	bDivA.ModInverse(A, p)
	bDivA.Mul(&bDivA, B)
	bDivA.Neg(&bDivA)
	bDivA.Mod(&bDivA, p)
	// pplus1div4 = (p+1)/4
	pPlus1Div4.SetInt64(1)
	pPlus1Div4.Add(&pPlus1Div4, p)
	pPlus1Div4.Rsh(&pPlus1Div4, 2)
	// u = -t^2
	u.Mul(t, t)
	u.Neg(&u)
	u.Mod(&u, p)
	// t0 = 1/(u^2+u)
	t0.Mul(&u, &u)
	t0.Add(&t0, &u)
	t0.Mod(&t0, p)
	// if t is {0,1,-1} returns error (point at infinity)
	if t0.Sign() == 0 {
		panic("Curve hash failure")
	}
	t0.ModInverse(&t0, p)
	// x = (-B/A)*( 1+1/(u^2+u) ) = bDivA*(1+t0)
	x.SetInt64(1)
	x.Add(&x, &t0)
	x.Mul(&x, &bDivA)
	x.Mod(&x, p)
	// g = (x^2+A)*x+B
	g.Mul(&x, &x)
	g.Mod(&g, p)
	g.Add(&g, A)
	g.Mul(&g, &x)
	g.Mod(&g, p)
	g.Add(&g, B)
	g.Mod(&g, p)
	// y = g^((p+1)/4)
	y.Exp(&g, &pPlus1Div4, p)
	// if y^2 != g, then x = -t^2*x and y = (-1)^{(p+1)/4}*t^3*y
	y2.Mul(&y, &y)
	y2.Mod(&y2, p)
	if y2.Cmp(&g) != 0 {
		// x = -t^2*x
		x.Mul(&x, &u)
		x.Mod(&x, p)
		// y = t^3*y
		y.Mul(&y, &u)
		y.Mod(&y, p)
		y.Neg(&y)
		y.Mul(&y, t)
		y.Mod(&y, p)
	}
	return &Point{&x, &y}
}
