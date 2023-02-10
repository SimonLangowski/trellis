package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime/pprof"
	"sync"

	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/crypto/pairing/mcl"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/network/synchronization"
	"github.com/simonlangowski/lightning1/server/checkpoint"
	"github.com/simonlangowski/lightning1/server/common"
	"github.com/simonlangowski/lightning1/server/prepareMessages"
	"github.com/simonlangowski/lightning1/server/processMessages"
)

type Server struct {
	GroupAliases   map[int32]*groupMember
	CommonState    *common.CommonState // tracks round and layer
	Caller         *network.Caller
	Keys           []*processMessages.KeyLookupTable
	synchronizer   *synchronization.Synchronizer
	TcpConnections *network.ConnectionManager

	// output of onion parser is processed differently depending on layer
	onionParsers []*processMessages.OnionParser

	// last layer
	lastLayer   int
	finalRouter *processMessages.TrusteeRouter
	// path establishment boomerangs
	receiptLayer int
	receipts     map[int64][]byte
	// middle layers
	lightingRouters []*processMessages.LightningRouter
	// process path establishment messages
	pathEstablishmentRouters []*processMessages.PathEstablishmentParser
	pathRound                bool
	pathLayer                int
	direction                int

	pool            *WorkPool
	handler         *Handlers
	isRoundComplete bool
	roundComplete   *sync.Cond
	mu              sync.RWMutex
	receiptLock     sync.Mutex
	started         bool
	coord.UnimplementedCoordinatorHandlerServer
}

func NewServer(configs *config.Servers, groups *config.Groups, handler *Handlers, addr string) *Server {
	myId, _ := network.FindConfig(addr, configs.Servers)
	s := &Server{
		GroupAliases: make(map[int32]*groupMember),
		CommonState:  common.NewCommonState(configs.Servers, myId, groups),
		Keys:         make([]*processMessages.KeyLookupTable, 0),
		handler:      handler,
	}
	config.InitLogger(s.CommonState.MyId)
	for gid, cfg := range groups.Groups {
		for _, sid := range cfg.Servers {
			if sid == myId {
				s.GroupAliases[int32(gid)] = NewGroupMember(int(gid), s.CommonState)
				break
			}
		}
	}
	s.roundComplete = sync.NewCond(s.mu.RLocker())
	s.TcpConnections = network.NewConnectionManager(s.CommonState.Configs, s.CommonState.MyId)
	handler.SetServer(s)
	s.pool = NewWorkPool(handler, s)
	return s
}

// connect to other servers
func (s *Server) Connect() error {
	var err error
	s.Caller, err = network.NewCaller(s.CommonState.Configs)
	if err != nil {
		return err
	}
	s.Caller.SetGroups(s.CommonState.GroupConfigs.Groups)
	s.Caller.HealthCheck()
	s.TcpConnections.SetCaller(s.Caller)
	s.TcpConnections.LaunchConnects()
	return nil
}

// process messages received directly from clients
func (s *Server) HandleSubmissionMessage(m *messages.SignedMessage) (*messages.SignedMessage, error) {
	// Signature will be checked in signed encryption.
	s.synchronizer.Sync(0)
	if s.pathRound {
		return nil, s.handlePathMessage(&m.Metadata, m.Data)
	} else {
		return nil, s.handleLightningMessage(&m.Metadata, m.Data)
	}
}

