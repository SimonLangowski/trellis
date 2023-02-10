package client

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/crypto/token"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/server/common"
	"github.com/simonlangowski/lightning1/server/prepareMessages"
)

// Simulates multiple clients

const blockSize = 10000

type ClientRunner struct {
	C                 *common.CommonState
	Clients           map[int64]*prepareMessages.Client
	Caller            *network.Caller
	Sem               chan bool
	RecordMessageFile string
	idx               int
	mu                sync.Mutex
	RecordedClients   []*prepareMessages.MarshallableClient
	coord.UnimplementedCoordinatorHandlerServer
}

func NewClientRunner(servers map[int64]*config.Server, groups map[int64]*config.Group) *ClientRunner {
	return &ClientRunner{
		C:       common.NewCommonState(servers, 0, &config.Groups{Groups: groups}),
		Clients: make(map[int64]*prepareMessages.Client),
		Sem:     make(chan bool, 32*runtime.NumCPU()),
	}
}

func (c *ClientRunner) Connect() error {
	var err error
	c.Caller, err = network.NewCaller(c.C.Configs)
	if err != nil {
		return err
	}
	c.Caller.SetGroups(c.C.GroupConfigs.Groups)
	return nil
}

func (c *ClientRunner) KeySet(_ context.Context, m *coord.KeyInformation) (*coord.KeyInformation, error) {
	c.C.CombinedKey = &token.TokenPublicKey{}
	err := c.C.CombinedKey.InterpretFrom(m.TokenPublicKey)
	if err != nil {
		return nil, err
	}
	err = c.C.GroupPublicKey.InterpretFrom(m.GroupKey)
	if err != nil {
		return nil, err
	}
	return &coord.KeyInformation{}, nil
}

func (c *ClientRunner) AddClient(id int64) error {
	st := &common.CommonState{}
	*st = *c.C
	st.MyId = int(id)
	cli, err := prepareMessages.NewClient(st, id, st.MyId%c.C.NumGroups)
	if err != nil {
		return err
	}
	c.Clients[id] = cli
	return nil
}

func (c *ClientRunner) ClientStart(_ context.Context, i *coord.RoundInfo) (*coord.Empty, error) {
	if i.Round == 0 && i.Interval > 0 {
		errors.MonitorMemory("client", c.C.MyId, i.Interval)
	}
	c.C.NumLayers = int(i.NumLayers)
	c.C.Round = int(i.Round)
	c.C.BoomerangLimit = int(i.BoomerangLimit)
	for id := i.StartId; id < i.EndId; id++ {
		if c.Clients[id] == nil {
			c.AddClient(id)
		}
	}
	if !i.PathEstablishment {
		c.Sem = make(chan bool, 256*runtime.NumCPU())
	}
	done := make(chan error)
	go func() {
		for id := i.StartId; id < i.EndId; id++ {
			go func(id int64) {
				c.Acquire()
				defer c.Release()
				cli := c.Clients[id]
				cli.Common.Round = int(i.Round)
				cli.Common.NumLayers = int(i.NumLayers)
				if i.PathEstablishment {
					err := cli.RegisterClient(c.Caller)
					if err != nil {
						done <- err
					} else {
						message, _, err := cli.MakeOptimizedPathEstablishmentMessage(c.Caller, c.C.NumLayers, c.C.BoomerangLimit)
						if err != nil {
							done <- err
						} else {
							if len(c.RecordMessageFile) > 0 {
								done <- c.WriteClientToFile(cli.Marshal(), message)
							} else {
								done <- cli.SubmitPathEstablishmentMessage(c.Caller, message)
							}
						}
					}
				} else {
					if i.SkipPathGen {
						cli.SkipPathGen(c.Caller, i)
					}
					m := make([]byte, i.MessageSize)
					binary.LittleEndian.PutUint64(m, uint64(id))
					done <- cli.SendLightningMessage(c.Caller, cli.PathKeys, m)
				}
			}(id)
		}
	}()
	for id := i.StartId; id < i.EndId; id++ {
		err := <-done
		if err != nil {
			return nil, err
		}
		if id%1024 == 512 {
			log.Printf("%d/%d", id-i.StartId, (i.EndId - i.StartId))
		}
	}
	if len(c.RecordMessageFile) > 0 && i.PathEstablishment {
		c.WriteClientsToFile()
	}
	return &coord.Empty{}, nil
}

