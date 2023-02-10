package server

// Call specific handlers based on enums
import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
)

type Handlers struct {
	messages.UnimplementedMessageHandlersServer
	round        int
	wait         *sync.Cond
	mu           sync.RWMutex
	errorHandler func(error)
	s            *Server
}

func DefaultErrorHandler(err error) {
	// Ideally run some kind of blame protocol
	// Might be a better idea to handle in errors package and recover where error is thrown
	// (Have error functions take appropriate arguments to fix error)
	panic(err)
}

func NewHandler() *Handlers {
	h := &Handlers{round: synchronization.Blocked, errorHandler: DefaultErrorHandler}
	h.wait = sync.NewCond(h.mu.RLocker())
	return h
}

func (h *Handlers) SetServer(s *Server) {
	h.s = s
}

func (h *Handlers) SetRound(round int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.round = round
	h.wait.Broadcast()
}

func (h *Handlers) WaitForRound(round int) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for (round > h.round) || (h.round == synchronization.Blocked) {
		h.wait.Wait()
		if round < h.round {
			return errors.BadMetadataError()
		}
	}
	return nil
}

// called by clients
func (h *Handlers) HandleSignedMessage(_ context.Context, m *messages.NetworkMessage) (*messages.NetworkMessage, error) {
	message := messages.ParseSignedMessage(m)
	if message == nil {
		return nil, errors.BadMetadataError()
	}
	err := h.WaitForRound(message.Round)
	if err != nil {
		return nil, err
	}
	groupOp, exists := h.checkForGroup(message.Type, message.Group)
	if groupOp && !exists {
		h.errorHandler(errors.WrongServerError())
	}
	var response *messages.SignedMessage = nil
	switch message.Type {
	// Receive a share of a key
	case messages.NetworkMessage_KeySharePush:
		response, err = nil, h.s.GroupAliases[message.Group].keyExchange[message.Layer].ReceiveKeyShare(message)
	case messages.NetworkMessage_ClientRegister:
		response, err = nil, h.s.GroupAliases[message.Group].messagePreparer.RegisterClient(message)
		// Request token signing from servers
	case messages.NetworkMessage_ClientTokenRequest:
		response, err = h.s.GroupAliases[message.Group].messagePreparer.HandleTokenRequest(message)
		// Submit a message for the next round
	case messages.NetworkMessage_ClientMessageSubmission:
		// TODO: Use anytrust group to check this signature
		// Also if this is too large, will also need to read from stream
		response, err = h.s.HandleSubmissionMessage(message)
	case messages.NetworkMessage_ClientGetReceipt:
		response, err = h.s.GetReceipt(message)
	default:
		err = errors.UnrecognizedError()
	}
	if err != nil {
		return nil, err
	}
	if response == nil {
		return &messages.NetworkMessage{}, nil
	} else {
		return response.AsNetworkMessage(), nil
	}

}

func (h *Handlers) HandleTcpStream(c *network.ConnectionManager, source int, vk *crypto.ExpandedVerificationKey) {
	for {
		metadata, raw, err := c.ReadMetadata(source)
		if err != nil {
			if err == io.EOF {
				break
			}
			h.errorHandler(errors.NetworkError(err))
		}
		config.LogTime("Received message: %v", metadata)
		err = h.WaitForRound(metadata.Round)
		start := time.Now()
		if err != nil {
			h.errorHandler(err)
		}
		groupOp, exists := h.checkForGroup(metadata.Type, metadata.Group)
		if groupOp && !exists {
			h.errorHandler(errors.WrongServerError())
		}
		stream := h.s.ReadStream(metadata, c.IncomingConnections[source])
		go stream.ContinuousReader(metadata)
		if !groupOp {
			err = h.s.WorkerPoolProcessStream(metadata, raw, stream)
		} else {
			err = h.s.WorkerPoolProcessGroup(metadata, raw, stream)
		}
		if err != nil {
			h.errorHandler(err)
		}
		config.LogTime("Processed message: %v %v", metadata, time.Since(start))
	}
}

func (h *Handlers) checkForGroup(t messages.NetworkMessage_MessageType, group int32) (bool, bool) {
	if t == messages.NetworkMessage_ClientRegister ||
		t == messages.NetworkMessage_ClientTokenRequest ||
		t == messages.NetworkMessage_GroupCheckpointToken ||
		t == messages.NetworkMessage_GroupCheckpointSignature {
		_, exists := h.s.GroupAliases[group]
		return true, exists
	}
	return false, true
}

func (h *Handlers) HealthCheck(_ context.Context, _ *messages.NetworkMessage) (*messages.NetworkMessage, error) {
	m := &messages.NetworkMessage{Data: make([]byte, 64)}
	return m, nil
}

// to skip path establishment and just test lightning
func (h *Handlers) SkipPathGen(_ context.Context, k *messages.SkipPathGenMessage) (*messages.NetworkMessage, error) {
	if k.Group >= 0 {
		// anonymous verification key
		verKey := crypto.VerificationKey{}
		verKey.InterpretFrom(k.SendingKey)
		h.s.GroupAliases[k.Group].CheckpointState.AnonymousSigningKeys.Add(verKey)
	} else {
		sendingKey := crypto.VerificationKey{}
		forwardingKey := crypto.VerificationKey{}
		sendingKey.InterpretFrom(k.SendingKey)
		forwardingKey.InterpretFrom(k.ForwardKey)
		p, err := sendingKey.ToCurvePoint()
		if err != nil {
			return nil, errors.BadElementError()
		}
		sharedKey := h.s.CommonState.ServerSecretKey.SharedKey(p)
		h.s.Keys[k.Layer].AddKey(sendingKey, sharedKey, int(k.SendingServer), int(k.ForwardingServer), forwardingKey)
	}
	return &messages.NetworkMessage{}, nil
}