/*
// block as well
func (s *Server) GetCurrentMessageLength(m *messages.Metadata) (int, error) {
	if m.Layer < 0 {
		return 0, errors.BadMetadataError()
	}

	// mark that we have received a message from this server for this layer
	// wait if we are not at the layer yet
	err := s.synchronizer.SyncOnce(m.Layer, m.Sender)
	if err != nil {
		return 0, err
	}
	switch m.Type {
	case messages.NetworkMessage_ServerMessageForward:
		return s.CommonState.OnionMessageLengths[m.Layer], nil
	case messages.NetworkMessage_PathMessageForward:
		return s.CommonState.PathMessageLengths[m.Layer], nil
	case messages.NetworkMessage_ServerMessageReverse:
		return s.CommonState.BoomerangMessageLengths[m.Layer], nil
	case messages.NetworkMessage_GroupCheckpointToken:
		return checkpoint.TOKEN_MESSAGE_LENGTH, nil
	case messages.NetworkMessage_GroupCheckpointSignature:
		return s.CommonState.OnionMessageLengths[m.Layer], nil
	default:
		return 0, errors.UnrecognizedError()
	}
}
*/

// common functions to check and synchronize metadata
func (s *Server) checkMessage(m *messages.Metadata) error {
	if m.Layer < 0 {
		return errors.BadMetadataError()
	}

	// mark that we have received a message from this server for this layer
	// wait if we are not at the layer yet
	err := s.synchronizer.SyncOnce(m.Layer, m.Sender)
	if err != nil {
		return err
	}
	return nil
}

// parse broadcast round envelopes, decrypt, mark keys used, and pack in buffers for next layer
func (s *Server) handleLightningMessage(m *messages.Metadata, message []byte) error {
	layer := s.CommonState.Layer
	decryption, key, err := s.onionParsers[layer].AuthenticatedOnionParse(m, message)
	if err != nil {
		return err
	}
	if layer != s.lastLayer {
		return s.lightingRouters[layer].AuthenticatedOnionPack(decryption, key, false)
	} else {
		return s.finalRouter.Pack(decryption, key)
	}
}

// Parse path establishment message, check tokens, record keys, and pack boomerang messages
func (s *Server) handlePathMessage(m *messages.Metadata, message []byte) error {
	layer := s.CommonState.Layer
	boomerangMessage, key, err := s.pathEstablishmentRouters[layer].ParseRecordAndGetNext(m, message)
	if err != nil {
		return err
	}

	if layer == 0 {
		s.recordReceipts(boomerangMessage, key)
	} else if layer != s.lastLayer {
		return s.lightingRouters[layer].AuthenticatedOnionPack(boomerangMessage, key, true)
	} else {
		// first need to route boomerang messages through anytrust groups - done in onThreshold once all messages have been received
	}
	return nil
}

// Parse boomerang messages, decrypt, mark keys used, and pack in buffers for next layer
func (s *Server) HandleBoomerangMessage(m *messages.Metadata, message []byte) error {
	layer := s.CommonState.Layer
	decryption, key, err := s.onionParsers[layer].AuthenticatedOnionParse(m, message)
	if err != nil {
		return err
	}
	if layer > s.receiptLayer {
		err = s.lightingRouters[layer].AuthenticatedOnionPack(decryption, key, true)
	} else if layer == 0 {
		s.recordReceipts(decryption, key)
	} else if layer == s.receiptLayer {
		// the receipt has passed through at least one honest server with high probability and no server has complained
		// so we can stop passing the receipt backwards
		// no one would actually check these because that would break anonymity
		// these are just stored/checked for test purposes
		s.receiptLock.Lock()
		defer s.receiptLock.Unlock()
		s.receipts[int64(len(s.receipts))] = decryption
	}
	return err
}

func (s *Server) recordReceipts(decryption []byte, key *processMessages.BootstrapKey) {
	// record receipts for client to check
	clientId := key.PrevServer
	s.receiptLock.Lock()
	defer s.receiptLock.Unlock()
	s.receipts[int64(clientId)] = decryption
}

