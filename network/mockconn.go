package network

import (
	"bytes"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/simonlangowski/lightning1/errors"
)

type MockConnNetwork struct {
	mu    sync.Mutex
	cond  *sync.Cond
	conns map[string]*MockConn
}

// implement net.Conn interface
type MockChan struct {
	mu     sync.Mutex
	cond   *sync.Cond
	buffer bytes.Buffer
}

type MockConn struct {
	in  *MockChan
	out *MockChan
	s1  int
	s2  int
}

func NewMockChan() *MockChan {
	m := &MockChan{}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func NewMockConnPair(s1, s2 int) (*MockConn, *MockConn) {
	c1 := NewMockChan()
	c2 := NewMockChan()
	return &MockConn{in: c1, out: c2, s1: s1, s2: s2}, &MockConn{in: c2, out: c1, s1: s2, s2: s1}
}

func NewMockConnNetwork() *MockConnNetwork {
	n := &MockConnNetwork{conns: make(map[string]*MockConn)}
	n.cond = sync.NewCond(&n.mu)
	return n
}

func (c *MockChan) Read(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.buffer.Bytes()) < len(b) {
		c.cond.Wait()
	}
	n, err := c.buffer.Read(b)
	// cleanup memory
	if len(c.buffer.Bytes()) == 0 {
		c.buffer.Reset()
	}
	return n, err
}

func (c *MockChan) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// will not wake until after we release the lock
	c.cond.Broadcast()
	return c.buffer.Write(b)
}

func (c *MockConn) Read(b []byte) (int, error) {
	return c.in.Read(b)
}

func (c *MockConn) Write(b []byte) (int, error) {
	return c.out.Write(b)
}

func (c *MockConn) Close() error {
	return nil
}

type MockAddr struct {
	serverId int
}

func (c *MockConn) LocalAddr() net.Addr {
	return &MockAddr{c.s1}
}

func (c *MockConn) RemoteAddr() net.Addr {
	return &MockAddr{c.s2}
}

func (m *MockAddr) Network() string {
	return "Mock"
}

func (m *MockAddr) String() string {
	return fmt.Sprintf("%d", m.serverId)
}

func (c *MockConn) SetDeadline(t time.Time) error {
	panic(errors.UnimplementedError())
}

func (c *MockConn) SetReadDeadline(t time.Time) error {
	panic(errors.UnimplementedError())
}

func (c *MockConn) SetWriteDeadline(t time.Time) error {
	panic(errors.UnimplementedError())
}

func (c *ConnectionManager) MockListen(address string, from int) (net.Conn, error) {
	id := int(c.MyCfg.Id)
	c1, c2 := NewMockConnPair(from, id)
	errors.DebugPrint("accept: %v->%v %v", from, id, address)
	c.Mock.mu.Lock()
	defer c.Mock.mu.Unlock()
	if c.Mock.conns[address] != nil {
		return nil, fmt.Errorf("bind %v already in use", address)
	}
	c.Mock.conns[address] = c2
	c.Mock.cond.Broadcast()
	return c1, nil
}

func (c *ConnectionManager) MockConnect(address string, to int) (net.Conn, error) {
	c.Mock.mu.Lock()
	defer c.Mock.mu.Unlock()
	errors.DebugPrint("Looking for: %v", address)
	for {
		conn, exists := c.Mock.conns[address]
		if exists {
			return conn, nil
		}
		c.Mock.cond.Wait()
	}
}

// func WrapMessage(m *messages.SignedMessage) *ConnectionReader {
// 	// signature will be checked in signed encryption
// 	c := &ConnectionReader{
// 		numMessages:   1,
// 		messageSize:   len(m.Data),
// 		buff:          make(chan []byte, 1),
// 		baseBatchSize: 1,
// 	}
// 	c.buff <- m.Data
// 	close(c.buff)
// 	return c
// }

func (c *ConnectionManager) SetMock(m *MockConnNetwork) {
	c.Mock = m
	c.LaunchAccepts()
}
