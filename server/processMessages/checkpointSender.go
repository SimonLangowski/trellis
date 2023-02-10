package processMessages

import (
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/checkpoint"
	"github.com/simonlangowski/lightning1/server/common"
)

type Progress struct {
	boomerang  []byte
	partialKey *crypto.DHPublicKey
	group      int
	key        *BootstrapKey
	used       bool
}

type CheckpointSender struct {
	c               *common.CommonState
	reverseMessages map[crypto.LookupKey]*Progress
	toGroupBuffers  map[int]*buffers.MemReadWriter
	mu              sync.Mutex
}

func NewCheckpointSender(c *common.CommonState, layer int) *CheckpointSender {
	s := &CheckpointSender{
		c:               c,
		reverseMessages: make(map[crypto.LookupKey]*Progress),
		toGroupBuffers:  make(map[int]*buffers.MemReadWriter),
	}
	for i := 0; i < c.NumGroups; i++ {
		s.toGroupBuffers[i] = buffers.NewMemReadWriter(checkpoint.TOKEN_MESSAGE_LENGTH, c.GroupBinSize, c.Shufflers[i])
	}
	return s
}

func (c *CheckpointSender) AddReverseMessage(boomerang []byte, info *checkpoint.CheckpointInfo, key *BootstrapKey, group int) error {
	p := &Progress{
		boomerang:  boomerang,
		partialKey: crypto.ZeroPoint(),
		used:       false,
		key:        key,
		group:      group,
	}
	err := c.toGroupBuffers[group].Write(info.Marshal())
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reverseMessages[info.AnonymousVerificationKey.LookupKey()] = p
	return nil
}

func (c *CheckpointSender) SendAndRecieve(m *network.ConnectionManager) error {
	messageCounts, err := m.SendGroupShuffleMessages(c.toGroupBuffers, c.c, messages.NetworkMessage_GroupCheckpointToken, checkpoint.RESPONSE_LENGTH)
	if err != nil {
		return err
	}
	done := make(chan error)
	groupLocks := make([]sync.Mutex, c.c.NumGroups)
	reverseGroups := m.ReverseGroups()
	groupCounts := make([]int, c.c.NumGroups)
	for gid, g := range c.c.GroupConfigs.Groups {
		groupCounts[gid] = len(g.Servers)
	}
	for sid := range reverseGroups {
		go func(sid int) {
			for _, gid := range reverseGroups[sid] {
				// expect one response for each send message
				sm, err := m.ReadReverse(sid, messageCounts[gid]*checkpoint.RESPONSE_LENGTH)
				if err != nil {
					done <- err
				}
				groupLocks[gid].Lock()
				err = c.HandleResponse(sm, gid)
				if err != nil {
					done <- err
				}
				groupCounts[gid]--
				if groupCounts[gid] == 0 {
					done <- c.DecryptGroup(gid)
				} else if groupCounts[gid] < 0 {
					panic("Group count went negative")
				}
				groupLocks[gid].Unlock()
			}
		}(sid)
	}
	for range groupCounts {
		err := <-done
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CheckpointSender) HandleResponse(sm *messages.SignedMessage, group int) error {
	if !c.c.Verify(sm) {
		return errors.SignatureError()
	}
	for pos := 0; pos < len(sm.Data); pos += checkpoint.RESPONSE_LENGTH {
		cm := checkpoint.CheckpointResponse{}
		err := cm.InterpretFrom(sm.Data[pos : pos+checkpoint.RESPONSE_LENGTH])
		if err != nil {
			return err
		}
		p := c.reverseMessages[cm.PublicKey]
		p.partialKey = p.partialKey.Accumulate(&cm.PartialKey)
	}
	return nil
}

func (c *CheckpointSender) DecryptGroup(group int) error {
	for _, s := range c.reverseMessages {
		if s.group == group {
			err := c.HandleOne(s)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CheckpointSender) HandleOne(s *Progress) error {
	nonce := crypto.Nonce(c.c.NumLayers, c.c.NumLayers, c.c.MyId)
	sharedKey := s.partialKey.AsShared()
	decrypted := crypto.SecretOpen(s.boomerang, &nonce, sharedKey)
	c.mu.Lock()
	defer c.mu.Unlock()
	if s.used {
		return errors.Duplicate()
	}
	s.used = true
	s.boomerang = decrypted
	return nil

}

// the decrypted boomerang messages
// attach the keys from path establishment
func (c *CheckpointSender) GetDecrypted() ([][]byte, []*BootstrapKey) {
	decrypted := make([][]byte, len(c.reverseMessages))
	keys := make([]*BootstrapKey, len(c.reverseMessages))
	idx := 0
	for _, p := range c.reverseMessages {
		decrypted[idx] = p.boomerang
		keys[idx] = p.key
		idx += 1
	}
	return decrypted, keys
}
