package mcl

// from https://github.com/alinush/go-mcl/blob/master/mcl.go

import (
	"crypto/rand"
	"fmt"
)

// Randomly picks an element in G1 by hashing enough bytes from crypto/rand to the group G1.
func (x *G1) Random() {
	// there are as many group elements as there are field elements in the exponent
	bytes := make([]byte, GetFrUnitSize()*8)

	// pick enough random bytes
	n, err := rand.Read(bytes)
	if err != nil || n == 0 {
		panic(fmt.Sprintf("Error randomly generating G1: %v\n", err))
	}

	// hash these bytes to a group element
	x.HashAndMapTo(bytes)
}

// Randomly picks an element in G2 by hashing enough bytes from crypto/rand to the group G2.
func (x *G2) Random() {
	// there are as many group elements as there are field elements in the exponent
	bytes := make([]byte, GetFrUnitSize()*8)

	// pick enough random bytes
	n, err := rand.Read(bytes)
	if err != nil || n == 0 {
		panic(fmt.Sprintf("Error randomly generating G2: %v\n", err))
	}

	// hash these bytes to a group element
	x.HashAndMapTo(bytes)
}

func (x *Fr) Random() {
	x.SetByCSPRNG()
}

func (x *Fp) Random() {
	x.SetByCSPRNG()
}
