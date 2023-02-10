package network

import (
	"context"
	"crypto/rand"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/network/messages"
)

// don't spend too long lol
const pingTime = time.Second * 20

func (c *Caller) PingServer(sid int) int {
	// do some pings and find the median RTT
	m := &messages.NetworkMessage{Data: make([]byte, 1024)}
	r := config.NewPRGShuffler(rand.Reader)
	cli := c.Network[sid]
	// throw away first result
	cli.HealthCheck(context.Background(), m)
	measurements := make([]int, 0)
	startStart := time.Now()
	for i := 0; i < 9; i++ {
		randSleep := r.Intn(int(time.Second))
		time.Sleep(time.Duration(randSleep))
		start := time.Now()
		cli.HealthCheck(context.Background(), m)
		end := time.Now()
		measurements = append(measurements, int(end.Sub(start)))
		if end.After(startStart.Add(pingTime)) {
			break
		}
	}
	// rough median
	sort.Ints(measurements)
	return measurements[len(measurements)/2]
}

func (c *Caller) HealthCheck() {
	medianPingTimes := make([]int, len(c.Network))
	wg := sync.WaitGroup{}
	for sid := range medianPingTimes {
		wg.Add(1)
		go func(sid int) {
			defer wg.Done()
			medianPingTimes[sid] = c.PingServer(sid)
		}(sid)
	}
	wg.Wait()
	// c.MedianPingTimes = medianPingTimes
	c.ServersSortedByLatency = ArgSort(medianPingTimes)
	log.Printf("Measured times %v", medianPingTimes)
	config.LogTime("Servers %v %v", c.ServersSortedByLatency, medianPingTimes)
}

// from the stack overflow https://stackoverflow.com/questions/31141202/get-the-indices-of-the-array-after-sorting-in-golang

type Slice struct {
	sort.IntSlice
	idx []int
}

func (s Slice) Swap(i, j int) {
	s.IntSlice.Swap(i, j)
	s.idx[i], s.idx[j] = s.idx[j], s.idx[i]
}

func NewSlice(n ...int) *Slice {
	s := &Slice{IntSlice: sort.IntSlice(n), idx: make([]int, len(n))}
	for i := range s.idx {
		s.idx[i] = i
	}
	return s
}

func ArgSort(v []int) []int {
	s := NewSlice(v...)
	sort.Sort(s)
	return s.idx
}
