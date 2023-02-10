package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/simonlangowski/lightning1/client"
	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server"
)

const (
	inprocess = iota // separate goroutines in this process
	local     = iota // separate processes on this machine
	remote    = iota // processes on remote machines
)

var base = 8000
var clientBase = 8900

const numRetries = 3

type CoordinatorNetwork struct {
	clientNetType int
	serverNetType int
	ServerConfigs map[int64]*config.Server
	GroupConfigs  map[int64]*config.Group
	ClientConfigs map[int64]*config.Server
	servers       []*server.Server
	clients       *client.ClientRunner
	remoteServers []coord.CoordinatorHandlerClient
	remoteClients []coord.CoordinatorHandlerClient
	processes     []*exec.Cmd
}

func NewRemoteNetwork(serverFile, groupFile, clientsFile string) *CoordinatorNetwork {
	servers, groups, clients := LoadConfigs(serverFile, groupFile, clientsFile)
	ok := TransferFileToAllServers(servers, serverFile)
	ok = ok && TransferFileToAllServers(servers, groupFile)
	ok = ok && StartRemoteServers(servers, ServerProcessName, serverFile, groupFile, clientsFile)
	if len(clientsFile) > 0 {
		// ok = ok && TransferFileToAllServers(clients, serverFile)
		// ok = ok && TransferFileToAllServers(clients, groupFile)
		ok = ok && TransferFileToAllServers(clients, clientsFile)
		ok = ok && StartRemoteServers(clients, ClientProcessName, serverFile, groupFile, clientsFile)
	}
	c := &CoordinatorNetwork{
		ServerConfigs: servers,
		GroupConfigs:  groups,
		ClientConfigs: clients,
		serverNetType: remote,
		clientNetType: remote,
	}
	if !ok {
		c.KillAll()
		panic("Error starting servers")
	}
	if len(clients) > 0 {
		c.remoteClients = c.Connect(c.ClientConfigs)
	} else {
		c.clientNetType = inprocess
		c.SetupInProcess(0)
		caller, err := network.NewCaller(c.ServerConfigs)
		if err != nil {
			c.KillAll()
			panic(err)
		}
		caller.SetGroups(c.GroupConfigs)
		c.clients.Caller = caller
	}
	c.remoteServers = c.Connect(c.ServerConfigs)
	return c
}

func LoadConfigs(serverFile, groupFile, clientsFile string) (map[int64]*config.Server, map[int64]*config.Group, map[int64]*config.Server) {
	servers, err := config.UnmarshalServersFromFile(serverFile)
	if err != nil {
		log.Fatalf("Could not read servers file %s", serverFile)
	}
	groups, err := config.UnmarshalGroupsFromFile(groupFile)
	if err != nil {
		log.Fatalf("Could not read group file %s", groupFile)
	}
	clients := make(map[int64]*config.Server)
	if len(clientsFile) > 0 {
		clients, err = config.UnmarshalServersFromFile(clientsFile)
		if err != nil {
			log.Fatalf("Could not read clients file %s", clientsFile)
		}
	}
	return servers, groups, clients
}

func NewLocalNetwork(serverConfigs map[int64]*config.Server, groupConfigs map[int64]*config.Group, clientConfigs map[int64]*config.Server) *CoordinatorNetwork {
	c := &CoordinatorNetwork{}
	c.clientNetType = local
	c.serverNetType = local
	c.ServerConfigs, c.GroupConfigs, c.ClientConfigs = serverConfigs, groupConfigs, clientConfigs
	// write configs to local file system
	err := config.MarshalServersToFile("servers.json", c.ServerConfigs)
	if err != nil {
		panic(err)
	}
	err = config.MarshalGroupsToFile("groups.json", c.GroupConfigs)
	if err != nil {
		panic(err)
	}
	// spawn each server process - assume we are in cmd/coordinator
	for _, s := range c.ServerConfigs {
		cmd := exec.Command("../server/server", "../coordinator/servers.json", "../coordinator/groups.json", s.Address)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
		c.processes = append(c.processes, cmd)
	}
	// make sure servers are ready
	time.Sleep(time.Second)
	c.remoteServers = c.Connect(c.ServerConfigs)
	if len(c.ClientConfigs) == 0 {
		c.clientNetType = inprocess
		c.SetupInProcess(0)
		caller, err := network.NewCaller(c.ServerConfigs)
		if err != nil {
			panic(err)
		}
		caller.SetGroups(c.GroupConfigs)
		c.clients.Caller = caller
	} else {
		err = config.MarshalServersToFile("clients.json", c.ClientConfigs)
		if err != nil {
			panic(err)
		}
		// spawn client processes
		for _, s := range c.ClientConfigs {
			cmd := exec.Command("../client/client", "../coordinator/servers.json", "../coordinator/groups.json", "../coordinator/clients.json", s.Address)
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			err := cmd.Start()
			if err != nil {
				panic(err)
			}
			c.processes = append(c.processes, cmd)
		}
		time.Sleep(time.Second)
		// TODO: connect to clients
		c.remoteClients = c.Connect(c.ClientConfigs)
		log.Print(len(c.remoteClients))
	}
	return c
}

