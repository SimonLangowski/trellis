package synchronization

import (
	"sync"

	"github.com/simonlangowski/lightning1/errors"
)

// Implement syncrhonized broadcast
// A server receives a message in layer l+1 only after all messages from layer l

// Basically, hold messages in layer l+1 until those from layer l are processed
// With correct functioning, we get one message from each server each layer, and so we know we are done
// Otherwise, if no message is received after a timeout, then start a churn protocol (not implemented)

type Synchronizer struct {
	// current state
	round int
	layer int

	// Ideally we would store all received messages for use in blame protocols

	// track messages
	processed int
	threshold int
	started   []bool
	countLock sync.Mutex
	markLock  sync.Mutex

	// to block rpcs on layer l+1 until ready
	wait *sync.Cond
	lock sync.RWMutex

	callback Callback
}

type Callback interface {
	// called when processed == threshold
	// called with completed layer number
	// return next threshold value and layer
	OnThreshold(int) (int, int)
}

// For submissions before layer 0
const PreRound = -1

// Block all
const Blocked = -2

const InProgress = 0
const Completed = 1

func NewSynchronizer(round int, layer int, threshold int, callback Callback) *Synchronizer {
	s := &Synchronizer{
		round:     round,
		layer:     layer,
		processed: 0,
		threshold: threshold,
		started:   make([]bool, threshold),
		callback:  callback,
	}
	s.wait = sync.NewCond(s.lock.RLocker())
	return s
}

func (s *Synchronizer) Sync(layer int) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for s.layer != layer {
		s.wait.Wait()
	}
}

func (s *Synchronizer) SyncOnce(layer int, id int) error {
	s.Sync(layer)
	s.markLock.Lock()
	defer s.markLock.Unlock()
	if id > len(s.started) || id < 0 {
		return errors.BadMetadataError()
	}
	if s.started[id] {
		return errors.SynchronizationError()
	}
	s.started[id] = true
	return nil
}

func (s *Synchronizer) Done() {
	s.countLock.Lock()
	defer s.countLock.Unlock()
	s.processed += 1
	if s.processed == s.threshold {
		go s.Trigger()
	} else if s.processed > s.threshold {
		panic("More clients passed SyncOnce than threshold")
	}
}

func (s *Synchronizer) Trigger() {
	s.countLock.Lock()
	defer s.countLock.Unlock()
	s.lock.Lock()
	defer s.lock.Unlock()
	s.markLock.Lock()
	defer s.markLock.Unlock()
	if s.callback != nil {
		s.threshold, s.layer = s.callback.OnThreshold(s.layer)
	} else {
		s.layer++
	}
	s.started = make([]bool, s.threshold)
	s.processed = 0
	s.wait.Broadcast()
}

func (s *Synchronizer) Reset(round int, layer int, threshold int) {
	s.countLock.Lock()
	defer s.countLock.Unlock()
	s.lock.Lock()
	defer s.lock.Unlock()
	s.markLock.Lock()
	defer s.markLock.Unlock()
	s.round = round
	s.layer = layer
	s.processed = 0
	s.threshold = threshold
	s.started = make([]bool, threshold)

	s.wait.Broadcast()
}
