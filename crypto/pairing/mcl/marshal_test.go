package mcl

import "testing"

func TestLengths(t *testing.T) {
	t.Logf("FR: %v, G1: %v, G2: %v", FR_LEN, G1_LEN, G2_LEN)
	if FR_LEN != 32 {
		t.Fail()
	}
	if G1_LEN != 48 {
		t.Fail()
	}
	if G2_LEN != 96 {
		t.Fail()
	}
}

func TestMarshalling(t *testing.T) {
	var f, a Fr
	var g1, b G1
	var g2, c G2
	f.Random()
	g1.Random()
	g2.Random()
	fb := make([]byte, FR_LEN)
	g1b := make([]byte, G1_LEN)
	g2b := make([]byte, G2_LEN)
	f.PackTo(fb)
	g1.PackTo(g1b)
	g2.PackTo(g2b)
	a.InterpretFrom(fb)
	b.InterpretFrom(g1b)
	c.InterpretFrom(g2b)
	if !f.IsEqual(&a) || !g1.IsEqual(&b) || !g2.IsEqual(&c) {
		t.Fail()
	}
}