func NewInProcessNetwork(numServers, numGroups, groupSize int) *CoordinatorNetwork {
	c := &CoordinatorNetwork{}
	c.clientNetType = inprocess
	c.serverNetType = inprocess
	c.ServerConfigs, c.GroupConfigs, c.ClientConfigs = NewLocalConfig(numServers, numGroups, groupSize, 0, true)
	c.SetupInProcess(numServers)
	return c
}

func NewLocalConfig(numServers, numGroups, groupSize, numClients int, inprocess bool) (map[int64]*config.Server, map[int64]*config.Group, map[int64]*config.Server) {
	serverIds := make([]int64, numServers)
	for i := range serverIds {
		serverIds[i] = int64(i)
	}
	groups := config.CreateSeparateGroupsWithSize(numGroups, groupSize, serverIds)
	servers := make(map[int64]*config.Server)
	for _, id := range serverIds {
		addr := fmt.Sprintf("localhost:%d", base+(int(id)*numServers))
		if inprocess {
			servers[id] = config.CreateServerWithCertificate(addr, id, nil, nil)
		} else {
			servers[id] = config.CreateServerWithExisting(addr, id, servers)
		}
	}
	clients := make(map[int64]*config.Server)
	for i := 0; i < numClients; i++ {
		addr := fmt.Sprintf("localhost:%d", clientBase+(i*numClients))
		clients[int64(i)] = config.CreateServerWithExisting(addr, int64(i), clients)
	}
	return servers, groups, clients
}

func (c *CoordinatorNetwork) SetupInProcess(numServers int) {
	c.servers = make([]*server.Server, numServers)
	mockNetwork := make([]messages.MessageHandlersServer, numServers)
	mockCondNetwork := network.NewMockConnNetwork()
	for i := range c.servers {
		h := server.NewHandler()
		mockNetwork[i] = h
		s := server.NewServer(&config.Servers{Servers: c.ServerConfigs}, &config.Groups{Groups: c.GroupConfigs}, h, c.ServerConfigs[int64(i)].Address)
		c.servers[i] = s
	}
	for i := range c.servers {
		c.servers[i].Caller = network.NewMockCaller(mockNetwork)
		c.servers[i].Caller.SetGroups(c.GroupConfigs)
		c.servers[i].TcpConnections = network.NewConnectionManager(c.ServerConfigs, i)
		c.servers[i].TcpConnections.SetCaller(c.servers[i].Caller)
		c.servers[i].TcpConnections.SetMock(mockCondNetwork)
	}
	c.clients = client.NewClientRunner(c.ServerConfigs, c.GroupConfigs)
	c.clients.Caller = network.NewMockCaller(mockNetwork)
	c.clients.Caller.SetGroups(c.GroupConfigs)
	for _, s := range c.servers {
		s.TcpConnections.LaunchConnects()
	}
}

