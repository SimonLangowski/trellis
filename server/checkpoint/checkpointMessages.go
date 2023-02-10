package checkpoint

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
)

type CheckpointInfo struct {
	// For simplicity we just send the signing key separately and don't worry about converting
	AnonymousVerificationKey crypto.VerificationKey
	Token                    token.SignedToken // signs round||layer||sending-server||PublicKey||SigningKey
}

type CheckpointResponse struct {
	PublicKey  crypto.LookupKey
	PartialKey crypto.DHPublicKey
}
