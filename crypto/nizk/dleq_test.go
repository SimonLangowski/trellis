// from privacypass/challenge-bypass-server

package nizk

import (
	"crypto/elliptic"
	"crypto/rand"
	_ "crypto/sha256"
	"testing"

	"github.com/simonlangowski/lightning1/crypto/ec"
)

func setup() (*ec.ScalarElement, *ec.Point, *ec.Point, error) {

	curve := ec.CURVE

	// All public keys are going to be generators, so GenerateKey is a handy
	// test function. However, TESTING ONLY. Maintaining the discrete log
	// relationship breaks the token scheme. Ideally the generator points
	// would come from a group PRF or something like Elligator.
	_, x, _, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}
	_, Gx, Gy, err := elliptic.GenerateKey(curve, rand.Reader)
	G := &ec.Point{X: Gx, Y: Gy}
	if err != nil {
		return nil, nil, nil, err
	}
	_, Mx, My, err := elliptic.GenerateKey(curve, rand.Reader)
	M := &ec.Point{X: Mx, Y: My}
	if err != nil {
		return nil, nil, nil, err
	}

	return ec.NewScalarElement(x), G, M, nil
}

// Tests that a DLEQ proof over validly signed tokens always verifies correctly
func TestValidProof(t *testing.T) {

	x, G, M, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	H := G.Copy()
	H.ScalarMult(x)

	Z := M.Copy()
	Z.ScalarMult(x)

	proof, err := NewProof(G, H, M, Z, x)
	if err != nil {
		t.Fatal(err)
	}

	if !proof.Verify(G, H, M, Z) {
		t.Fatal("proof was invalid")
	}
}

func TestInvalidProof(t *testing.T) {

	x, G, M, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	curve := ec.CURVE

	_, n, _, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	H := G.Copy()
	H.ScalarMult(x)

	// using Z = nM instead
	Z := M.Copy()
	Z.ScalarMult(ec.NewScalarElement(n))

	proof, err := NewProof(G, H, M, Z, x)
	if err != nil {
		t.Fatal(err)
	}

	if proof.Verify(G, H, M, Z) {
		t.Fatal("validated an invalid proof")
	}
}
