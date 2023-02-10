package common

import (
	"crypto/rand"
	"encoding/binary"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/network/messages"
)

type CommonState struct {
	MyId int

	Round                   int
	Layer                   int
	NumLayers               int
	NumServers              int
	BinSize                 int
	GroupBinSize            int
	BoomerangLimit          int
	PathMessageLengths      []int
	OnionMessageLengths     []int
	BoomerangMessageLengths []int

	Configs         map[int64]*config.Server
	GroupConfigs    *config.Groups
	NumGroups       int
	MasterGroupSize int

	VerificationKeys         []crypto.VerificationKey          // sign messages along links
	ExpandedVerificationKeys []*crypto.ExpandedVerificationKey // faster verfication
	SecretSigningKey         crypto.SigningKey                 // my signing secret

	ServerPublicKeys []crypto.DHPublicKey // server authenticated encryption public keys
	ServerSecretKey  crypto.DHPrivateKey  // corresponding to the public diffie helman key parts for each server

	CombinedKey *token.TokenPublicKey // public key shared by all anytrust groups
	// PublicGroupKeys [][]*token.TokenPublicKey // group, server, used for signing tokens

	GroupPublicKey crypto.DHPublicKey // public key shared by all anytrust groups
	// a different secret is held for each group this server is a member of, in checkpoint.go

	RevokedKeys []map[crypto.DHPublicKey]bool // user keys revoked at each layer

	Shufflers []*config.Shuffler
}

func NewCommonState(configs map[int64]*config.Server, myId int64, groups *config.Groups) *CommonState {
	c := &CommonState{
		MyId:    int(myId),
		Configs: configs,

		Layer:      0,
		NumServers: len(configs),

		GroupConfigs:    groups,
		NumGroups:       len(groups.Groups),
		MasterGroupSize: len(groups.Groups[config.MASTER_GROUP].Servers),

		// public signatures on links
		VerificationKeys:         make([]crypto.VerificationKey, len(configs)),
		ExpandedVerificationKeys: make([]*crypto.ExpandedVerificationKey, len(configs)),
		SecretSigningKey:         configs[myId].SignatureKey,
		// public keys for authenticated encryption
		ServerPublicKeys: make([]crypto.DHPublicKey, len(configs)),

		Shufflers: make([]*config.Shuffler, len(configs)),
	}

	for i := range c.VerificationKeys {
		c.VerificationKeys[i] = configs[int64(i)].VerificationKey
		var err error
		c.ExpandedVerificationKeys[i], err = c.VerificationKeys[i].ExpandKey()
		if err != nil {
			panic(err)
		}
	}

	err := c.ServerSecretKey.InterpretFrom(configs[myId].PrivateKey)
	if err != nil {
		panic("Bad config")
	}
	for i := range c.ServerPublicKeys {
		err := c.ServerPublicKeys[i].InterpretFrom(configs[int64(i)].PublicKey)
		if err != nil {
			panic("Bad config")
		}
	}
	for i := range c.Shufflers {
		c.Shufflers[i] = config.NewPRGShuffler(rand.Reader)
	}
	return c
}

func (c *CommonState) Sign(m *messages.SignedMessage) {
	SignMessage(c.SecretSigningKey, m)
}

func (c *CommonState) Verify(m *messages.SignedMessage) bool {
	return crypto.VerifyExpanded(c.ExpandedVerificationKeys[m.Sender], m.GetSignedData(), m.Signature)
}

func (c *CommonState) IsRevoked(layer int, key *crypto.DHPublicKey) bool {
	return c.RevokedKeys[layer][*key]
}

func NewMockCommonStates(n int, template *CommonState) []*CommonState {
	// TODO: for testing only; add fields as necessary
	// return common states whose signatures are consistent
	// return only one group?
	states := make([]*CommonState, n)
	publicSignatureKeys := make([]crypto.VerificationKey, n)
	authPublicKeys := make([]crypto.DHPublicKey, n)
	if template == nil {
		template = &CommonState{NumServers: n}
	}
	for i := range states {
		verifyKey, signingKey := crypto.NewSigningKeyPair()
		privateKey, publicKey := crypto.NewDHKeyPair()
		publicSignatureKeys[i] = verifyKey
		authPublicKeys[i] = publicKey
		states[i] = &CommonState{}
		*states[i] = *template
		states[i].MyId = i
		states[i].VerificationKeys = publicSignatureKeys
		states[i].SecretSigningKey = signingKey
		states[i].ServerPublicKeys = authPublicKeys
		states[i].ServerSecretKey = privateKey
	}
	return states
}

func (c *CommonState) HashToGroup(hash *[token.HASH_SIZE]byte) uint64 {
	return binary.LittleEndian.Uint64(hash[:]) % uint64(c.NumGroups)
}

func (c *CommonState) HashToServer(hash *[token.HASH_SIZE]byte) uint64 {
	return binary.LittleEndian.Uint64(hash[:]) % uint64(c.NumServers)
}