/*
	for id := i.StartId; id < i.EndId; id++ {
		c.clients.AddClient(id)
	}
	for layer := 0; layer < int(i.NumLayers); layer++ {
		for _, cli := range c.clients.Clients {
			// create a new key pair
			csk, cpk := crypto.NewDHKeyPair()
			// choose a random server
			sid := rand.Intn(len(c.servers))
			clientPathKey := &prepareMessages.PathKey{
				Secret:   csk,
				Public:   cpk,
				ServerID: int64(sid),
				Shared:   csk.SharedKey(&c.clients.C.AuthEncPublicKeys[sid]),
			}
			cli.PathKeys = append(cli.PathKeys, clientPathKey)
		}
	}
	for _, s := range c.servers {
		s.Keys = make([]*processMessages.KeyLookupTable, i.NumLayers)
		for layer := 0; layer < int(i.NumLayers); layer++ {
			s.Keys[layer] = processMessages.NewKeyLookupTable(s.CommonState)
		}
	}

	for _, cli := range c.clients.Clients {
		prev := cli.ID
		next := 0
		var nextKey crypto.DHPublicKey
		for layer := 0; layer < int(i.NumLayers); layer++ {
			pk := cli.PathKeys[layer]
			sid := pk.ServerID
			s := c.servers[sid]
			if layer < int(i.NumLayers)-1 {
				next = int(cli.PathKeys[layer+1].ServerID)
				nextKey = cli.PathKeys[layer+1].Public
			} else {
				next = rand.Intn(len(c.GroupConfigs))
				for _, gsid := range c.GroupConfigs[int64(next)].Servers {
					c.servers[gsid].GroupAliases[int32(next)].CheckpointState.AnonymousSigningKeys.Add(cli.AnonymousPublicKey)
				}
				// ignore?
				nextKey = cli.PathKeys[layer].Public
			}
			s.Keys[layer].AddKey(pk.Public, pk.Shared, int(prev), next, nextKey)
			prev = sid
		}
	}
}
*/

const retries = 3

