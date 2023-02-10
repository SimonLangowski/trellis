package prepareMessages

import (
	"bytes"
	"crypto/rand"

	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/common"
)

type Client struct {
	CombinedKey              *token.TokenPublicKey // should actually be an array, with one key for each layer
	Common                   *common.CommonState   // contains server message verification keys
	ID                       int64
	submissionKey            crypto.SigningKey
	verificationKey          crypto.VerificationKey
	GroupPublicKey           crypto.DHPublicKey
	group                    int
	PathKeys                 []*PathKey
	routingKey               crypto.LookupKey
	AnonymousVerificationKey crypto.VerificationKey
	Receipts                 [][]byte
}

type PathKey struct {
	SigningKey   crypto.SigningKey
	Secret       crypto.DHPrivateKey
	ServerID     int64
	Shared       crypto.DHSharedKey
	PrevServerID int64
	PrevShared   crypto.DHSharedKey
}

func NewClient(c *common.CommonState, ID int64, group int) (*Client, error) {
	psk, ssk := crypto.NewSigningKeyPair()
	return &Client{
		ID:              ID,
		submissionKey:   ssk,
		verificationKey: psk,
		group:           group,

		// public values
		Common:         c,
		CombinedKey:    c.CombinedKey,
		GroupPublicKey: c.GroupPublicKey,
	}, nil
}

func (t *Client) RegisterClient(c *network.Caller) error {
	req := NewClientRequest{
		ID:              t.ID,
		VerificationKey: t.verificationKey,
	}
	m := messages.NewSignedMessage(req.Len(), t.Common.Round, -1, int(t.ID), t.group, 0, 1, messages.NetworkMessage_ClientRegister)
	req.PackTo(m.Data)
	common.SignMessage(t.submissionKey, m)
	_, err := c.SendToGroup(t.group, m)
	return err
}

func (t *Client) SubmitPathEstablishmentMessage(c *network.Caller, message *common.PathEstablishmentEnvelope) error {
	dest := int(t.PathKeys[0].ServerID)
	submission := messages.NewSignedMessage(message.Len(), t.Common.Round, 0, int(t.ID), t.group, dest, 1, messages.NetworkMessage_ClientMessageSubmission)
	message.PackTo(submission.Data)
	common.SignMessage(t.submissionKey, submission)
	_, err := c.SendSignedMessage(dest, submission)
	// post to public bulletin board - send to anytrust group since message is signed
	// _, err = c.SendToGroup(t.group, submission)
	return err
}

func (t *Client) MakeOptimizedPathEstablishmentMessage(c *network.Caller, numLayers, boomerangLimit int) (*common.PathEstablishmentEnvelope, [][]byte, error) {
	tokens, pks, err := t.MakeTokensAndPath(c, numLayers)
	if err != nil {
		return nil, nil, err
	}
	receipts := make([][]byte, numLayers)
	var nextEnvelope []byte = nil
	for l := numLayers - 1; l >= 0; l-- {
		pInfo := common.PathEstablishmentInfo{}
		pInfo.OutToken = *tokens[l+1]
		pInfo.OutKey = pks[l+1]
		pInfo.BoomerangEnvelope, receipts[l] = t.BoomerangBase(t.PathKeys, l, boomerangLimit)
		pInfo.NextEnvelope = nextEnvelope
		if l == numLayers-1 {
			// encrypt the final boomerang message through the anytrust group
			pInfo.BoomerangEnvelope = t.Encrypt(pInfo.BoomerangEnvelope, t.PathKeys[numLayers], numLayers, numLayers, int(t.PathKeys[numLayers].PrevServerID), true)
		}
		nextEnvelope = t.Encrypt(pInfo.Marshal(), t.PathKeys[l], l, l, int(t.PathKeys[l].ServerID), false)
	}
	pMessage := &common.PathEstablishmentEnvelope{}
	pMessage.InKey = pks[0]
	pMessage.InToken = *tokens[0]
	pMessage.SignedCiphertext = nextEnvelope
	t.Receipts = receipts
	return pMessage, receipts, nil
}

