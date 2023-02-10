package pairing

import (
	"github.com/simonlangowski/lightning1/crypto/pairing/kyber_wrap"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
)

var G2GeneratorPrecompute *Precompute
var NegatedPrecompute *Precompute

func init() {
	G2GeneratorPrecompute = NewPrecompute(&kyber_wrap.G2Generator)
	var minusOne mcl.G2
	mcl.G2Neg(&minusOne, &kyber_wrap.G2Generator)
	NegatedPrecompute = NewPrecompute(&minusOne)
}

type Precompute struct {
	data []uint64
}

func NewPrecompute(base *mcl.G2) *Precompute {
	precomputeSize := mcl.GetUint64NumToPrecompute()
	p := &Precompute{data: make([]uint64, precomputeSize)}
	mcl.PrecomputeG2(p.data, base)
	return p
}

func (p *Precompute) Pairing(out *mcl.GT, val *mcl.G1) {
	mcl.PrecomputedMillerLoop(out, val, p.data)
	mcl.FinalExp(out, out)
}

func (p *Precompute) PrecomputedPairingCheck(val1 *mcl.G1, val2 *mcl.G1) bool {
	// https://hackmd.io/@benjaminion/bls12-381#Final-exponentiation
	var e1, e2 mcl.GT
	mcl.PrecomputedMillerLoop(&e1, val1, NegatedPrecompute.data) // e1^-1 = e(sign, -Q)
	mcl.PrecomputedMillerLoop(&e2, val2, p.data)                 // e2 = e(hash, sQ)
	mcl.GTMul(&e1, &e1, &e2)                                     // e1^-1 * e2 = 1
	mcl.FinalExp(&e1, &e1)
	return e1.IsOne()
}

func AdditiveShares(secret *mcl.Fr, numShares int) []mcl.Fr {
	signingShares := make([]mcl.Fr, numShares)
	signingShares[0] = *secret
	for i := 1; i < len(signingShares); i++ {
		signingShares[i].Random()
		mcl.FrSub(&signingShares[0], &signingShares[0], &signingShares[i])
	}
	return signingShares
}