func (c *CoordinatorNetwork) SendKeys(keyMessages []map[int64]*coord.KeyInformation, publicKeys *coord.KeyInformation) error {
	done := make(chan error)
	for sid, keys := range keyMessages {
		go func(sid int, keys map[int64]*coord.KeyInformation) {
			for i := 0; i < retries; i++ {
				ctx := context.Background()
				if len(keys) == 0 {
					// send public keys
					keys[0] = publicKeys
				}
				ok := true
				for _, k := range keys {
					var err error
					if c.serverNetType == inprocess {
						_, err = c.servers[sid].KeySet(ctx, k)
					} else {
						_, err = c.remoteServers[sid].KeySet(ctx, k)
					}
					if err != nil {
						log.Printf("Attempt %d, sid %d: %v", i, sid, err)
						ok = false
						break
					}
				}
				if ok {
					break
				}
			}
			done <- nil
		}(int(sid), keys)
	}
	for range c.ServerConfigs {
		err := <-done
		if err != nil {
			return err
		}
	}
	if c.clientNetType == inprocess {
		ctx := context.Background()
		_, err := c.clients.KeySet(ctx, publicKeys)
		if err != nil {
			return err
		}
	} else {
		for _, c := range c.remoteClients {
			go func(c coord.CoordinatorHandlerClient) {
				ctx := context.Background()
				_, err := c.KeySet(ctx, publicKeys)
				done <- err
			}(c)
		}
		for range c.remoteClients {
			err := <-done
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CoordinatorNetwork) SendClientStart(i *coord.RoundInfo, numMessages int) error {
	if c.clientNetType == inprocess {
		ctx := context.Background()
		i.StartId = 0
		i.EndId = int64(numMessages)
		_, err := c.clients.ClientStart(ctx, i)
		if err != nil {
			return err
		}
	} else {
		ctx := context.Background()
		numClients := len(c.remoteClients)
		loads := Spans(numMessages, numClients)
		idx := 0
		done := make(chan error)
		for _, c := range c.remoteClients {
			go func(c coord.CoordinatorHandlerClient, idx int) {
				info := i.Copy()
				info.StartId = int64(loads[idx][0])
				info.EndId = int64(loads[idx][1])
				_, err := c.ClientStart(ctx, info)
				done <- err
			}(c, idx)
			idx++
		}
		for range c.remoteClients {
			err := <-done
			if err != nil {
				return err
			}
		}

	}
	return nil
}

func (c *CoordinatorNetwork) CheckClientReceipt(i *coord.RoundInfo, numMessages int) error {
	if c.clientNetType == inprocess {
		ctx := context.Background()
		i.StartId = 0
		i.EndId = int64(numMessages)
		_, err := c.clients.CheckReceipt(ctx, i)
		if err != nil {
			return err
		}
	} else {
		ctx := context.Background()
		numClients := len(c.remoteClients)
		loads := Spans(numMessages, numClients)
		idx := 0
		done := make(chan error)
		for _, c := range c.remoteClients {
			go func(c coord.CoordinatorHandlerClient, idx int) {
				for j := 0; j < numRetries; j++ {
					info := i.Copy()
					info.StartId = int64(loads[idx][0])
					info.EndId = int64(loads[idx][1])
					_, err := c.CheckReceipt(ctx, info)
					if err != nil {
						log.Printf("%d: retry %d %v", idx, j, err)
					} else {
						break
					}
				}
				done <- nil
			}(c, idx)
			idx++
		}
		for range c.remoteClients {
			err := <-done
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *CoordinatorNetwork) SendRoundSetup(i *coord.RoundInfo) error {
	done := make(chan error)
	for idx := range c.ServerConfigs {
		go func(idx int) {
			ctx := context.Background()
			var err error
			if c.serverNetType == inprocess {
				_, err = c.servers[idx].RoundSetup(ctx, i)
			} else {
				_, err = c.remoteServers[idx].RoundSetup(ctx, i)
			}
			done <- err
		}(int(idx))
	}
	for range c.ServerConfigs {
		err := <-done
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CoordinatorNetwork) SendRoundStart(i *coord.RoundInfo) error {
	done := make(chan error)
	for idx := range c.ServerConfigs {
		go func(idx int) {
			var err error
			ctx := context.Background()
			if c.serverNetType == inprocess {
				_, err = c.servers[idx].RoundStart(ctx, i)
			} else {
				_, err = c.remoteServers[idx].RoundStart(ctx, i)
			}
			done <- err
		}(int(idx))
	}
	for range c.ServerConfigs {
		err := <-done
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CoordinatorNetwork) GetMessages(i *coord.RoundInfo) ([][]byte, error) {
	responses := make([][]byte, 0)
	mu := sync.Mutex{}
	done := make(chan error)
	for idx := range c.ServerConfigs {
		go func(idx int) {
			ctx := context.Background()
			var messages *coord.ServerMessages
			var err error
			if c.serverNetType == inprocess {
				messages, err = c.servers[idx].GetMessages(ctx, i)
			} else {
				messages, err = c.remoteServers[idx].GetMessages(ctx, i)
			}
			if err != nil {
				done <- err
			} else {
				if messages != nil {
					mu.Lock()
					responses = append(responses, messages.Messages...)
					mu.Unlock()
				}
				done <- nil
			}
		}(int(idx))
	}
	for range c.ServerConfigs {
		err := <-done
		if err != nil {
			return nil, err
		}
	}
	return responses, nil
}

func (c *CoordinatorNetwork) Connect(cfgs map[int64]*config.Server) []coord.CoordinatorHandlerClient {
	conn, err := network.GetConnections(cfgs)
	if err != nil {
		c.KillAll()
		panic(err)
	}
	clients := make([]coord.CoordinatorHandlerClient, len(conn))
	for id, cc := range conn {
		clients[id] = coord.NewCoordinatorHandlerClient(cc)
	}
	return clients
}

func (c *CoordinatorNetwork) KillAll() {
	for _, p := range c.processes {
		err := p.Process.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("Killing %v", p.Process.Pid)
			p.Process.Kill()
		}
	}
	if c.serverNetType == remote {
		KillRemoteServers(c.ServerConfigs, ServerProcessName)
		// RunRemoteCommandOnEach(c.ServerConfigs, "sudo tc qdisc del dev eth0 root handle 1:0")
	}
	if c.clientNetType == remote {
		KillRemoteServers(c.ClientConfigs, ClientProcessName)
	}
}
func (c *CoordinatorNetwork) SetKill() {
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		c.KillAll()
		log.Fatalf("Manually stopped")
	}()
}

func Spans(total int, numSpans int) [][2]int {
	spans := make([][2]int, numSpans)
	start := 0
	skip := total / numSpans
	extra := total % numSpans
	for i := range spans {
		end := start + skip
		if extra > 0 {
			end++
			extra--
		}
		spans[i] = [2]int{start, end}
		start = end
	}
	return spans
}