func (t *Client) MakeTokensAndPath(c *network.Caller, numLayers int) ([]*token.SignedToken, []crypto.VerificationKey, error) {
	t.PathKeys = make([]*PathKey, numLayers+1)
	publicKeys := make([]crypto.VerificationKey, numLayers+1)
	tokens := make([]*token.SignedToken, numLayers+1)
	prevServer := int(t.ID)
	for i := 0; i < numLayers; i++ {
		pk, sk := crypto.NewSigningKeyPair()
		secret, err := sk.ToScalar()
		if err != nil {
			return nil, nil, err
		}
		tokenContent := common.TokenContent(pk, i, i, prevServer)
		if config.SkipToken {
			tokens[i] = token.SkipToken(tokenContent)
		} else {
			tokens[i], err = t.GetToken(c, tokenContent, i)
		}
		if err != nil {
			return nil, nil, err
		}
		hash := tokens[i].Hash()
		nextServer := t.Common.HashToServer(&hash)
		t.PathKeys[i] = &PathKey{
			SigningKey: sk,
			Secret:     *secret,
			ServerID:   int64(nextServer),
			Shared:     secret.SharedKey(&t.Common.ServerPublicKeys[nextServer]),
		}
		if i != 0 {
			t.PathKeys[i].PrevServerID = int64(prevServer)
			t.PathKeys[i].PrevShared = secret.SharedKey(&t.Common.ServerPublicKeys[prevServer])
		}
		publicKeys[i] = pk
		prevServer = int(nextServer)
	}
	// the last token has a signature key and the shared key is with the anytrust group public key
	pk, sk := crypto.NewSigningKeyPair()
	secret, _ := sk.ToScalar()
	t.PathKeys[numLayers] = &PathKey{
		Secret:       *secret,
		SigningKey:   sk,
		PrevServerID: int64(prevServer),
		PrevShared:   secret.SharedKey(&t.GroupPublicKey),
	}
	publicKeys[numLayers] = pk
	t.AnonymousVerificationKey = pk
	lastTokenContent := common.TokenContent(pk, numLayers, numLayers, prevServer)
	var err error = nil
	if config.SkipToken {
		tokens[numLayers] = token.SkipToken(lastTokenContent)
	} else {
		tokens[numLayers], err = t.GetToken(c, lastTokenContent, numLayers)
	}
	t.routingKey = publicKeys[0].LookupKey()
	return tokens, publicKeys, err
}

func (t *Client) GetToken(c *network.Caller, message []byte, layer int) (*token.SignedToken, error) {
	blindedHash, issuanceInfo := t.CombinedKey.Prepare(message)
	tr := TokenRequest{
		ID:           t.ID,
		TokenRequest: *blindedHash,
	}
	m := messages.NewSignedMessage(tr.Len(), t.Common.Round, layer, int(t.ID), t.group, 0, 1, messages.NetworkMessage_ClientTokenRequest)
	tr.PackTo(m.Data)
	common.SignMessage(t.submissionKey, m)
	responses, err := c.SendToGroup(t.group, m)
	if err != nil {
		return nil, err
	}
	partialSignatures := make([]mcl.G1, len(responses))
	for i := range partialSignatures {
		err := partialSignatures[i].InterpretFrom(responses[i].Data)
		if err != nil {
			return nil, err
		}
	}
	return issuanceInfo.Create(partialSignatures)
}

func (t *Client) BoomerangBase(currentPath []*PathKey, round, boomerangLimit int) ([]byte, []byte) {
	nonce := make([]byte, 8)
	rand.Read(nonce)
	message := nonce
	start := round - boomerangLimit
	if start < 0 {
		// the client will handle the end of the path, otherwise a server will get this
		start = 0
	}
	// reverse onion encrypt
	for layer := start; layer < round; layer++ {
		// use shared key for outgoing link
		// add to nonce to avoid reuse during signature
		message = t.Encrypt(message, currentPath[layer+1], round, layer+t.Common.NumLayers, int(currentPath[layer].ServerID), true)
	}
	return message, nonce
}

// onion encrypt the message under the path keys.
func (t *Client) OnionEncrypt(message []byte, keys []*PathKey) []byte {
	// onion encryption from last to first layer
	for layer := len(keys) - 1; layer >= 0; layer-- {
		message = t.Encrypt(message, keys[layer], t.Common.Round, layer, int(keys[layer].ServerID), false)
	}
	return message
}

func (t *Client) Encrypt(message []byte, key *PathKey, round, layer, serverId int, boomerang bool) []byte {
	nonce := crypto.Nonce(round, layer, serverId)
	var sharedKey crypto.DHSharedKey
	if !boomerang {
		sharedKey = key.Shared
	} else {
		sharedKey = key.PrevShared
	}
	m := crypto.SignedSecretSeal(message, &nonce, sharedKey, key.SigningKey)
	return m
}