// Called after "synchronizer.Done" is called by the rpc from each server
// This code handles the sending of envelopes for the next round
func (s *Server) OnThreshold(layer int) (int, int) {
	config.LogTime("Finished layer %d", layer)
	s.mu.Lock()
	if layer != s.pathLayer {
		if !s.onionParsers[layer].AllKeysAccountedFor() {
			panic(errors.MissingMessages())
		}
	}
	// setup next layer
	nextLayer := layer + s.direction
	// track layer
	s.CommonState.Layer = nextLayer

	// allocate resources needed for next layer
	if !s.pathRound && layer != s.lastLayer {
		s.onionParsers[nextLayer] = processMessages.NewOnionParser(s.CommonState, s.Keys[nextLayer], false)
		s.lightingRouters[nextLayer] = processMessages.NewLightningRouter(s.CommonState, nextLayer, false)
	} else if s.pathRound && layer != s.receiptLayer {
		s.onionParsers[nextLayer] = processMessages.NewOnionParser(s.CommonState, s.Keys[nextLayer], true)
		s.lightingRouters[nextLayer] = processMessages.NewLightningRouter(s.CommonState, nextLayer, true)
	}
	// start sending messages to next layer
	go func(lBufs map[int]*buffers.MemReadWriter) {
		var err error = nil
		if layer == s.receiptLayer {
			// Mark round completed
			// Release receipts
			// Wait for all clients to confirm receipt delivery and then continue
			s.isRoundComplete = true
			s.roundComplete.Broadcast()
		} else {
			if layer != s.lastLayer {
				// send regular onion messages
				t := messages.NetworkMessage_ServerMessageForward
				if s.pathRound {
					// boomerang messages
					t = messages.NetworkMessage_ServerMessageReverse
				}
				err = s.TcpConnections.SendShuffleMessages(lBufs, s.CommonState, nextLayer, t)
			} else {
				if !s.pathRound {
					// send to trustees
					_, err := s.TcpConnections.SendGroupShuffleMessages(s.finalRouter.OutgoingBuffers, s.CommonState, messages.NetworkMessage_GroupCheckpointSignature, 0)
					if err != nil {
						panic(err)
					}
					s.isRoundComplete = true
					s.roundComplete.Broadcast()
				} else {
					// route through anytrust group
					checkpoint := s.pathEstablishmentRouters[layer].Checkpoint
					// this waits for all groups to respond
					err = checkpoint.SendAndRecieve(s.TcpConnections)
					if err != nil {
						panic(err)
					}
					decryptions, keys := checkpoint.GetDecrypted()
					// unless there's only one layer, this is never the receipt layer as well
					for idx := range decryptions {
						err = s.lightingRouters[layer].AuthenticatedOnionPack(decryptions[idx], keys[idx], true)
						if err != nil {
							panic(err)
						}
					}
					// send back boomerang messages
					err = s.TcpConnections.SendShuffleMessages(s.lightingRouters[layer].OutgoingBuffers, s.CommonState, s.CommonState.Layer, messages.NetworkMessage_ServerMessageReverse)
				}
			}
		}
		if err != nil {
			panic(err)
		}
		// free memory - actually needs to be stored until end of round for blame protocols (e.g on disk?)
		s.onionParsers[layer] = nil
		s.lightingRouters[layer] = nil
		// do not let next onThreshold start until this one completes
		// (go lets another thread unlock the mutex)
		s.mu.Unlock()
	}(s.lightingRouters[layer].OutgoingBuffers)
	return s.CommonState.NumServers, nextLayer
}

func (s *Server) SetupNewPathEstablishmentRound(numLayers, receipt_size, boomerangLimit int, last bool) {
	s.pathLayer = 0
	s.receiptLayer = 0
	s.receipts = make(map[int64][]byte)
	s.lastLayer = numLayers - 1
	s.pathRound = true
	s.direction = -1
	s.CommonState.PathMessageLengths = prepareMessages.PathEstablishmentLengths(numLayers, receipt_size, boomerangLimit)
	s.CommonState.BoomerangMessageLengths = prepareMessages.BoomerangLengths(numLayers, receipt_size, boomerangLimit)
	s.CommonState.OnionMessageLengths = prepareMessages.WireBoomerangLengths(numLayers, receipt_size, boomerangLimit)
	s.pathEstablishmentRouters = make([]*processMessages.PathEstablishmentParser, numLayers)
	// initalize first path establishment round
	s.pathEstablishmentRouters[0] = processMessages.NewPathEstablishmentParser(s.CommonState, s.Keys[0], 0, nil)
}

