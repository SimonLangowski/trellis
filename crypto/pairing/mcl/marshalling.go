package mcl

import "github.com/simonlangowski/lightning1/errors"

var FR_LEN int
var G1_LEN int
var G2_LEN int

func initSizes() {
	var f Fr
	var g1 G1
	var g2 G2
	FR_LEN = len(f.Serialize())
	G1_LEN = len(g1.Serialize())
	G2_LEN = len(g2.Serialize())
}

func (f *Fr) Len() int {
	return FR_LEN
}

func (f *Fr) PackTo(b []byte) {
	if len(b) != f.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:], f.Serialize())
}

func (f *Fr) InterpretFrom(b []byte) error {
	if len(b) != f.Len() {
		return errors.LengthInvalidError()
	}
	err := f.Deserialize(b)
	if err != nil {
		return err
	}
	if !f.IsValid() {
		return errors.BadElementError()
	}
	return nil
}

func (f *G1) Len() int {
	return G1_LEN
}

func (f *G1) PackTo(b []byte) {
	if len(b) != f.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:], f.Serialize())
}

func (f *G1) InterpretFrom(b []byte) error {
	if len(b) != f.Len() {
		return errors.LengthInvalidError()
	}
	err := f.Deserialize(b)
	if err != nil {
		return err
	}
	if !f.IsValid() {
		return errors.BadElementError()
	}
	return nil
}

func (f *G2) Len() int {
	return G2_LEN
}

func (f *G2) PackTo(b []byte) {
	if len(b) != f.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:], f.Serialize())
}

func (f *G2) InterpretFrom(b []byte) error {
	if len(b) != f.Len() {
		return errors.LengthInvalidError()
	}
	err := f.Deserialize(b)
	if err != nil {
		return err
	}
	if !f.IsValid() {
		return errors.BadElementError()
	}
	return nil
}
