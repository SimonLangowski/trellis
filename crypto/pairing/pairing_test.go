package pairing

import (
	"fmt"
	"testing"

	"github.com/simonlangowski/lightning1/crypto/pairing/kyber_wrap"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

func TestOrder(t *testing.T) {
	if kyber_wrap.CURVE_MOD == nil || kyber_wrap.CURVE_MOD.IsUint64() {
		t.Fail()
	}
	if fmt.Sprintf("%v", kyber_wrap.CURVE_MOD) != mcl.GetCurveOrder() {
		t.Fail()
	}
	t.Logf("Curve mod: %v", kyber_wrap.CURVE_MOD)
	t.Logf("Curve order: %v", mcl.GetCurveOrder())
}

func TestPrecompute(t *testing.T) {
	var e1, e2 mcl.GT
	var r mcl.G1
	r.Random()
	t.Logf("r: %v, Q: %v", r, kyber_wrap.G2Generator)
	t.Logf("precompute: %v", G2GeneratorPrecompute)
	mcl.Pairing(&e1, &r, &kyber_wrap.G2Generator)
	G2GeneratorPrecompute.Pairing(&e2, &r)
	t.Logf("e1: %v, e2: %v", e1, e2)
	if !e1.IsEqual(&e2) {
		t.Fail()
	}
}

func BenchmarkPrecompute(b *testing.B) {
	var e1, e2 mcl.GT
	var r mcl.G1
	r.Random()
	mcl.Pairing(&e1, &r, &kyber_wrap.G2Generator)
	for i := 0; i < b.N; i++ {
		G2GeneratorPrecompute.Pairing(&e2, &r)
		if !e1.IsEqual(&e2) {
			b.Fail()
		}
	}
}

func BenchmarkOrder(b *testing.B) {
	var r mcl.G1
	r.Random()
	for i := 0; i < b.N; i++ {
		if !r.IsValidOrder() {
			b.Fail()
		}
	}
}

func BenchmarkValid(b *testing.B) {
	var r mcl.G1
	r.Random()
	for i := 0; i < b.N; i++ {
		if !r.IsValid() {
			b.Fail()
		}
	}
}
