package processMessages

import (
	"sync"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/server/common"
)

// Information about an established key path
type BootstrapKey struct {
	// this server is the server in layer l
	// server at l-1
	PrevServer int
	// A precomputation of the DH shared key between the client's key and this server's on link l-1 to l
	SharedKey crypto.DHSharedKey
	// The client's verification key on link l-1 to l
	VerificationKey crypto.VerificationKey

	//
	ExpandedVerificationKey *crypto.ExpandedVerificationKey

	// server at l+1
	NextServer int
	// the shared key corresponding to the link l to l+1
	OutgoingSharedKey crypto.DHSharedKey
	// The client's verification key on the link l to l+1
	OutgoingVerificationKey crypto.VerificationKey

	ExpandedOutgoingVerificationKey *crypto.ExpandedVerificationKey

	used bool // set to true when used
}

type KeyLookupTable struct {
	table        map[crypto.LookupKey]*BootstrapKey // by IncomingLookupKey
	reverseTable map[crypto.LookupKey]*BootstrapKey // by OutgoingLookupKey - used when routing boomerang or in reverse
	secretKey    *crypto.DHPrivateKey               // the secret key for this layer
	mu           sync.Mutex
}

func NewKeyLookupTable(c *common.CommonState) *KeyLookupTable {
	t := &KeyLookupTable{
		table:        make(map[crypto.LookupKey]*BootstrapKey),
		reverseTable: make(map[crypto.LookupKey]*BootstrapKey),
		secretKey:    &c.ServerSecretKey,
	}
	return t
}

func (t *KeyLookupTable) AddKey(key crypto.VerificationKey, sharedKey crypto.DHSharedKey, prev, next int, nextKey crypto.VerificationKey) (*BootstrapKey, error) {
	l := key.LookupKey()
	rl := nextKey.LookupKey()
	pt, err := nextKey.ToCurvePoint()
	if err != nil {
		return nil, errors.BadElementError()
	}
	b := &BootstrapKey{
		SharedKey:               sharedKey,
		VerificationKey:         key.Copy(),
		OutgoingSharedKey:       t.secretKey.SharedKey(pt),
		OutgoingVerificationKey: nextKey.Copy(),
		PrevServer:              prev,
		NextServer:              next,
		used:                    false,
	}
	if config.PreExpandKeys {
		// in lightning rounds
		b.ExpandedVerificationKey, err = b.VerificationKey.ExpandKey()
		if err != nil {
			return nil, err
		}
		// in path establishment rounds
		// b.ExpandedOutgoingVerificationKey, err = b.OutgoingVerificationKey.ExpandKey()
		// if err != nil {
		// 	return nil, err
		// }
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.table[l] = b
	t.reverseTable[rl] = b
	return b, nil
}

func (t *KeyLookupTable) Lookup(key *crypto.LookupKey, reverse bool) *BootstrapKey {
	t.mu.Lock()
	defer t.mu.Unlock()
	if reverse {
		return t.reverseTable[*key]
	} else {
		return t.table[*key]
	}
}

func (t *KeyLookupTable) NumKeys() int {
	return len(t.table)
}

func (t *KeyLookupTable) RevokeKey(key *crypto.VerificationKey) {
	delete(t.table, key.LookupKey())
	// delete(t.reverseTable, key.LookupKey())
}

func (t *KeyLookupTable) ResetUsage() {
	for _, k := range t.table {
		k.used = false
	}
}

/*
const size = 2 * (crypto.POINT_SIZE + crypto.KEY_SIZE + 8)

func (table *KeyLookupTable) LoadTableFromFile(fn string) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	b := make([]byte, size)
	for err != io.EOF {
		_, err = r.Read(b)
		if err != nil && err != io.EOF {
			return err
		}
		key := &BootstrapKey{}
		pos := 0
		// key.ClientKey.InterpretFrom(b[pos : pos+crypto.POINT_SIZE])
		// pos += crypto.POINT_SIZE
		key.SharedKey.InterpretFrom(b[pos : pos+crypto.POINT_SIZE])
		pos += crypto.POINT_SIZE
		copy(key.IncomingLookupKey[:], b[pos:pos+crypto.KEY_SIZE])
		pos += crypto.KEY_SIZE
		key.NextServer = int(binary.LittleEndian.Uint64(b[pos : pos+8]))
		pos += 8
		copy(key.OutgoingLookupKey[:], b[pos:pos+crypto.KEY_SIZE])
		pos += crypto.KEY_SIZE
		key.PrevServer = int(binary.LittleEndian.Uint64(b[pos : pos+8]))

		table.table[key.IncomingLookupKey] = key
		table.reverseTable[key.OutgoingLookupKey] = key
	}
	return nil
}

func (table *KeyLookupTable) WriteTableToFile(fn string) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	t := make([]byte, 8)
	for _, key := range table.table {
		// write each key to the file
		// w.Write(key.ClientKey.Marshal())
		w.Write(key.SharedKey.Marshal())
		w.Write(key.IncomingLookupKey[:])
		binary.LittleEndian.PutUint64(t, uint64(key.NextServer))
		w.Write(t)
		w.Write(key.OutgoingLookupKey[:])
		binary.LittleEndian.PutUint64(t, uint64(key.PrevServer))
		_, err = w.Write(t)
		if err != nil {
			return err
		}
	}
	return nil
}
*/
