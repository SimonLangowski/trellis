package network

// Common code for rpc calls

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"sync"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/messages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Caller struct {
	Network     []messages.MessageHandlersClient
	Streams     []messages.MessageHandlers_HandleSignedMessageStreamClient
	streamLocks []sync.Mutex
	responseIds []int
	// useful for debugging (with stack traces), profiling, etc.
	mock                   bool
	mockNetwork            []messages.MessageHandlersServer
	mockStreams            []messages.MessageHandlers_HandleSignedMessageStreamClient
	Groups                 []map[int]int
	ReverseGroups          [][]int
	ServersSortedByLatency []int
}

func GetConnections(serverConfigs map[int64]*config.Server) (map[int]*grpc.ClientConn, error) {
	conn := make(map[int]*grpc.ClientConn)
	for id, s := range serverConfigs {
		pool := x509.NewCertPool()
		ok := pool.AppendCertsFromPEM(s.Identity)
		if !ok {
			panic("Could not create cert pool for TLS connection")
		}
		creds := credentials.NewClientTLSFromCert(pool, "")

		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(creds),
			grpc.WithWriteBufferSize(config.StreamSize),
			grpc.WithReadBufferSize(config.StreamSize),
			grpc.WithInitialWindowSize(int32(config.StreamSize)),
			grpc.WithInitialConnWindowSize(int32(config.StreamSize)),
		}
		a := config.IP(s.Address) + config.Port(s.Address)
		cc, err := grpc.Dial(a, opts...)
		if err != nil {
			return nil, err
		}

		conn[int(id)] = cc
	}
	return conn, nil
}

func NewCaller(serverConfigs map[int64]*config.Server) (*Caller, error) {
	conn, err := GetConnections(serverConfigs)
	if err != nil {
		return nil, err
	}
	callees := make([]messages.MessageHandlersClient, len(conn))
	for id, cc := range conn {
		callees[id] = messages.NewMessageHandlersClient(cc)
	}
	streams := make([]messages.MessageHandlers_HandleSignedMessageStreamClient, len(conn))
	for id, callee := range callees {
		streams[id], err = callee.HandleSignedMessageStream(context.Background())
		if err != nil {
			return nil, err
		}
	}
	return &Caller{mock: false,
		Network:     callees,
		Streams:     streams,
		streamLocks: make([]sync.Mutex, len(streams)),
		responseIds: make([]int, len(streams))}, nil
}

func NewMockCaller(mockNetwork []messages.MessageHandlersServer) *Caller {
	mockStreams := make([]messages.MessageHandlers_HandleSignedMessageStreamClient, len(mockNetwork))
	for i := range mockNetwork {
		mockStreams[i] = NewMockCallStream(mockNetwork[i])
	}
	return &Caller{mock: true,
		mockNetwork:            mockNetwork,
		ServersSortedByLatency: config.NewPRGShuffler(rand.Reader).Perm(len(mockNetwork)),
		mockStreams:            mockStreams,
		streamLocks:            make([]sync.Mutex, len(mockStreams)),
	}
}

func (c *Caller) SetGroups(groups map[int64]*config.Group) {
	c.Groups = make([]map[int]int, len(groups))
	numServers := 0
	for _, v := range groups {
		for _, sid := range v.Servers {
			if sid > int64(numServers) {
				numServers = int(sid)
			}
		}
	}
	c.ReverseGroups = make([][]int, numServers+1)
	for k, v := range groups {
		c.Groups[k] = make(map[int]int)
		for idx, sid := range v.Servers {
			c.Groups[k][idx] = int(sid)
			c.ReverseGroups[sid] = append(c.ReverseGroups[sid], int(k))
		}
	}
}

func (c *Caller) SendNetworkMessage(dest int, message *messages.NetworkMessage) (*messages.SignedMessage, error) {
	if !c.mock {
		resp, err := c.Network[dest].HandleSignedMessage(context.Background(), message)
		if err != nil || resp == nil {
			return nil, err
		}
		return messages.ParseSignedMessage(resp), nil
	} else {
		handler := c.mockNetwork[dest]
		resp, err := handler.HandleSignedMessage(context.Background(), copyMessage(message))
		if err != nil || resp == nil {
			return nil, err
		}
		return messages.ParseSignedMessage(copyMessage(resp)), nil
	}
}

func (c *Caller) SendSignedMessage(dest int, message *messages.SignedMessage) (*messages.SignedMessage, error) {
	return c.SendNetworkMessage(dest, message.AsNetworkMessage())
}

func (c *Caller) SendToGroup(groupNumber int, message *messages.SignedMessage) ([]*messages.SignedMessage, error) {
	group := c.Groups[groupNumber]
	responses := make([]*messages.SignedMessage, len(group))
	done := make(chan error)
	for i, dest := range group {
		go func(i int, dest int) {
			var err error
			responses[i], err = c.SendSignedMessage(dest, message)
			done <- err
		}(i, dest)
	}
	for range group {
		err := <-done
		if err != nil {
			return responses, err
		}
	}
	return responses, nil
}

func (c *Caller) UseStream(dest int) messages.MessageHandlers_HandleSignedMessageStreamClient {
	c.streamLocks[dest].Lock()
	if !c.mock {
		if c.Streams[dest] == nil {
			// create new stream
			var err error
			c.Streams[dest], err = c.Network[dest].HandleSignedMessageStream(context.Background())
			if err != nil {
				panic(errors.NetworkError(err))
			}
		}
		return c.Streams[dest]
	} else {
		return c.mockStreams[dest]
	}
}

func (c *Caller) DoneSendingStream(dest int, close bool) {
	if close && !c.mock {
		c.Streams[dest].CloseSend()
		c.Streams[dest] = nil
	}
	c.streamLocks[dest].Unlock()
}

func (c *Caller) GetStreamResponse(dest int) (*messages.SignedMessage, error) {
	if !c.mock {
		nm, err := c.Streams[dest].Recv()
		if err != nil || nm == nil {
			return nil, err
		}
		return messages.ParseSignedMessage(nm), nil
	} else {
		nm, err := c.mockStreams[dest].Recv()
		if err != nil || nm == nil {
			return nil, err
		}
		return messages.ParseSignedMessage(nm), nil
	}
}

func (c *Caller) GetJobs() chan int {
	j := make(chan int, len(c.ServersSortedByLatency))
	// longest route first to minimize total delay
	for i := len(c.ServersSortedByLatency) - 1; i >= 0; i-- {
		j <- c.ServersSortedByLatency[i]
	}
	close(j)
	return j
}

func (c *Caller) SkipPathGen(s *messages.SkipPathGenMessage, dest int, group bool) error {
	if group {
		group := c.Groups[s.Group]
		for _, dest := range group {
			err := c.SkipPathGen(s, dest, false)
			if err != nil {
				return err
			}
		}
	} else {
		if c.mock {
			c.mockNetwork[dest].SkipPathGen(context.Background(), s)
		} else {
			_, err := c.Network[dest].SkipPathGen(context.Background(), s)
			return err
		}
	}
	return nil
}
