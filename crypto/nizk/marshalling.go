package nizk

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/ec"
	"github.com/simonlangowski/lightning1/errors"
)

var DL_SIZE = ec.CurveElementSize + ec.ScalarElementSize + crypto.HASH_SIZE
var DLEQ_SIZE = ec.CurveElementSize + ec.ScalarElementSize + crypto.HASH_SIZE

func (d *DLEQProof) Len() int {
	return ec.ScalarElementSize + crypto.HASH_SIZE
}

func (d *DLEQProof) PackTo(b []byte) {
	if len(b) != d.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b[:crypto.HASH_SIZE], d.Hash)
	d.R.PackTo(b[crypto.HASH_SIZE:])
}

func (d *DLEQProof) InterpretFrom(b []byte) error {
	if len(b) != d.Len() {
		return errors.LengthInvalidError()
	}
	d.Hash = b[:crypto.HASH_SIZE]
	d.R = &ec.ScalarElement{}
	err := d.R.InterpretFrom(b[crypto.HASH_SIZE:])
	if err != nil {
		return err
	}
	d.C = ec.ZeroScalar()
	d.C.FromBytes(d.Hash)
	return nil
}

func (d *DLProof) Len() int {
	return d.Value.MarshalSize() + d.V.MarshalSize() + len(d.Hash)
}

func (d *DLProof) PackTo(b []byte) {
	if len(b) != d.Len() {
		panic(errors.LengthInvalidError())
	}
	pos := 0
	copy(b[:], d.Hash)
	pos += len(d.Hash)
	b1, _ := d.V.MarshalBinary()
	b2, _ := d.Value.MarshalBinary()
	copy(b[pos:], b1)
	pos += len(b1)
	copy(b[pos:], b2)
}

func (d *DLProof) InterpretFrom(b []byte) error {
	if len(b) != d.Len() {
		return errors.LengthInvalidError()
	}
	panic(errors.UnimplementedError())
}

func (d *DLProof) Marshal() []byte {
	b := make([]byte, d.Len())
	d.PackTo(b)
	return b
}