// index of last layer e.g 0 for one layer
func (s *Server) SetupNewLightningRound(numLayers, payloadSize int) {
	s.lastLayer = numLayers - 1
	s.receiptLayer = -1
	s.pathLayer = -1
	s.pathRound = false
	s.direction = 1
	s.CommonState.OnionMessageLengths = prepareMessages.LightningMessageLengths(numLayers, payloadSize)
	s.onionParsers = make([]*processMessages.OnionParser, numLayers)
	s.lightingRouters = make([]*processMessages.LightningRouter, numLayers)
	// initialize first lightning layer
	s.onionParsers[0] = processMessages.NewOnionParser(s.CommonState, s.Keys[0], false)
	s.lightingRouters[0] = processMessages.NewLightningRouter(s.CommonState, 0, false)
	s.finalRouter = processMessages.NewTrusteeRouter(s.CommonState, s.lastLayer+1)
	for _, g := range s.GroupAliases {
		g.NewLightningRound(s.lastLayer + 1)
	}
}

// func (s *Server) DoKeyExchange(c *network.Caller) error {
// 	done := make(chan error)
// 	for _, g := range s.groupAliases {
// 		go func(g *groupMember) {
// 			err := g.ExchangeKeys(c)
// 			if err != nil {
// 				done <- err
// 			}
// 			done <- nil
// 		}(g)
// 	}
// 	for range s.groupAliases {
// 		err := <-done
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	for _, g := range s.groupAliases {
// 		s.commonState.SetKeys(g.GetPublicKeys())
// 		break
// 	}
// 	return nil
// }

// func (s *Server) HandleInformationRequest() (*messages.SignedMessage, error) {
// 	i := keyExchange.InformationRequest{
// 		Round:       uint32(s.commonState.Round),
// 		NumLayers:   uint32(s.commonState.NumLayers),
// 		SigningKeys: s.commonState.VerificationKeys,
// 	}
// 	m := &messages.SignedMessage{
// 		Data: make([]byte, i.Len()),
// 	}
// 	i.PackTo(m.Data)
// 	return m, nil
// }

func (s *Server) RoundSetup(_ context.Context, m *coord.RoundInfo) (*coord.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Caller == nil {
		err := s.Connect()
		if err != nil {
			return nil, err
		}
	}
	if m.Round == 0 && m.Interval > 0 {
		errors.MonitorMemory("server", s.CommonState.MyId, m.Interval)
	}
	s.CommonState.Round = int(m.Round)
	s.CommonState.BinSize = int(m.BinSize)
	// TODO: chernoff on M messages / n * numGroups (rather than n * n * L for regular bin size)
	s.CommonState.GroupBinSize = int(m.BinSize) * s.CommonState.NumServers
	s.CommonState.NumLayers = int(m.NumLayers)
	s.CommonState.Layer = 0
	s.isRoundComplete = false
	s.synchronizer = synchronization.NewSynchronizer(s.CommonState.Round, 0, s.CommonState.NumServers, s)
	numLayers := int(m.NumLayers)
	if m.Round == 0 {
		s.Keys = make([]*processMessages.KeyLookupTable, numLayers)
		for i := range s.Keys {
			s.Keys[i] = processMessages.NewKeyLookupTable(s.CommonState)
		}
		s.onionParsers = make([]*processMessages.OnionParser, numLayers)
		s.lightingRouters = make([]*processMessages.LightningRouter, numLayers)
	}
	if m.PathEstablishment {
		s.SetupNewPathEstablishmentRound(int(m.NumLayers), int(m.MessageSize), int(m.BoomerangLimit), m.LastLayer)
	} else {
		s.SetupNewLightningRound(int(m.NumLayers), int(m.MessageSize))
	}
	// this will allow processing of messages for this round
	s.handler.SetRound(s.CommonState.Round)
	// log.Printf("%d: round setup", s.CommonState.MyId)
	return &coord.Empty{}, nil
}

// I think this function could wait for all of the messages to be sent and for the round to complete
// Then check could just skip getmessages and it would be much simpler
func (s *Server) RoundStart(_ context.Context, m *coord.RoundInfo) (*coord.Empty, error) {
	// path establishment -> forward one layer -> send to group -> boomerang back send receipts
	// coordinator asks clients to check receipts and then calls this again
	// broadcast round -> forward messages through all layers -> send to trustees
	if m.Interval == -1 {
		f, err := os.Create(fmt.Sprintf("%dRound%d.pprof", s.CommonState.MyId, m.Round))
		if err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}
	if !s.started {
		s.started = true
		for sid := range s.CommonState.Configs {
			go s.handler.HandleTcpStream(s.TcpConnections, int(sid), s.CommonState.ExpandedVerificationKeys[sid])
		}
	}

	if s.pathRound {
		s.mu.Lock()
		s.CommonState.Round = int(m.Round)
		s.pathLayer = int(m.NextLayer)
		s.CommonState.Layer = s.pathLayer
		s.isRoundComplete = false
		startingLayer := s.pathLayer - 1
		s.receiptLayer = int(m.ReceiptLayer)
		s.receipts = make(map[int64][]byte)
		checkpoint := (*processMessages.CheckpointSender)(nil)
		if s.pathLayer == s.lastLayer {
			checkpoint = processMessages.NewCheckpointSender(s.CommonState, s.pathLayer)
		}
		s.pathEstablishmentRouters[s.pathLayer] = processMessages.NewPathEstablishmentParser(s.CommonState, s.Keys[s.pathLayer], s.pathLayer, checkpoint)

		// this only works iteratively
		if s.pathLayer-int(m.BoomerangLimit) > 0 {
			for i := len(s.CommonState.OnionMessageLengths) - 1; i > 0; i-- {
				s.CommonState.OnionMessageLengths[i] = s.CommonState.OnionMessageLengths[i-1]
			}
		}
		s.lightingRouters[s.pathLayer] = processMessages.NewLightningRouter(s.CommonState, s.pathLayer, true)
		s.synchronizer.Reset(int(m.Round), s.pathLayer, s.CommonState.NumServers)
		// this will allow processing of messages for this round
		s.handler.SetRound(s.CommonState.Round)
		err := s.TcpConnections.SendShuffleMessages(s.pathEstablishmentRouters[startingLayer].OutgoingBuffers, s.CommonState, s.CommonState.Layer, messages.NetworkMessage_PathMessageForward)
		if err != nil {
			return nil, err
		}
		// free memory
		s.pathEstablishmentRouters[startingLayer] = nil
		s.mu.Unlock()
	} else {
		s.synchronizer.Trigger()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for !s.isRoundComplete {
		s.roundComplete.Wait()
	}
	if !s.pathRound {
		for _, g := range s.GroupAliases {
			g.GetMessages()
		}
	}
	return &coord.Empty{}, nil
}

func (s *Server) GetMessages(_ context.Context, m *coord.RoundInfo) (*coord.ServerMessages, error) {
	if s.pathRound {
		s.handler.WaitForRound(int(m.Round))
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for !s.isRoundComplete {
		s.roundComplete.Wait()
	}
	resp := &coord.ServerMessages{
		Messages: make([][]byte, 0),
	}
	if !s.pathRound {
		// retrieve the final messages that would be forwarded/posted anonymously so we can check everything is working correctly
		for _, g := range s.GroupAliases {
			messages := g.GetMessages()
			if m.Check {
				resp.Messages = append(resp.Messages, messages...)
			}
		}
	} else if m.Check {
		// retrieve intermediate receipts by the coordinator so we can check everything is working correctly
		// the ids are meaningless because they just point to each next server on the path
		// (This server doesn't know who the client is!)
		for _, r := range s.receipts {
			resp.Messages = append(resp.Messages, r)
		}
	}
	return resp, nil
}

func (s *Server) GetReceipt(m *messages.SignedMessage) (*messages.SignedMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for !s.isRoundComplete {
		s.roundComplete.Wait()
	}
	c := prepareMessages.NewClientRequest{}
	err := c.InterpretFrom(m.Data)
	if err != nil {
		return nil, err
	}
	// technically don't need to check signature or care if someone else also sees receipt?
	// if !crypto.Verify(c.VerificationKey, m.Data, m.Signature) {
	// 	return nil, errors.SignatureError()
	// }
	receipt := s.receipts[c.ID]
	if len(receipt) == 0 {
		return nil, errors.ClientNotFoundError()
	}
	resp := messages.NewSignedMessage(len(receipt), s.CommonState.Round, s.CommonState.Layer, s.CommonState.MyId, 0, 0, 1, m.Type)
	copy(resp.Data, receipt)
	s.CommonState.Sign(resp)
	return resp, nil
}

func (s *Server) KeySet(_ context.Context, info *coord.KeyInformation) (*coord.KeyInformation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// set keys for each group based on the info

	if s.CommonState.CombinedKey == nil {
		tokenPublicKey := &token.TokenPublicKey{}
		groupPublicKey := crypto.DHPublicKey{}
		err := tokenPublicKey.InterpretFrom(info.TokenPublicKey)
		if err != nil {
			return nil, err
		}
		err = groupPublicKey.InterpretFrom(info.GroupKey)
		if err != nil {
			return nil, err
		}

		s.CommonState.CombinedKey = tokenPublicKey
		s.CommonState.GroupPublicKey = groupPublicKey
	}

	g := s.GroupAliases[int32(info.GroupId)]

	if g == nil {
		return &coord.KeyInformation{}, nil
	}
	tokenShare := mcl.Fr{}
	groupShare := &crypto.DHPrivateKey{}

	err := tokenShare.InterpretFrom(info.TokenKeyShare)
	if err != nil {
		return nil, err
	}
	tokenSigningKey := token.NewTokenSigningKey(&tokenShare)
	err = groupShare.InterpretFrom(info.GroupShare)
	if err != nil {
		return nil, err
	}

	g.SetKeys(tokenSigningKey, groupShare)
	return &coord.KeyInformation{}, nil
}

func (s *Server) ReadStream(m *messages.Metadata, conn net.Conn) *network.ConnectionReader {
	numMessages := m.NumMessages
	var messageSize int
	var excludeDummies bool
	switch m.Type {
	case messages.NetworkMessage_ServerMessageForward:
		messageSize = s.CommonState.OnionMessageLengths[m.Layer]
		excludeDummies = true
	case messages.NetworkMessage_ServerMessageReverse:
		messageSize = s.CommonState.OnionMessageLengths[m.Layer+1]
		excludeDummies = true
	case messages.NetworkMessage_PathMessageForward:
		messageSize = s.CommonState.PathMessageLengths[m.Layer]
		excludeDummies = true
	case messages.NetworkMessage_GroupCheckpointToken:
		messageSize = checkpoint.TOKEN_MESSAGE_LENGTH
		excludeDummies = false
	case messages.NetworkMessage_GroupCheckpointSignature:
		messageSize = s.CommonState.OnionMessageLengths[s.CommonState.NumLayers]
		excludeDummies = false
	default:
		return nil
	}

	// if smaller than a mtu, might as well read many at once
	baseBatchSize := network.CalculateBatchSize(config.TCPReadSize, messageSize)

	return network.NewConnectionReader(int(numMessages), messageSize, baseBatchSize, excludeDummies, conn)
}
