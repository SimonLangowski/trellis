package server

import (
	"crypto/sha512"
	"runtime"
	"sync"

	"github.com/simonlangowski/lightning1/crypto"
	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network"
	"github.com/simonlangowski/lightning1/network/messages"
	"github.com/simonlangowski/lightning1/server/checkpoint"
)

type Job struct {
	idx      int
	m        *messages.Metadata
	Message  []byte
	Response []byte
	wg       *sync.WaitGroup
}

// type SignatureJob struct {
// 	Message   []byte
// 	Signature crypto.Signature
// 	Vk        crypto.VerificationKey
// 	Evk       *crypto.ExpandedVerificationKey
// 	done      chan bool
// }

type WorkPool struct {
	jobs         chan Job
	errorHandler func(error)
	s            *Server
}

var numWorkers = runtime.NumCPU()

func NewWorkPool(h *Handlers, s *Server) *WorkPool {
	w := &WorkPool{
		jobs:         make(chan Job, 100),
		errorHandler: h.errorHandler,
		s:            s,
	}
	for i := 0; i < numWorkers; i++ {
		go w.ProcessThread()
	}
	return w
}

func (s *Server) WorkerPoolProcessStream(m *messages.Metadata, metadataBytes []byte, stream *network.ConnectionReader) error {
	wg := sync.WaitGroup{}
	h := sha512.New()
	h.Write(metadataBytes)
	err := s.checkMessage(m)
	defer s.synchronizer.Done()
	if err != nil {
		return err
	}
	idx := 0
	for message := range stream.Buff {
		h.Write(message)
		idx++
		// discard dummies?
		allZero := true
		// zero key does not appear
		for i := 0; i < crypto.KEY_SIZE; i++ {
			if message[i] != 0 {
				allZero = false
				break
			}
		}
		if !allZero {
			wg.Add(1)
			s.pool.jobs <- Job{
				idx:     idx,
				m:       m,
				Message: message,
				wg:      &wg,
			}
		}
	}
	if stream.Err != nil {
		return stream.Err
	}
	// check signature
	if !stream.CheckSignature(h, s.CommonState.ExpandedVerificationKeys[m.Sender]) {
		errors.DebugPrint("Verifying %v %v %v", m, s.CommonState.VerificationKeys[m.Sender], stream.Signature)
		return errors.SignatureError()
	}
	wg.Wait()
	return nil
}

func (s *Server) WorkerPoolProcessGroup(m *messages.Metadata, metadataBytes []byte, stream *network.ConnectionReader) error {
	wg := sync.WaitGroup{}
	h := sha512.New()
	h.Write(metadataBytes)
	group := s.GroupAliases[m.Group]
	// the purpose of this is to track when the round is complete
	// e.g After I have received all of the messages for this group
	var response *messages.SignedMessage
	if m.Type == messages.NetworkMessage_GroupCheckpointSignature {
		err := group.checkpointSynchronizer.SyncOnce(int(m.Layer), int(m.Sender))
		if err != nil {
			return err
		}
		defer group.checkpointSynchronizer.Done()
	} else if m.Type == messages.NetworkMessage_GroupCheckpointToken {
		response = messages.NewSignedMessage(checkpoint.RESPONSE_LENGTH*int(m.NumMessages), s.CommonState.Round, s.CommonState.NumLayers, s.CommonState.MyId, int(m.Group), m.Sender, int(m.NumMessages), messages.NetworkMessage_GroupCheckpointToken)
	}
	pos := 0
	for message := range stream.Buff {
		h.Write(message)
		wg.Add(1)
		j := Job{
			m:       m,
			Message: message,
			wg:      &wg,
		}
		if response != nil {
			j.Response = response.Data[pos : pos+checkpoint.RESPONSE_LENGTH]
		}
		s.pool.jobs <- j
		pos += checkpoint.RESPONSE_LENGTH
	}
	if stream.Err != nil {
		return stream.Err
	}
	// check signature
	if !stream.CheckSignature(h, s.CommonState.ExpandedVerificationKeys[m.Sender]) {
		// errors.DebugPrint("Verifying %v %v %v", m, s.CommonState.VerificationKeys[m.Sender], stream.Signature)
		return errors.SignatureError()
	}
	wg.Wait()
	if response != nil {
		s.CommonState.Sign(response)
		return stream.Send(response.AsArray())
	} else {
		return nil
	}
}

func (w *WorkPool) ProcessThread() {
	for job := range w.jobs {
		var err error
		metadata := job.m
		stream := job.Message
		// start := time.Now()
		switch job.m.Type {
		case messages.NetworkMessage_ServerMessageForward:
			err = w.s.handleLightningMessage(metadata, stream)
			// config.LogTime("Checked %v part %d in %v", metadata, job.idx, time.Since(start))
			// boomerang back
		case messages.NetworkMessage_ServerMessageReverse:
			err = w.s.HandleBoomerangMessage(metadata, stream)
			// path establishment forwards
		case messages.NetworkMessage_PathMessageForward:
			err = w.s.handlePathMessage(metadata, stream)

			// Check Tokens
		case messages.NetworkMessage_GroupCheckpointToken:
			err = w.s.GroupAliases[metadata.Group].CheckpointState.HandleCheckpointMessage(metadata, stream, job.Response)
			// Check final decryption
		case messages.NetworkMessage_GroupCheckpointSignature:
			err = w.s.GroupAliases[metadata.Group].CheckpointState.HandleTrusteeMessage(metadata, stream)
		default:
			err = errors.UnrecognizedError()
		}
		if err != nil {
			w.errorHandler(err)
		}
		job.wg.Done()

	}
}

// func (w *WorkPool) WorkerThread() {
// 	for {
// 		select {
// 		case j := <-w.sendingJobs:
// 			// sign data for outgoing
// 		default:
// 			select {
// 			case j := <-w.sendingJobs:
// 				// sign data for outgoing
// 			case j := <-w.jobs:
// 				// process received data
// 			}
// 		}
// 	}
// }

// func (w *WorkPool) SignatureCoProcessor(signatureJobs chan *SignatureJob) {
// 	// check signatures in parallel with decryption
// 	// TODO: batch verify when len(signatureJobs) is high (e.g we have a backup of things to verify)
// 	for j := range signatureJobs {
// 		if j.Evk != nil {
// 			j.done <- crypto.VerifyExpanded(j.Evk, j.Message, j.Signature)
// 		} else {
// 			j.done <- crypto.Verify(j.Vk, j.Message, j.Signature)
// 		}
// 	}
// }
