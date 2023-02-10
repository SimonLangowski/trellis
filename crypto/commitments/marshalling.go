package commitments

import "github.com/simonlangowski/lightning1/errors"

func (c *Commitment) Len() int {
	return COMMIT_SIZE
}

func (c *Commitment) PackTo(b []byte) {
	if len(b) != c.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b, c.hash[:])
}

func (c *Commitment) InterpretFrom(b []byte) error {
	if len(b) != c.Len() {
		return errors.LengthInvalidError()
	}
	copy(c.hash[:], b)
	return nil
}

func (o *CommitmentOpening) Len() int {
	return NONCE_SIZE
}

func (o *CommitmentOpening) PackTo(b []byte) {
	if len(b) != o.Len() {
		panic(errors.LengthInvalidError())
	}
	copy(b, o.nonce[:])
}

func (o *CommitmentOpening) InterpretFrom(b []byte) error {
	if len(b) != o.Len() {
		return errors.LengthInvalidError()
	}
	copy(o.nonce[:], b)
	return nil
}
