package common

import (
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/network/messages"

	"github.com/simonlangowski/lightning1/crypto"
)

// See shuffleMessages.go as well
type LightningEnvelope struct {
	Key              crypto.LookupKey
	SignedCiphertext []byte
	raw              []byte
}

type PathEstablishmentEnvelope struct {
	InKey   crypto.VerificationKey
	InToken token.SignedToken // token signs round||layer||sending-server||InKey
	// Signs round||layer||this-server||ciphertext under InKey
	SignedCiphertext []byte // encryption of PathEstablishmentInfo under DHSharedKey(InKey, server key) <- this auth enc message is used in the traceback blame protocol to prove key mapping
	raw              []byte
}

// message encrypted in path establishment message
type PathEstablishmentInfo struct {
	OutKey   crypto.VerificationKey // for processing messages in reverse direction, this is the key they will be signed under
	OutToken token.SignedToken      // token signs round||layer||this-server||OutKey

	BoomerangEnvelope []byte // receipt message to be decrypted and reverse onion routed.  Add lookup key from InKey and return as LightningEnvelope
	NextEnvelope      []byte // next path establishment message.  Add key from OutKey and OutToken as InKey and InToken and send as PathEstablishmentEnvelope
	raw               []byte
}

type FinalLightningMessage struct {
	AnonymousVerificationKey crypto.VerificationKey // the final OutKey, whose token was checked by the anytrust group during path establishment
	Signature                crypto.Signature       // a signature under the key
	Message                  []byte                 // The user's message
}

// Zero is not a point on the curve
// We can do this because hidden by TLS, but isn't constant time
func (l *LightningEnvelope) IsDummy() bool {
	return l.Key == crypto.LookupKey{}
}

func ValidateSignature(key crypto.VerificationKey, m *messages.SignedMessage) bool {
	return crypto.VerifyMessage(key, m.GetSignedData(), m.Signature)
}

func SignMessage(key crypto.SigningKey, m *messages.SignedMessage) {
	m.Signature = crypto.SignData(key, m.GetSignedData())
}
