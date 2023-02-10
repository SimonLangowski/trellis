package keyExchange

import (
	"crypto/rand"
	"math/big"
	"sync"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto/ec"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
)

type KeyExchange struct {
	myId        int
	layerNumber int
	master      bool

	secretSigningKey *ec.ScalarElement
	publicSigningKey *ec.Point

	synchronizer *synchronization.Synchronizer
	mu           sync.Mutex
}

func NewMasterKeyExchange(myId, layerNumber, masterGroupSize int) *KeyExchange {
	k := NewKeyExchange(myId, layerNumber, masterGroupSize)
	k.master = true
	// If master group, choose random secrets
	k.secretSigningKey, _ = ec.RandomCurveScalar(rand.Reader)
	return k
}

func NewKeyExchange(myID, layerNumber, masterGroupSize int) *KeyExchange {
	k := &KeyExchange{
		myId:        myID,
		layerNumber: layerNumber,
		master:      false,
	}
	k.secretSigningKey = ec.NewScalarElement(new(big.Int))
	k.publicSigningKey = ec.ZeroPoint()
	k.synchronizer = synchronization.NewSynchronizer(0, synchronization.InProgress, masterGroupSize, k)
	return k
}

// call in master group only
func (k *KeyExchange) StartKeyExchange(c *network.Caller) error {
	if !k.master {
		panic("Key exchange starts in master group")
	}
	done := make(chan error)
	for i := range c.Groups {
		go func(i int) {
			if i == config.MASTER_GROUP {
				// send shares of public key
				done <- k.SendPublicKeyShares(c)
			} else {
				// send shares of secret key
				done <- k.SendPrivateKeyShares(c, i)
			}
		}(i)
	}
	for range c.Groups {
		err := <-done
		if err != nil {
			return err
		}
	}
	k.synchronizer.Sync(synchronization.Completed)
	return nil
}

func (k *KeyExchange) SendPublicKeyShares(c *network.Caller) error {
	data := NewPublicKeysMessage()
	data.SigningKey = *ec.ScalarBaseMult(k.secretSigningKey)
	m := messages.NewSignedMessage(data.Len(), 0, k.layerNumber, k.myId, config.MASTER_GROUP, 0, 1, messages.NetworkMessage_KeySharePush)
	data.PackTo(m.Data)
	c.SendToGroup(config.MASTER_GROUP, m)
	return nil
}

func (k *KeyExchange) SendPrivateKeyShares(c *network.Caller, group int) error {
	// I send one share to each member of a group
	// At least one member is honest (and therefore non colluding) so my secret is not revealed
	numShares := len(c.Groups[group])
	signingShares := ec.AdditiveShares(k.secretSigningKey, numShares)
	done := make(chan error)
	for i, server := range c.Groups[group] {
		go func(i int, server int) {
			data := NewPrivateKeySharesMessage()
			data.SecretSigningShare = *signingShares[i]
			m := messages.NewSignedMessage(data.Len(), 0, k.layerNumber, k.myId, group, 0, 1, messages.NetworkMessage_KeySharePush)
			data.PackTo(m.Data)
			_, err := c.SendSignedMessage(server, m)
			done <- err
		}(i, int(server))
	}
	for range c.Groups[group] {
		err := <-done
		if err != nil {
			return err
		}
	}
	return nil
}

func (k *KeyExchange) ReceiveKeyShare(m *messages.SignedMessage) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.synchronizer.Sync(synchronization.InProgress)
	if k.master {
		// add together public key shares
		p := NewPublicKeysMessage()
		err := p.InterpretFrom(m.Data)
		if err != nil {
			return err
		}
		k.publicSigningKey.Accumulate(&p.SigningKey)
	} else {
		// add together secret key shares
		p := NewPrivateKeySharesMessage()
		err := p.InterpretFrom(m.Data)
		if err != nil {
			return err
		}
		k.secretSigningKey.Accumulate(&p.SecretSigningShare)
	}
	k.synchronizer.Done()
	return nil
}

func (k *KeyExchange) OnThreshold(step int) (int, int) {
	return 0, step + 1
}

/*
func (k *KeyExchange) RetrieveKeys(c *network.Caller) error {
	k.synchronizer.Sync(synchronization.Completed)
	i, err := MakeInformationRequest(c)
	if err != nil {
		return err
	}
	k.publicSigningKey = &i.SigningKeys[k.layerNumber]
	return nil
}

func MakeInformationRequest(c *network.Caller) (*InformationRequest, error) {
	// choose any random member of the master group to request the keys from
	masterGroupServers := c.Groups[config.MASTER_GROUP].Servers
	server := masterGroupServers[mrand.Intn(len(masterGroupServers))]
	m := &messages.SignedMessage{
		MessageType: messages.SignedMessage_InformationRequest,
	}
	result, err := c.SendSignedMessage(int(server), m)
	if err != nil {
		return nil, err
	}
	return NewInformationRequest(result.Data), nil
}
*/
func (k *KeyExchange) GetPublicKeys() *ec.Point {
	k.synchronizer.Sync(synchronization.Completed)
	return k.publicSigningKey
}
