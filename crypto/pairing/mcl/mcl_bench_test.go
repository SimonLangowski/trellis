package mcl

// from https://github.com/alinush/go-mcl/blob/master/mcl.go

//import "fmt"
//#import "strings"
import "testing"

func BenchmarkG1mul(b *testing.B) {
	var a Fr
	var R, P G1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		P.Random()
		a.Random()
		//var numOnes = strings.Count(a.GetString(2), "1")
		//fmt.Printf("1s: %d\n", numOnes)

		b.StartTimer()
		G1Mul(&R, &P, &a)
	}
}

func BenchmarkG2mul(b *testing.B) {
	var a Fr
	var R, P G2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		P.Random()
		a.Random()

		b.StartTimer()
		G2Mul(&R, &P, &a)
	}
}

func BenchmarkPairing(b *testing.B) {
	var e GT
	var Q G1
	var P G2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()

		P.Random()
		Q.Random()

		b.StartTimer()
		Pairing(&e, &Q, &P)
	}
}

//
//func testVecPairing(t *testing.T) {
//	N := 50
//	g1Vec := make([]G1, N)
//	g2Vec := make([]G2, N)
//	var e1, e2 GT
//	e1.SetInt64(1)
//	for i := 0; i < N; i++ {
//		g1Vec[0].HashAndMapTo([]byte("aa"))
//		g2Vec[0].HashAndMapTo([]byte("aa"))
//		Pairing(&e2, &g1Vec[i], &g2Vec[i])
//		GTMul(&e1, &e1, &e2)
//	}
//	MillerLoopVec(&e2, g1Vec, g2Vec)
//	FinalExp(&e2, &e2)
//	if !e1.IsEqual(&e2) {
//		t.Errorf("wrong MillerLoopVec")
//	}
//}
