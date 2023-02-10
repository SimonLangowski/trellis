package network

import (
	"hash"
	"io"
	"net"
	"time"

	"github.com/simonlangowski/lightning1/config"
	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/buffers"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/common"
)

// Should batch size be number of messages, or a number of bytes?
// Should the buffer be reused?  Probably not if you're passing it to other gofunctions
// But that means a memory allocation again :/

// For now, this just batches to limit syscalls
// But could be for batch eddsa verification

type ConnectionReader struct {
	numMessages   int
	messageSize   int
	baseBatchSize int
	conn          net.Conn
	Signature     []byte
	Buff          chan []byte
	Err           error
}

func NewConnectionReader(numMessages, messageSize, baseBatchSize int, excludeDummies bool, conn net.Conn) *ConnectionReader {
	if conn == nil {
		panic("Nil connection")
	}
	if baseBatchSize <= 0 {
		baseBatchSize = 1
	}
	return &ConnectionReader{
		numMessages:   numMessages,
		messageSize:   messageSize,
		conn:          conn,
		Buff:          make(chan []byte, config.BatchSize+baseBatchSize),
		baseBatchSize: baseBatchSize,
		Signature:     make([]byte, crypto.SIGNATURE_SIZE),
	}
}

func (c *ConnectionReader) ContinuousReader(m *messages.Metadata) {
	for i := 0; i < c.numMessages; i += c.baseBatchSize {
		baseBatchSize := c.baseBatchSize
		if c.numMessages-i < c.baseBatchSize {
			baseBatchSize = c.numMessages - i
		}
		b := make([]byte, c.messageSize*baseBatchSize)
		readStart := time.Now()
		_, err := io.ReadFull(c.conn, b)
		config.LogTime("Read: %v part %d in %v", m, i, time.Since(readStart))
		if err != nil {
			c.Err = errors.NetworkError(err)
			close(c.Buff)
			return
		} else {
			for pos := 0; pos < len(b); pos += c.messageSize {
				c.Buff <- b[pos : pos+c.messageSize]
			}
		}
	}
	io.ReadFull(c.conn, c.Signature)
	close(c.Buff)
}

func (c *ConnectionReader) CheckSignature(h hash.Hash, vk *crypto.ExpandedVerificationKey) bool {
	return crypto.PreHashVerify(h, vk, c.Signature)
}

// send a message to the destination with timeout (e.g because the tcp connection blocks because a buffer is full)
// the write will continue even on timeout.  Call FinishSends() to ensure the write has finished
// func (c *ConnectionManager) trySend(b []byte, dest int, timeout time.Duration) (chan error, error) {
// 	return trySend(c.OutgoingConnections[dest], b, timeout, dest)
// }

// func trySend(c net.Conn, b []byte, timeout time.Duration, dest int) (chan error, error) {
// 	done := make(chan error)
// 	go func(c net.Conn, b []byte) {
// 		start := time.Now()
// 		err := send(c, b)
// 		b = nil
// 		config.LogTime("Sent to %v at %v for %v (Expected: %v)", dest, start, time.Since(start), timeout)
// 		done <- err
// 	}(c, b)
// 	// time.Sleep(timeout)
// 	select {
// 	case err := <-done:
// 		return nil, err
// 	default:
// 		return done, nil
// 	}
// }

func (c *ConnectionManager) Send(b []byte, dest int) (chan error, error) {
	err := send(c.OutgoingConnections[dest], b)
	return nil, err
}

func (c *ConnectionManager) SendShuffleMessages(Messages map[int]*buffers.MemReadWriter, common *common.CommonState, layer int, t messages.NetworkMessage_MessageType) error {
	m := messages.Metadata{
		Type:        t,
		Round:       common.Round,
		Layer:       layer,
		Sender:      common.MyId,
		NumMessages: uint32(common.BinSize),
	}
	inProgress, err := c.SendSignedMessageChunks(&m, Messages, common)
	go c.FinishSends(inProgress)
	return err
}

func (c *ConnectionManager) SendSignedMessageChunks(m *messages.Metadata, Messages map[int]*buffers.MemReadWriter, common *common.CommonState) ([]chan error, error) {
	// done := make(chan error)
	jobs := c.caller.GetJobs()
	// timeout := BandwidthTimeout(Messages[0].Len())
	inProgress := make([]chan error, len(Messages))
	// for i := 0; i < numWorkers; i++ {
	// 	go func() {
	for {
		sid, ok := <-jobs
		if !ok {
			break
		}
		f := Messages[sid]
		f.Shuffle(!config.NoDummies)
		sm := messages.NewSignedMessage(f.Len(), m.Round, m.Layer, m.Sender, 0, sid, f.NumMessages(), m.Type)
		r, err := f.ReadNextChunk(sm.Data)
		if err != nil {
			return inProgress, err //done <- err
		}
		sm.Data = sm.Data[:r]
		PreHashSign(c.MyCfg.SignatureKey, sm)
		inProgress[sid], err = c.Send(sm.AsArray(), sid)
		if err != nil {
			return inProgress, err //done <- err
		}
	}
	// 		done <- nil
	// 	}()
	// }
	// for i := 0; i < numWorkers; i++ {
	// 	err := <-done
	// 	if err != nil {
	// 		return inProgress, err
	// 	}
	// }
	return inProgress, nil
}

// ensure all writes have finished
func (c *ConnectionManager) FinishSends(inProgress []chan error) error {
	for _, c := range inProgress {
		if c == nil {
			continue
		}
		err := <-c
		if err != nil {
			return err
		}
	}
	return nil
}

func CalculateBatchSize(readSize int, messageSize int) int {
	if messageSize >= readSize || messageSize <= 0 {
		return 1
	} else {
		return readSize / messageSize
	}
}

func (c *ConnectionReader) BandwidthTimeout() time.Duration {
	readSize := c.baseBatchSize * c.messageSize
	return BandwidthTimeout(readSize)
}

func BandwidthTimeout(dataLen int) time.Duration {
	bandwidthBytesPerSecond := float64(config.Bandwidth) * (1000000 / 8)
	return time.Duration(float64(dataLen) * float64(time.Second) / bandwidthBytesPerSecond)
}

func (c *ConnectionReader) Send(b []byte) error {
	return send(c.conn, b)
}

func send(c net.Conn, b []byte) error {
	for written := 0; written < len(b); {
		n, err := c.Write(b[written:])
		written += n
		if err != nil {
			return err
		}
	}
	return nil
}

func PreHashSign(key []byte, m *messages.SignedMessage) {
	m.Signature = crypto.PreHashSign(m.GetSignedData(), key)
}
