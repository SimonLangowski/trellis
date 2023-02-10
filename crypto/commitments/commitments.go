package commitments

import (
	"crypto/rand"
	"crypto/sha256"

	"github.com/simonlangowski/lightning1/crypto"
)

/*
I think the simplest (fastest) commitment scheme is just
C = Hash(nonce, message)
Open = nonce
*/

const NONCE_SIZE = 16 // 128 bits
const COMMIT_SIZE = crypto.HASH_SIZE
const OPENING_SIZE = NONCE_SIZE

type Commitment struct {
	// A SHA-256 hash
	hash [crypto.HASH_SIZE]byte
}

type CommitmentOpening struct {
	nonce [NONCE_SIZE]byte
	// message []byte
}

func MakeCommitment(m []byte) (error, *Commitment, *CommitmentOpening) {
	o := &CommitmentOpening{}
	n, err := rand.Reader.Read(o.nonce[:])
	if err != nil || n != NONCE_SIZE {
		return err, nil, nil
	}
	data := make([]byte, NONCE_SIZE+len(m))
	copy(data[:NONCE_SIZE], o.nonce[:])
	copy(data[NONCE_SIZE:], m)
	c := &Commitment{
		hash: sha256.Sum256(data),
	}
	return nil, c, o
}

func (c *Commitment) Open(o *CommitmentOpening, m []byte) bool {
	return CheckCommitment(c, o, m)
}

func CheckCommitment(c *Commitment, o *CommitmentOpening, m []byte) bool {
	data := make([]byte, NONCE_SIZE+len(m))
	copy(data[:NONCE_SIZE], o.nonce[:])
	copy(data[NONCE_SIZE:], m)
	h := sha256.Sum256(data)
	return h == c.hash
}
