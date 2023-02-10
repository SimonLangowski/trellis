package network

import (
	"encoding/binary"
	"sync"
)

func (c *ConnectionManager) LaunchAccepts() {
	for k := range c.configs {
		if k == c.MyCfg.Id {
			continue
		}
		go func(s int) {
			conn, err := c.Accept(s)
			if err != nil {
				panic(err)
			}
			c.IncomingConnections[s] = conn
			// log.Printf("Accepted %s -> %s", conn.RemoteAddr(), conn.LocalAddr())
			// hacky: force handshake
			b := [4]byte{}
			conn.Read(b[:])
			if int(binary.LittleEndian.Uint32(b[:])) != s {
				panic("Wrong server connected to accept")
			}
		}(int(k))
	}
}

func (c *ConnectionManager) LaunchConnects() {
	wg := sync.WaitGroup{}
	for k := range c.configs {
		if k == c.MyCfg.Id {
			continue
		}
		wg.Add(1)
		go func(s int) {
			conn, err := c.Connect(s)
			if err != nil {
				panic(err)
			}
			c.OutgoingConnections[s] = conn
			// log.Printf("Connected %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
			// hacky: force handshake
			b := [4]byte{}
			binary.LittleEndian.PutUint32(b[:], uint32(c.MyCfg.Id))
			conn.Write(b[:])
			wg.Done()
		}(int(k))
	}
	wg.Wait()
}
