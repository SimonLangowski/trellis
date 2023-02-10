package prepareMessages

import (
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/common"

	"github.com/simonlangowski/lightning1/crypto/token"
)

type MessagePreparer struct {
	common   *common.CommonState
	mapLock  sync.RWMutex
	markLock sync.Mutex               // could have a lock in each PerClientInfo
	Clients  map[int64]*PerClientInfo // map clientID -> info
	signer   *token.TokenSigningKey
	group    int
}

type PerClientInfo struct {
	SignatureKey crypto.VerificationKey
	signed       int // don't sign twice for a client
	submitted    bool
}

func NewMessagePreparer(c *common.CommonState, signer *token.TokenSigningKey, group int) *MessagePreparer {
	return &MessagePreparer{
		common:  c,
		Clients: make(map[int64]*PerClientInfo),
		signer:  signer,
		group:   group,
	}
}

func (p *MessagePreparer) RegisterClient(m *messages.SignedMessage) error {
	n := &NewClientRequest{}
	err := n.InterpretFrom(m.Data)
	if err != nil {
		return err
	}
	// check that the user owns this signature since they signed the signature
	if !common.ValidateSignature(n.VerificationKey, m) {
		return errors.SignatureError()
	}
	p.mapLock.Lock()
	defer p.mapLock.Unlock()
	if p.Clients[n.ID] != nil {
		return errors.Duplicate()
	}
	p.Clients[n.ID] = &PerClientInfo{SignatureKey: n.VerificationKey, signed: -1, submitted: false}
	return nil
}

func (p *MessagePreparer) MarkSubmitted(ID int64, m *messages.SignedMessage) error {
	info := p.Clients[ID]
	if info == nil {
		return errors.ClientNotFoundError()
	}
	if !common.ValidateSignature(info.SignatureKey, m) {
		return errors.SignatureError()
	}
	p.markLock.Lock()
	defer p.markLock.Unlock()
	if info.submitted {
		return errors.Duplicate()
	}
	info.submitted = true
	return nil
}

func (p *MessagePreparer) RevokeClient(ID int64) {
	p.Clients[ID] = nil
}

func (p *MessagePreparer) HandleTokenRequest(m *messages.SignedMessage) (*messages.SignedMessage, error) {
	request := &TokenRequest{}
	err := request.InterpretFrom(m.Data)
	if err != nil {
		return nil, err
	}
	// sign tokens for 0..L
	if m.Layer < 0 || m.Layer > p.common.NumLayers {
		return nil, errors.BadMetadataError()
	}
	p.mapLock.RLock()
	info := p.Clients[request.ID]
	p.mapLock.RUnlock()
	if info == nil {
		return nil, errors.ClientNotFoundError()
	}
	if !common.ValidateSignature(info.SignatureKey, m) {
		return nil, errors.SignatureError()
	}
	err = p.signer.BlindSign(&request.TokenRequest, &request.TokenRequest)
	if err != nil {
		return nil, err
	}
	response := messages.NewSignedMessage(request.TokenRequest.Len(), p.common.Round, m.Layer, p.common.MyId, p.group, 0, 1, m.Type)
	request.TokenRequest.PackTo(response.Data)
	p.common.Sign(response)
	p.markLock.Lock()
	defer p.markLock.Unlock()
	if info.signed >= m.Layer {
		return nil, errors.Duplicate()
	} else {
		info.signed = m.Layer
	}
	return response, nil
}

func (p *MessagePreparer) ResetSigned() {
	for _, c := range p.Clients {
		c.signed = -1
	}
}

func (p *MessagePreparer) ResetSubmitted() {
	for _, c := range p.Clients {
		c.submitted = false
	}
}
