package server

import (
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
	"github.com/simonlangowski/lightning1/server/checkpoint"
	"github.com/simonlangowski/lightning1/server/common"
	"github.com/simonlangowski/lightning1/server/keyExchange"
	"github.com/simonlangowski/lightning1/server/prepareMessages"
)

// A server can be part of multiple anytrust groups
// This class represents state and functionality a server
// has on a per group basis

/*
Group based actions
signing messages
signing tokens
checkpoints
final decryption release
*/

type groupMember struct {
	c                      *common.CommonState
	CheckpointState        *checkpoint.Checkpoint
	keyExchange            []*keyExchange.KeyExchange
	messagePreparer        *prepareMessages.MessagePreparer
	signingKey             token.TokenSigningKey
	secretShare            crypto.DHPrivateKey
	checkpointSynchronizer *synchronization.Synchronizer
	myGroupNumber          int
	mu                     sync.Mutex
	messagesReady          bool
	messagesWait           *sync.Cond
}

func NewGroupMember(myGroupNumber int, common *common.CommonState) *groupMember {
	g := &groupMember{
		c:             common,
		myGroupNumber: myGroupNumber,
		// keyExchange:   make([]*keyExchange.KeyExchange, numLayers),
	}
	g.messagesWait = sync.NewCond(&g.mu)
	g.checkpointSynchronizer = synchronization.NewSynchronizer(g.c.Round, 0, g.c.NumServers, g)

	// these take pointers to the keys, whose values will be set layer
	g.CheckpointState = checkpoint.NewCheckpointState(g.c, g.myGroupNumber, &g.secretShare, g.checkpointSynchronizer)
	g.messagePreparer = prepareMessages.NewMessagePreparer(common, &g.signingKey, myGroupNumber)

	// for i := range g.keyExchange {
	// 	if g.myGroupNumber == config.MASTER_GROUP {
	// 		g.keyExchange[i] = keyExchange.NewMasterKeyExchange(i, common.MasterGroupSize)
	// 	} else {
	// 		g.keyExchange[i] = keyExchange.NewKeyExchange(i, common.MasterGroupSize)
	// 	}
	// }
	return g
}

func (g *groupMember) OnThreshold(layer int) (int, int) {
	if g.CheckpointState.AllSignaturesAccountedFor() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.messagesReady = true
		g.messagesWait.Broadcast()
	} else {
		// do blame and recovery protocol
		panic(errors.MissingMessages())
	}
	return g.c.NumServers, layer + 1
}

func (g *groupMember) NewLightningRound(checkpointLayer int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.CheckpointState.FinalMessages = make([][]byte, 0)
	g.messagesReady = false
	g.checkpointSynchronizer.Reset(g.c.Round, checkpointLayer, g.c.NumServers)
	g.CheckpointState.AnonymousSigningKeys.ResetSignatureMarking()
}

func (g *groupMember) HandleMessageSubmission(m *messages.SignedMessage) error {
	// post starting message to public bulletin board?
	return g.messagePreparer.MarkSubmitted(int64(m.Sender), m)
}

func (g *groupMember) SetKeys(t *token.TokenSigningKey, s *crypto.DHPrivateKey) {
	g.signingKey = *t
	g.secretShare = *s
}

func (g *groupMember) GetMessages() [][]byte {
	g.mu.Lock()
	defer g.mu.Unlock()
	for !g.messagesReady {
		g.messagesWait.Wait()
	}
	return g.CheckpointState.FinalMessages
}

// func (g *groupMember) ExchangeKeys(c *network.Caller) error {
// 	// could be done in parallel
// 	errs := make(chan error)
// 	for _, k := range g.keyExchange {
// 		go func(k *keyExchange.KeyExchange) {
// 			if g.myGroupNumber == config.MASTER_GROUP {
// 				errs <- k.StartKeyExchange(c)
// 			} else {
// 				errs <- k.RetrieveKeys(c)
// 			}
// 		}(k)
// 	}
// 	for range g.keyExchange {
// 		err := <-errs
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// // trustee key?
// func (g *groupMember) GetPublicKeys() ([]crypto.CurvePoint, token.TokenPublicKey) {
// 	signingKeys := make([]crypto.CurvePoint, len(g.keyExchange))
// 	for i, k := range g.keyExchange {
// 		signingKeys[i] = k.GetPublicKeys()
// 	}
// 	return signingKeys
// }
