package processMessages

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/checkpoint"
	"github.com/simonlangowski/lightning1/server/common"
)

type PathEstablishmentParser struct {
	c               *common.CommonState
	table           *KeyLookupTable
	layer           int
	OutgoingBuffers map[int]*buffers.MemReadWriter
	// It might be a lot more efficient to only send to one group (e.g a group containing this server).  It depends on how well the groups are balance
	// But then the signature only has to be checked by one group instead of many groups
	Checkpoint *CheckpointSender
}

func NewPathEstablishmentParser(c *common.CommonState, table *KeyLookupTable, layer int, checkpoint *CheckpointSender) *PathEstablishmentParser {
	p := &PathEstablishmentParser{
		c:               c,
		table:           table,
		layer:           layer,
		OutgoingBuffers: make(map[int]*buffers.MemReadWriter),
		Checkpoint:      checkpoint,
	}
	if checkpoint == nil {
		for i := 0; i < c.NumServers; i++ {
			p.OutgoingBuffers[i] = buffers.NewMemReadWriter(c.PathMessageLengths[layer+1], c.BinSize, c.Shufflers[i])
		}
	}
	return p
}

func (p *PathEstablishmentParser) ParseRecordAndGetNext(metadata *messages.Metadata, message []byte) ([]byte, *BootstrapKey, error) {
	boomerangLength := p.c.BoomerangMessageLengths[p.layer]
	round := p.c.Round
	layer := p.c.Layer
	server := p.c.MyId
	nonce := crypto.Nonce(round, layer, server)
	source := metadata.Sender

	pm := common.PathEstablishmentEnvelope{}
	err := pm.InterpretFrom(message)
	if err != nil {
		return nil, nil, err
	}

	// TODO: Batch verification of tokens?
	if !VerifyToken(p.c.CombinedKey, &pm.InToken, p.c.Round, p.c.Round, source, pm.InKey) {
		return nil, nil, errors.TokenInvalid()
	}
	tokenHash := pm.InToken.Hash()
	if p.c.HashToServer(&tokenHash) != uint64(p.c.MyId) {
		return nil, nil, errors.WrongServerError()
	}

	inPoint, err := pm.InKey.ToCurvePoint()
	if err != nil {
		return nil, nil, errors.BadElementError()
	}

	sharedKey := p.table.secretKey.SharedKey(inPoint)
	if !crypto.Verify(pm.InKey, pm.GetSignedData(round, layer, server), pm.ReadSignature()) {
		return nil, nil, errors.DecryptionFailure()
	}
	decrypted := crypto.SecretOpen(pm.SignedCiphertext, &nonce, sharedKey)
	pi := common.PathEstablishmentInfo{}
	err = pi.InterpretFrom(decrypted, boomerangLength)
	if err != nil {
		return nil, nil, err
	}
	if !VerifyToken(p.c.CombinedKey, &pi.OutToken, p.c.Round+1, p.c.Round+1, p.c.MyId, pi.OutKey) {
		return nil, nil, errors.TokenInvalid()
	}
	tokenHash = pi.OutToken.Hash()
	next := int(p.c.HashToServer(&tokenHash))
	key, err := p.table.AddKey(pm.InKey, sharedKey, source, next, pi.OutKey)
	if err != nil {
		return nil, nil, errors.BadElementError()
	}
	if p.Checkpoint == nil {
		nextMessage := common.PathEstablishmentEnvelope{
			InKey:            pi.OutKey,
			InToken:          pi.OutToken,
			SignedCiphertext: pi.NextEnvelope,
		}
		err = p.OutgoingBuffers[next].Write(nextMessage.Marshal())
	} else {
		cm := checkpoint.CheckpointInfo{
			Token:                    pi.OutToken,
			AnonymousVerificationKey: pi.OutKey,
		}
		group := int(p.c.HashToGroup(&tokenHash))
		key.NextServer = group
		// also check signature of message to be decrypted by anytrust group
		signedData := pi.GetSignedData(round, layer, server)
		signature := pi.GetSignature()
		errors.DebugPrint("Verifying %v on %v with %v", pi.GetSignature(), pi.GetSignedData(p.c.NumLayers, p.c.NumLayers, server), cm.AnonymousVerificationKey.PublicKey())
		if !crypto.Verify(cm.AnonymousVerificationKey, signedData, signature) {
			return nil, nil, errors.DecryptionFailure()
		}
		err = p.Checkpoint.AddReverseMessage(pi.BoomerangEnvelope, &cm, key, group)
	}
	if err != nil {
		return nil, nil, err
	}

	return pi.BoomerangEnvelope, key, nil
}

func VerifyToken(pk *token.TokenPublicKey, t *token.SignedToken, round, layer, server int, key crypto.VerificationKey) bool {
	tokenContent := common.TokenContent(key, round, layer, server)
	return pk.VerifyMessage(t, tokenContent)
}
