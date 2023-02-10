package network

import (
	"runtime"
	"sync"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/common"
)

var numWorkers = runtime.NumCPU()

func CreateGroupMessages(chunks map[int]*buffers.MemReadWriter, c *common.CommonState, layer int, t messages.NetworkMessage_MessageType) ([]int, [][]byte) {
	wg := sync.WaitGroup{}
	signedMessages := make([][]byte, len(chunks))
	messageCounts := make([]int, len(chunks))
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := i; j < len(signedMessages); j += numWorkers {
				f := chunks[j]
				f.Shuffle(false)
				messageCounts[j] = f.NumMessages()
				lengthLeft := f.Len()
				sm := messages.NewSignedMessage(lengthLeft, c.Round, c.Layer, c.MyId, j, 0, f.NumMessages(), t)
				f.ReadNextChunk(sm.Data)
				PreHashSign(c.SecretSigningKey, sm)
				signedMessages[j] = sm.AsArray()
			}
		}(i)
	}
	wg.Wait()
	return messageCounts, signedMessages
}

// no need for dummies, but do shuffle
func (c *ConnectionManager) SendGroupShuffleMessages(chunks map[int]*buffers.MemReadWriter, common *common.CommonState, t messages.NetworkMessage_MessageType, responseSize int) ([]int, error) {
	// first compile and sign all the messages.  No dummies so no extra memory overhead
	config.LogTime("Starting signing %d", common.Layer)
	messageCounts, groupMessages := CreateGroupMessages(chunks, common, common.Layer, t)
	config.LogTime("Finished signing %d", common.Layer)
	// then by order of latency send messages to group members
	// need a reverse group lookup I guess
	// done := make(chan error)
	jobs := c.caller.GetJobs()
	reverseGroups := c.ReverseGroups()
	inProgress := make([]chan error, len(jobs))
	// for i := 0; i < numWorkers; i++ {
	// 	go func() {
	for {
		sid, ok := <-jobs
		if !ok {
			break
		}
		b := make([]byte, 0)
		for _, gid := range reverseGroups[sid] {
			b = append(b, groupMessages[gid]...)
		}
		//timeout := BandwidthTimeout(len(b))
		var err error
		inProgress[sid], err = c.Send(b, sid)
		if err != nil {
			return nil, err //done <- err
		}
	}
	// 		done <- nil
	// 	}()
	// }
	// for i := 0; i < numWorkers; i++ {
	// 	err := <-done
	// 	if err != nil {
	// 		return messageCounts, err
	// 	}
	// }
	go c.FinishSends(inProgress)
	return messageCounts, nil
}
