package processMessages

import (
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/common"
)

type OnionParser struct {
	c         *common.CommonState
	usageLock sync.Mutex
	count     int
	keyTable  *KeyLookupTable
	reverse   bool
}

type LightningRouter struct {
	OutgoingBuffers map[int]*buffers.MemReadWriter
}

func NewOnionParser(c *common.CommonState, table *KeyLookupTable, reverse bool) *OnionParser {
	return &OnionParser{
		c:        c,
		count:    0,
		keyTable: table,
		reverse:  reverse,
	}
}

// Decypt the message as lightning messages
// Check that each key is used exactly once
// Return the next destinations if set
func (o *OnionParser) AuthenticatedOnionParse(metadata *messages.Metadata, message []byte) ([]byte, *BootstrapKey, error) {
	// to make the nonce different for the boomerang messages
	layer := o.c.Layer
	if o.reverse {
		layer += o.c.NumLayers
	}
	round := o.c.Round
	server := o.c.MyId
	nonce := crypto.Nonce(round, layer, server)

	lm := common.LightningEnvelope{}
	err := lm.InterpretFrom(message)
	if err != nil {
		return nil, nil, err
	}
	key := o.keyTable.Lookup(&lm.Key, o.reverse)
	if key == nil {
		return nil, nil, errors.KeyNotFound()
	}

	var verificationKey crypto.VerificationKey
	var decryptionKey crypto.DHSharedKey
	var expandedKey *crypto.ExpandedVerificationKey
	if !o.reverse {
		decryptionKey = key.SharedKey
		verificationKey = key.VerificationKey
		expandedKey = key.ExpandedVerificationKey
	} else {
		verificationKey = key.OutgoingVerificationKey
		decryptionKey = key.OutgoingSharedKey
		expandedKey = key.ExpandedOutgoingVerificationKey
	}
	// or send to signature co processor
	m := lm.GetSignedData(round, layer, server)
	s := lm.GetSignature()
	errors.DebugPrint("Verifying %v on %v with %v", m, s, verificationKey.PublicKey())

	ok := false
	if expandedKey != nil {
		ok = crypto.VerifyExpanded(expandedKey, m, s)
	} else {
		ok = crypto.Verify(verificationKey, m, s)
	}
	if !ok {
		return nil, nil, errors.DecryptionFailure()
	}

	decrypted := crypto.SecretOpen(lm.SignedCiphertext, &nonce, decryptionKey)
	o.usageLock.Lock()
	if key.used {
		o.usageLock.Unlock()
		return nil, nil, errors.Duplicate()
	}
	key.used = true
	o.count++
	o.usageLock.Unlock()
	return decrypted, key, nil
}

func (o *OnionParser) AllKeysAccountedFor() bool {
	o.usageLock.Lock()
	defer o.usageLock.Unlock()
	ok := o.count == o.keyTable.NumKeys()
	if ok {
		o.keyTable.ResetUsage()
	}
	return ok
}

func NewLightningRouter(c *common.CommonState, layer int, reverse bool) *LightningRouter {
	l := &LightningRouter{
		OutgoingBuffers: make(map[int]*buffers.MemReadWriter),
	}
	var length int
	if reverse {
		length = c.OnionMessageLengths[layer]
	} else {
		length = c.OnionMessageLengths[layer+1]
	}
	for i := 0; i < c.NumServers; i++ {
		l.OutgoingBuffers[i] = buffers.NewMemReadWriter(length, c.BinSize, c.Shufflers[i])
	}
	return l
}

// Pack the decryptions into lightning messages
func (l *LightningRouter) AuthenticatedOnionPack(decrypted []byte, k *BootstrapKey, reverse bool) error {
	var dest int
	m := common.LightningEnvelope{
		SignedCiphertext: decrypted,
	}
	if reverse {
		dest = k.PrevServer
		m.Key = k.VerificationKey.LookupKey()
	} else {
		dest = k.NextServer
		m.Key = k.OutgoingVerificationKey.LookupKey()
	}
	err := l.OutgoingBuffers[dest].Write(m.Marshal())
	if err != nil {
		return err
	}
	return nil
}
