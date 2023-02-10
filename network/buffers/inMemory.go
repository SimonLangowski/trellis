package buffers

import (
	"sync"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/errors"
)

// If I had a read/write mmap I could just map anonymous for in memory and map file otherwise
// For lightning/boomerang it will fit in memory - shuffle sender probably also okay
// chunk stream still allows computing during network

type MemReadWriter struct {
	data          [][]byte
	permutation   []int
	position      int
	elementLength int
	elementCount  int
	offset        int
	element       []byte
	zeros         []byte
	numElements   int
	shuf          *config.Shuffler
	mu            sync.Mutex
}

func NewMemReadWriter(elementLength, numElements int, shuf *config.Shuffler) *MemReadWriter {
	m := &MemReadWriter{
		data:          make([][]byte, numElements),
		offset:        elementLength,
		element:       make([]byte, elementLength),
		zeros:         make([]byte, elementLength),
		elementLength: elementLength,
		numElements:   numElements,
		shuf:          shuf,
	}
	return m
}

func (m *MemReadWriter) Shuffle(dummies bool) {
	if !dummies {
		m.numElements = m.elementCount
	}
	m.permutation = m.shuf.Perm(m.numElements)
}

func (m *MemReadWriter) NumMessages() int {
	return m.elementCount
}

func (m *MemReadWriter) Write(b []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.elementCount >= len(m.data) {
		return errors.LinkOverflow()
	}
	m.data[m.elementCount] = b
	m.elementCount += 1
	return nil
}

func (r *MemReadWriter) ReadNextChunk(b []byte) (int, error) {
	size := len(b)
	written := 0
	var err error = nil
	// write any remainder from previous call
	if r.elementLength-r.offset > 0 {
		// this could fill the entire buffer for very large elements...
		written = copy(b, r.element[r.offset:])
		r.offset += written
	}
	// read next element as applicable
	for written < size && r.position < r.numElements && err == nil {
		if written+r.elementLength > size {
			// partial read into buffer
			err = r.ReadElement(r.element)
			// copy part into buffer, leave remainder for next time
			r.offset = copy(b[written:], r.element)
			written += r.offset
		} else {
			// full read
			err = r.ReadElement(b[written : written+r.elementLength])
			written += r.elementLength
		}
	}
	// if the last part is short, it will need to be truncated to written
	return written, err
}

func (r *MemReadWriter) ReadElement(b []byte) error {
	elementIndex := r.permutation[r.position]
	r.position++
	if elementIndex >= r.elementCount {
		// dummy element - write 0s in buffer
		copy(b, r.zeros)
		return nil
	}
	copy(b, r.data[elementIndex])
	// free memory - we should only read each element once
	r.data[elementIndex] = nil
	return nil
}

func (r *MemReadWriter) Len() int {
	return r.numElements * r.elementLength
}
