package prepareMessages

import (
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/server/common"
)

// calculate the size of dummy messages

func LightningMessageLengths(layers, payloadSize int) []int {
	lengths := OnionLengths(layers, payloadSize+common.FINAL_MESSAGE_BASE_LENGTH)
	// overhead to send key on wire, but not include inside of decryption
	for i := 0; i < layers; i++ {
		lengths[i] += crypto.KEY_SIZE
	}
	lengths[layers] += crypto.VERIFICATION_KEY_SIZE
	return lengths
}

func PathEstablishmentLengths(layers, receiptSize, limitSize int) []int {
	// boomerang is onion of reverse onion
	// layers-1 previous keys, then one layer with the group public key
	reverseLengths := BoomerangLengths(layers, receiptSize, limitSize)
	// Path establishment message with all parts
	lengths := make([]int, layers+1)
	lengths[layers] = 0
	for l := layers - 1; l >= 0; l-- {
		lengths[l] = crypto.Overhead + crypto.POINT_SIZE + token.TOKEN_SIZE + lengths[l+1] + reverseLengths[l]
	}
	// overhead to send inkey and intoken on wire, but not include inside of decryption (since its already outKey of the previous)
	for i := range lengths {
		lengths[i] += crypto.POINT_SIZE + token.TOKEN_SIZE
	}
	return lengths
}

func OnionLengths(layers, size int) []int {
	lengths := make([]int, layers+1)
	for layer := layers; layer >= 0; layer-- {
		lengths[layer] = size
		// overhead of authenticated encryption and including public key in message  (techincally the key could serve as the authentication to reduce this size)
		size += crypto.Overhead
	}
	return lengths
}

func BoomerangLengths(numLayers, size, limitSize int) []int {
	lengths := make([]int, numLayers)
	lengths[0] = size
	for layer := 1; layer <= limitSize && layer < numLayers; layer++ {
		lengths[layer] = lengths[layer-1] + crypto.Overhead
	}
	for layer := limitSize + 1; layer < numLayers; layer++ {
		lengths[layer] = lengths[layer-1]
	}
	// extra overhead for group message encryption
	lengths[numLayers-1] += crypto.Overhead
	return lengths
}

func WireBoomerangLengths(numLayers, size, limitSize int) []int {
	lengths := BoomerangLengths(numLayers, size, limitSize)
	// group just sends back the key to decrypt - never sent at this size
	lengths[numLayers-1] -= crypto.Overhead
	for i := 1; i < len(lengths); i++ {
		// overhead to send key on wire, but not include inside of decryption
		lengths[i] += crypto.KEY_SIZE
	}
	return lengths
}

// write test cases to ensure the above lengths are correct