func (c *ClientRunner) CheckReceipt(_ context.Context, i *coord.RoundInfo) (*coord.Empty, error) {
	done := make(chan error)
	go func() {
		for id := i.StartId; id < i.EndId; id++ {
			go func(id int64) {
				cli := c.Clients[id]
				cli.Common.Round = int(i.Round)
				cli.Common.NumLayers = int(i.NumLayers)
				if i.PathEstablishment {
					// for receiptLayer > 0 the receipt is not retrieved
					if i.ReceiptLayer > 0 {
						panic("Retrieving receipt breaks anonymity")
					}
					err := cli.CheckReceipt(c.Caller, int(i.Round))
					done <- err
				}
			}(id)
		}
	}()
	for id := i.StartId; id < i.EndId; id++ {
		err := <-done
		if err != nil {
			return nil, err
		}
	}
	return &coord.Empty{}, nil
}

// simple go semaphore
func (c *ClientRunner) Acquire() {
	c.Sem <- true
}

func (c *ClientRunner) Release() {
	<-c.Sem
}

// ideally I could just write the whole array?
func (c *ClientRunner) WriteClientToFile(m *prepareMessages.MarshallableClient, message *common.PathEstablishmentEnvelope) error {
	m.Message = message.Marshal()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.RecordedClients = append(c.RecordedClients, m)
	if len(c.RecordedClients) > blockSize {
		c.WriteClientsToFile()
		// could also remove from client mapping... however we didn't guarantee the iteration order
		c.RecordedClients = make([]*prepareMessages.MarshallableClient, 0)
	}
	return nil
}

func (c *ClientRunner) WriteClientsToFile() {
	b, err := json.Marshal(c.RecordedClients)
	if err != nil {
		panic(err)
	}
	f, err := os.Create(fmt.Sprintf("%s%d", c.RecordMessageFile, c.idx))
	if err != nil {
		panic(err)
	}
	_, err = f.Write(b)
	if err != nil {
		panic(err)
	}
	c.idx++
}

// read each client from the file
func (c *ClientRunner) readClientsFromFile(fn string, idx int) error {
	c.RecordedClients = make([]*prepareMessages.MarshallableClient, 0)
	n := fmt.Sprintf("%s%d", fn, idx)
	b, err := ioutil.ReadFile(n)
	if err != nil {
		return err
	}
	log.Printf("Read %s", n)
	err = json.Unmarshal(b, &c.RecordedClients)
	if err != nil {
		panic(err)
	}
	return nil
}

func (c *ClientRunner) sendLoadedMessages() {
	done := make(chan error, len(c.RecordedClients))
	for _, m := range c.RecordedClients {
		st := &common.CommonState{}
		*st = *c.C
		st.MyId = int(m.ID)
		cli := m.Unmarshal(st)
		c.Clients[m.ID] = cli
		go func(message []byte) {
			pm := common.PathEstablishmentEnvelope{}
			pm.InterpretFrom(message)
			done <- cli.SubmitPathEstablishmentMessage(c.Caller, &pm)
		}(m.Message)
	}
	for range c.RecordedClients {
		err := <-done
		if err != nil {
			panic(err)
		}
	}
}

func (c *ClientRunner) SendLoadedMessages(i *coord.RoundInfo) {
	c.C.NumLayers = int(i.NumLayers)
	c.C.Round = int(i.Round)
	c.C.BoomerangLimit = int(i.BoomerangLimit)

	for {
		err := c.readClientsFromFile(c.RecordMessageFile, c.idx)
		if err != nil {
			return
		}
		c.sendLoadedMessages()
		c.idx++
	}
}
