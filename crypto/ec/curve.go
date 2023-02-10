// from privacypass/challenge-bypass-server

package ec

import (
	"crypto/elliptic"
)

// This could all be swapped out with gmp ints
// and montgomery multiplication for speed
var CURVE = elliptic.P256()

func ScalarBaseMult(s *ScalarElement) *Point {
	x, y := CURVE.ScalarBaseMult(s.Int.Bytes())
	return &Point{X: x, Y: y}
}
