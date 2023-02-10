package prepareMessages

import (
	"encoding/binary"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/server/common"
)

type NewClientRequest struct {
	ID              int64
	VerificationKey crypto.VerificationKey
}

type TokenRequest struct {
	ID           int64
	TokenRequest mcl.G1
}

func (t *NewClientRequest) Len() int {
	return 8 + crypto.VERIFICATION_KEY_SIZE
}
func (t *NewClientRequest) PackTo(b []byte) {
	if len(b) != t.Len() {
		panic(errors.LengthInvalidError())
	}
	binary.LittleEndian.PutUint64(b[:8], uint64(t.ID))
	t.VerificationKey.PackTo(b[8:])
}
func (t *NewClientRequest) InterpretFrom(b []byte) error {
	if len(b) != t.Len() {
		return errors.LengthInvalidError()
	}
	t.ID = int64(binary.LittleEndian.Uint64(b[:8]))
	return t.VerificationKey.InterpretFrom(b[8:])
}

func (t *TokenRequest) Len() int {
	return 8 + t.TokenRequest.Len()
}
func (t *TokenRequest) PackTo(b []byte) {
	if len(b) != t.Len() {
		panic(errors.LengthInvalidError())
	}
	binary.LittleEndian.PutUint64(b[:8], uint64(t.ID))
	t.TokenRequest.PackTo(b[8:])
}
func (t *TokenRequest) InterpretFrom(b []byte) error {
	if len(b) != t.Len() {
		return errors.LengthInvalidError()
	}
	t.ID = int64(binary.LittleEndian.Uint64(b[:8]))
	return t.TokenRequest.InterpretFrom(b[8:])
}

// for writing clients to file to test later parts of the system
type MarshallableClient struct {
	ID                 int64
	SubmissionKey      crypto.SigningKey
	VerificationKey    crypto.VerificationKey
	AnonymousPublicKey crypto.VerificationKey
	AnonymousSecretKey crypto.SigningKey
	Group              int
	PathKeys           []*PathKey
	Receipts           [][]byte
	Message            []byte
}

func (c *Client) Marshal() *MarshallableClient {
	return &MarshallableClient{
		ID:                 c.ID,
		SubmissionKey:      c.submissionKey,
		VerificationKey:    c.verificationKey,
		AnonymousPublicKey: c.AnonymousVerificationKey,
		Group:              c.group,
		PathKeys:           c.PathKeys,
		Receipts:           c.Receipts,
	}
}

func (m *MarshallableClient) Unmarshal(c *common.CommonState) *Client {
	return &Client{
		ID:                       m.ID,
		submissionKey:            m.SubmissionKey,
		verificationKey:          m.VerificationKey,
		AnonymousVerificationKey: m.AnonymousPublicKey,
		group:                    m.Group,
		PathKeys:                 m.PathKeys,
		Receipts:                 m.Receipts,

		Common:         c,
		CombinedKey:    c.CombinedKey,
		GroupPublicKey: c.GroupPublicKey,
	}
}