func (t *Client) GetFinalMessage(numLayers int, message []byte) *common.FinalLightningMessage {
	finalMessage := &common.FinalLightningMessage{
		Message: message,
	}
	finalMessage.Signature = crypto.SignData(t.PathKeys[numLayers].SigningKey, finalMessage.MarshalSigned())
	return finalMessage
}

// Submit the message to the network in lightning round
func (t *Client) SendLightningMessage(c *network.Caller, keys []*PathKey, message []byte) error {
	finalMessage := t.GetFinalMessage(len(keys)-1, message)
	submission := common.LightningEnvelope{
		Key:              t.routingKey,
		SignedCiphertext: t.OnionEncrypt(finalMessage.MarshalI(), keys[:t.Common.NumLayers]),
	}
	submissionMessage := messages.NewSignedMessage(submission.Len(), t.Common.Round, 0, int(t.ID), t.group, 0, 1, messages.NetworkMessage_ClientMessageSubmission)
	submission.PackTo(submissionMessage.Data)
	common.SignMessage(t.submissionKey, submissionMessage)
	// Send to the first server and public bulletin board?
	_, err := c.SendSignedMessage(int(keys[0].ServerID), submissionMessage)
	// _, err = c.SendToGroup(t.group, submissionMessage)
	return err
}

func (t *Client) CheckReceipt(c *network.Caller, round int) error {
	req := NewClientRequest{}
	req.ID = t.ID
	req.VerificationKey = t.verificationKey
	m := messages.NewSignedMessage(req.Len(), t.Common.Round, -1, int(t.ID), t.group, 0, 1, messages.NetworkMessage_ClientGetReceipt)
	req.PackTo(m.Data)
	m.GetSignedData()
	// common.SignMessage(t.submissionKey, m)
	receipt, err := c.SendSignedMessage(int(t.PathKeys[0].ServerID), m)
	if err != nil {
		return err
	}
	if !t.Common.Verify(receipt) {
		return errors.SignatureError()
	}
	if !bytes.Equal(t.Receipts[round], receipt.Data) {
		return errors.WrongReceipt()
	}
	return nil
}

// to skip path generation
func (t *Client) SkipPathGen(c *network.Caller, info *coord.RoundInfo) error {
	numLayers := int(info.NumLayers)
	numServers := t.Common.NumServers
	t.PathKeys = make([]*PathKey, numLayers+1)
	publicKeys := make([]crypto.VerificationKey, numLayers+1)
	r := config.NewPRGShuffler(rand.Reader)
	// generate a random path and keys, and send to servers
	for i := 0; i < numLayers; i++ {
		pk, sk := crypto.NewSigningKeyPair()
		s, _ := sk.ToScalar()
		nextServer := r.Intn(numServers)
		t.PathKeys[i] = &PathKey{
			Secret:     *s,
			SigningKey: sk,
			ServerID:   int64(nextServer),
			Shared:     s.SharedKey(&t.Common.ServerPublicKeys[nextServer]),
		}
		publicKeys[i] = pk
	}

	group := r.Intn(t.Common.NumGroups)
	pk, sk := crypto.NewSigningKeyPair()
	s, _ := sk.ToScalar()
	t.PathKeys[numLayers] = &PathKey{
		Secret:     *s,
		SigningKey: sk,
		ServerID:   int64(group),
		Shared:     s.SharedKey(&t.GroupPublicKey),
	}
	publicKeys[numLayers] = pk
	t.AnonymousVerificationKey = pk
	t.routingKey = publicKeys[0].LookupKey()
	for l := numLayers - 1; l >= 0; l-- {
		s := &messages.SkipPathGenMessage{
			SendingKey:       publicKeys[l].Bytes(),
			Layer:            int32(l),
			ForwardingServer: int32(t.PathKeys[l+1].ServerID),
			ForwardKey:       publicKeys[l+1].Bytes(),
			Group:            -1,
		}
		if l > 0 {
			s.SendingServer = int32(t.PathKeys[l-1].ServerID)
		} else {
			s.SendingServer = int32(t.ID)
		}
		dest := t.PathKeys[l].ServerID
		err := c.SkipPathGen(s, int(dest), false)
		if err != nil {
			return err
		}
	}
	f := &messages.SkipPathGenMessage{
		Group:      int32(group),
		SendingKey: t.AnonymousVerificationKey.Bytes(),
	}
	return c.SkipPathGen(f, group, true)
}
