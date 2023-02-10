package keyExchange

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto/ec"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
)

func TestSigningKeyShare(t *testing.T) {
	numShares := 5
	secrets := make([]*ec.ScalarElement, numShares)
	public := make([]*ec.Point, numShares)
	publicKey := ec.ZeroPoint()
	shares := make([][]*ec.ScalarElement, numShares)
	for i := range secrets {
		secrets[i], _ = ec.RandomCurveScalar(rand.Reader)
		shares[i] = ec.AdditiveShares(secrets[i], numShares)
		public[i] = ec.ScalarBaseMult(secrets[i])
	}
	publicKey.Accumulate(public...)

	newSecrets := make([]*ec.ScalarElement, numShares)
	newPublics := make([]*ec.Point, numShares)
	newPublic := ec.ZeroPoint()
	for i := range newSecrets {
		newSecrets[i] = ec.ZeroScalar()
		for j := range shares {
			newSecrets[i].Accumulate(shares[j][i])
		}
		newPublics[i] = ec.ScalarBaseMult(newSecrets[i])
	}
	newPublic.Accumulate(newPublics...)
	if newPublic.X.Cmp(publicKey.X) != 0 || newPublic.Y.Cmp(publicKey.Y) != 0 {
		t.Fail()
	}
}

type MockKeyHandler struct {
	k             *KeyExchange
	t             *testing.T
	expectedLayer int
	messages.UnimplementedMessageHandlersServer
}

func (m *MockKeyHandler) HandleSignedMessage(_ context.Context, raw *messages.NetworkMessage) (*messages.NetworkMessage, error) {
	message := messages.ParseSignedMessage(raw)
	if message.Layer != m.expectedLayer {
		m.t.FailNow()
	}
	if message.Type != messages.NetworkMessage_KeySharePush {
		m.t.FailNow()
	}
	return nil, m.k.ReceiveKeyShare(message)
}

func TestKeyExchange(t *testing.T) {
	masterGroupSize := 3
	otherGroupSize := 3
	layerNumber := 5
	masterGroup := make([]*KeyExchange, masterGroupSize)
	for i := range masterGroup {
		masterGroup[i] = NewMasterKeyExchange(i, layerNumber, masterGroupSize)
	}
	otherGroup := make([]*KeyExchange, otherGroupSize)
	for i := range masterGroup {
		otherGroup[i] = NewKeyExchange(i, layerNumber, masterGroupSize)
	}
	mockMasterHandlers := make([]*MockKeyHandler, masterGroupSize)
	mockOtherHandlers := make([]*MockKeyHandler, otherGroupSize)
	allHandlers := make([]messages.MessageHandlersServer, masterGroupSize+otherGroupSize)
	groups := make(map[int64]*config.Group)
	groups[0] = &config.Group{
		Gid:     0,
		Servers: make([]int64, 0),
	}
	groups[1] = &config.Group{
		Gid:     1,
		Servers: make([]int64, 0),
	}
	for i := range mockMasterHandlers {
		mockMasterHandlers[i] = &MockKeyHandler{
			k:             masterGroup[i],
			t:             t,
			expectedLayer: layerNumber,
		}
		allHandlers[i] = mockMasterHandlers[i]
		groups[0].Servers = append(groups[0].Servers, int64(i))
	}
	for i := range mockOtherHandlers {
		mockOtherHandlers[i] = &MockKeyHandler{
			k:             otherGroup[i],
			t:             t,
			expectedLayer: layerNumber,
		}
		allHandlers[i+masterGroupSize] = mockOtherHandlers[i]
		groups[1].Servers = append(groups[1].Servers, int64(i+masterGroupSize))
	}
	done := make(chan error)
	for i, k := range masterGroup {
		go func(i int, k *KeyExchange) {
			mockNetwork := network.NewMockCaller(allHandlers)
			mockNetwork.SetGroups(groups)
			done <- k.StartKeyExchange(mockNetwork)
		}(i, k)
	}
	for range masterGroup {
		err := <-done
		if err != nil {
			t.Log(err)
			t.Fail()
		}
	}
	// Check for no deadlock
	for _, k := range otherGroup {
		k.synchronizer.Sync(synchronization.Completed)
	}
	// check that the other groups secret keys correspond to the same public keys as the master group
}
