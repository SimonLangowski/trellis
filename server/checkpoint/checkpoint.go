package checkpoint

import (
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
	"github.com/simonlangowski/lightning1/server/common"
)

type VerificationKeyTable struct {
	mu    sync.Mutex
	keys  map[[crypto.VERIFICATION_KEY_SIZE]byte]bool
	count int
}

type Checkpoint struct {
	commonState          *common.CommonState
	myGroupId            int
	numGroups            int
	groupKeyShare        *crypto.DHPrivateKey
	AnonymousSigningKeys VerificationKeyTable
	synchronizer         *synchronization.Synchronizer
	mu                   sync.Mutex
	FinalMessages        [][]byte
}

func NewCheckpointState(c *common.CommonState, myGroupId int, secret *crypto.DHPrivateKey, synchronizer *synchronization.Synchronizer) *Checkpoint {
	return &Checkpoint{
		commonState: c,
		myGroupId:   myGroupId,
		numGroups:   c.NumGroups,

		groupKeyShare: secret,
		AnonymousSigningKeys: VerificationKeyTable{
			keys:  make(map[[32]byte]bool),
			count: 0,
		},
		synchronizer:  synchronizer,
		FinalMessages: make([][]byte, 0),
	}
}

func (c *Checkpoint) HandleCheckpointMessage(metadata *messages.Metadata, message []byte, response []byte) error {
	cm := CheckpointInfo{}
	err := cm.InterpretFrom(message)
	if err != nil {
		return err
	}
	if !c.commonState.CombinedKey.VerifyMessage(&cm.Token, common.TokenContent(cm.AnonymousVerificationKey, c.commonState.NumLayers, c.commonState.NumLayers, metadata.Sender)) {
		return errors.TokenInvalid()
	}
	pt, err := cm.AnonymousVerificationKey.ToCurvePoint()
	if err != nil {
		return errors.BadElementError()
	}
	c.AnonymousSigningKeys.Add(cm.AnonymousVerificationKey)
	r := CheckpointResponse{}
	r.PartialKey = c.groupKeyShare.Mul(pt)
	r.PublicKey = cm.AnonymousVerificationKey.LookupKey()
	r.PackTo(response)

	return nil
}

func (s *VerificationKeyTable) Add(key crypto.VerificationKey) {
	buf := [crypto.VERIFICATION_KEY_SIZE]byte{}
	copy(buf[:], key)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[buf] = false
}

func (s *VerificationKeyTable) GetAndMark(key crypto.VerificationKey) error {
	buf := [crypto.VERIFICATION_KEY_SIZE]byte{}
	copy(buf[:], key)
	s.mu.Lock()
	defer s.mu.Unlock()
	used, exists := s.keys[buf]
	if !exists {
		return errors.KeyNotFound()
	}
	if used {
		return errors.Duplicate()
	}
	s.keys[buf] = true
	s.count += 1
	return nil
}

func (s *VerificationKeyTable) ResetSignatureMarking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.keys {
		s.keys[k] = false
	}
	s.count = 0
}

func (c *Checkpoint) HandleTrusteeMessage(metadata *messages.Metadata, message []byte) error {
	// err := c.synchronizer.SyncOnce(int(metadata.Layer), int(metadata.Sender))

	fm := common.FinalLightningMessage{}
	err := fm.InterpretFrom(message)
	if err != nil {
		return err
	}
	err = c.AnonymousSigningKeys.GetAndMark(fm.AnonymousVerificationKey)
	if err != nil {
		return err
	}
	if !crypto.VerifyMessage(fm.AnonymousVerificationKey, fm.Message, fm.Signature) {
		return errors.DecryptionFailure()
	}
	c.mu.Lock()
	c.FinalMessages = append(c.FinalMessages, fm.Message)
	c.mu.Unlock()

	return nil
}

func (c *Checkpoint) AllSignaturesAccountedFor() bool {
	return c.AnonymousSigningKeys.count == len(c.AnonymousSigningKeys.keys)
}
