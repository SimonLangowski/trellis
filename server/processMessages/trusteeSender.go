package processMessages

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/server/common"
)

type TrusteeRouter struct {
	c               *common.CommonState
	OutgoingBuffers map[int]*buffers.MemReadWriter
}

func NewTrusteeRouter(c *common.CommonState, layer int) *TrusteeRouter {
	t := &TrusteeRouter{
		c:               c,
		OutgoingBuffers: make(map[int]*buffers.MemReadWriter),
	}
	for i := 0; i < c.NumGroups; i++ {
		t.OutgoingBuffers[i] = buffers.NewMemReadWriter(c.OnionMessageLengths[layer], c.GroupBinSize, c.Shufflers[i])
	}
	return t
}

func (t *TrusteeRouter) Pack(decrypted []byte, destination *BootstrapKey) error {
	pm := common.FinalLightningMessage{
		AnonymousVerificationKey: destination.OutgoingVerificationKey,
		Signature:                decrypted[:crypto.SIGNATURE_SIZE],
		Message:                  decrypted[crypto.SIGNATURE_SIZE:],
	}
	err := t.OutgoingBuffers[destination.NextServer].Write(pm.Marshal())
	if err != nil {
		return err
	}
	return nil
}
