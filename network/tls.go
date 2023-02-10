package network

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/network/messages"
)

// have each connect to half and use bidirectional connections
const defaultPort = "8001"

type ConnectionManager struct {
	Mock                *MockConnNetwork
	configs             map[int64]*config.Server
	MyCfg               *config.Server
	OutgoingConnections []net.Conn
	IncomingConnections []net.Conn
	locks               []sync.Mutex
	caller              *Caller
	terminated          bool
}

func NewConnectionManager(cfgs map[int64]*config.Server, id int) *ConnectionManager {
	c := &ConnectionManager{
		configs:             cfgs,
		MyCfg:               cfgs[int64(id)],
		OutgoingConnections: make([]net.Conn, len(cfgs)),
		IncomingConnections: make([]net.Conn, len(cfgs)),
		locks:               make([]sync.Mutex, len(cfgs)),
	}
	selfConnectionIn, selfConnectionOut := NewMockConnPair(id, id)
	c.IncomingConnections[id] = selfConnectionIn
	c.OutgoingConnections[id] = selfConnectionOut
	go c.CatchInterrupt()
	return c
}

func (c *ConnectionManager) ShutDown() {
	c.terminated = true
	for _, conn := range c.OutgoingConnections {
		if conn != nil {
			conn.Close()
		}
	}
	for _, conn := range c.IncomingConnections {
		if conn != nil {
			conn.Close()
		}
	}
}

func (c *ConnectionManager) CatchInterrupt() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	c.ShutDown()
}

func (c *ConnectionManager) Accept(from int) (net.Conn, error) {
	ip, port := CalculateAddress(c.MyCfg.Address, from)
	if c.Mock != nil {
		return c.MockListen(ip+port, from)
	}
	cer, err := tls.X509KeyPair(c.MyCfg.Identity, c.MyCfg.PrivateIdentity)
	if err != nil {
		panic(err)
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(c.configs[int64(from)].Identity)
	if !ok {
		panic("Could not create cert pool for TLS connection")
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCertPool}
	ln, err := tls.Listen("tcp", port, config)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer ln.Close()
	return ln.Accept()
}

func (c *ConnectionManager) Connect(id int) (net.Conn, error) {
	s := c.configs[int64(id)]
	ip, port := CalculateAddress(s.Address, int(c.MyCfg.Id))
	if c.Mock != nil {
		return c.MockConnect(ip+port, id)
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(s.Identity)
	if !ok {
		panic("Could not create cert pool for TLS connection")
	}
	certificate, err := tls.X509KeyPair(c.MyCfg.Identity, c.MyCfg.PrivateIdentity)
	if err != nil {
		panic(err)
	}
	conf := &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{certificate},
		// InsecureSkipVerify: true,
	}
	return tls.Dial("tcp", ip+port, conf)
}

func CalculateAddress(address string, offset int) (string, string) {
	ip := config.IP(address)
	port := config.Port(address)
	if len(port) > 1 {
		portNum, err := strconv.Atoi(port[1:])
		if err != nil {
			panic(err)
		}
		portNum += offset + 1000
		return ip, fmt.Sprintf(":%d", portNum)
	} else {
		return ip, ":" + defaultPort
	}
}

func (c *ConnectionManager) ReadMetadata(src int) (*messages.Metadata, []byte, error) {
	m := make([]byte, messages.Metadata_size)
	_, err := io.ReadFull(c.IncomingConnections[src], m)
	if err != nil {
		if c.terminated {
			return nil, nil, io.EOF
		}
		return nil, nil, err
	}
	metadata := &messages.Metadata{}
	metadata.InterpretFrom(m)
	return metadata, m, nil
}

func (c *ConnectionManager) SetCaller(caller *Caller) {
	c.caller = caller
}

func (c *ConnectionManager) ReverseGroups() [][]int {
	return c.caller.ReverseGroups
}

func (c *ConnectionManager) ReadReverse(sid int, length int) (*messages.SignedMessage, error) {
	conn := c.OutgoingConnections[sid]
	sm := messages.NewSignedMessage(length, 0, 0, 0, 0, 0, 0, 0)
	_, err := io.ReadFull(conn, sm.Raw)
	if err != nil {
		return nil, err
	}
	sm.Metadata.InterpretFrom(sm.Raw[:messages.Metadata_size])
	sm.Signature = sm.Raw[len(sm.Raw)-crypto.SIGNATURE_SIZE:]
	return sm, nil
}
