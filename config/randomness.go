package config

import (
	"encoding/binary"
	"io"
	"log"
	"sync"

	"golang.org/x/crypto/sha3"
)

// just some random bytes. picking a seed in advance to repeat
// experiments deterministically
var Seed = []byte{
	196, 90, 111, 1, 181, 197, 66, 167, 74, 39, 198, 144, 4, 179, 62, 115,
	192, 144, 122, 196, 242, 225, 81, 118, 131, 206, 191, 12, 210, 221, 64, 192,
	161, 225, 17, 161, 202, 156, 90, 143, 55, 195, 143, 187, 143, 252, 7, 6,
	0, 245, 9, 16, 3, 192, 43, 236, 164, 230, 24, 1, 174, 71, 189, 252,
	216, 146, 139, 105, 3, 0, 70, 9, 102, 179, 127, 147, 154, 104, 11, 155,
	63, 133, 121, 66, 142, 141, 23, 240, 81, 53, 166, 154, 39, 105, 179, 45,
}

type Shuffler struct {
	src io.Reader
	buf []byte
	pos int
	mu  sync.Mutex
}

// use a fixed seed
func SeededShuffler() *Shuffler {
	h := sha3.NewShake128()
	h.Write(Seed)
	b := make([]byte, 128)
	h.Read(b)
	return &Shuffler{src: h, buf: b}
}

// initalize a PRG from the randomness
func NewPRGShuffler(randomnessSource io.Reader) *Shuffler {
	seed := make([]byte, 128)
	randomnessSource.Read(seed)
	h := sha3.NewShake128()
	h.Write(seed)
	return &Shuffler{src: h, buf: seed}
}

// use the randomness source
func NewShuffler(randomnessSource io.Reader, buffSize int) *Shuffler {
	b := make([]byte, buffSize)
	randomnessSource.Read(b)
	return &Shuffler{src: randomnessSource, buf: b}
}

func (r *Shuffler) readBytes(length int) []byte {
	e := r.pos + length
	if e >= len(r.buf) {
		// refill buffer
		n, err := r.src.Read(r.buf)
		if err != nil || (n != len(r.buf)) {
			log.Fatalf("Error getting randomness: %v", err)
		}
		r.pos = 0
		e = length
	}
	output := r.buf[r.pos:e]
	r.pos = e
	return output
}

func (r *Shuffler) UInt64() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return binary.BigEndian.Uint64(r.readBytes(8))
}

func (r *Shuffler) UInt32() uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return binary.BigEndian.Uint32(r.readBytes(4))
}

// Knuth shuffle
func (r *Shuffler) Perm(n int) []int {
	pi := make([]int, n)
	for i := 0; i < n; i++ {
		pi[i] = i
	}
	for i := 0; i <= n-2; i++ {
		j := int(r.UInt32() % uint32(n-i))
		j += i
		pi[i], pi[j] = pi[j], pi[i]
	}
	return pi
}

// select t out of n, without replacement
// Note that this has a higher probability than a group that is selected with replacement as the groupSize formula calculates
// Each duplicate reselected if there is replacement is a lost chance to select an honest server
func (r *Shuffler) SelectRandom(n int, t int) []int {
	return r.Perm(n)[:t]
}

// random [0,max)
func (r *Shuffler) Intn(max int) int {
	return int(r.UInt64() % uint64(max))
}
