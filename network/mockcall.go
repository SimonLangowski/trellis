package network

import (
	"context"
	"io"

	"github.com/simonlangowski/lightning1/errors"
	"github.com/simonlangowski/lightning1/network/messages"
	"google.golang.org/grpc/metadata"
)

var MockLatency = 0             // millisecond
var MockBandwidth = 10000000000 // MB/s

type MockCall struct {
	data     []*messages.NetworkMessage
	response *messages.NetworkMessage
}

type MockCallStream struct {
	send chan *messages.NetworkMessage
	recv chan *messages.NetworkMessage
}

func NewMockCall(chunks []*messages.NetworkMessage) *MockCall {
	m := &MockCall{data: make([]*messages.NetworkMessage, len(chunks))}
	for i := range chunks {
		m.data[i] = copyMessage(chunks[i])
	}
	return m
}

func NewMockCallStream(s messages.MessageHandlersServer) *MockCallStream {
	r := &MockCallStream{
		send: make(chan *messages.NetworkMessage, 10),
		recv: make(chan *messages.NetworkMessage, 10),
	}
	go func() {
		s.HandleSignedMessageStream(r)
	}()
	return &MockCallStream{
		send: r.recv,
		recv: r.send,
	}
}

func (m *MockCall) GetResponse() *messages.NetworkMessage {
	return copyMessage(m.response)
}

// making a copy allows modification to the received version to not propagate back to the sender
func copyMessage(m *messages.NetworkMessage) *messages.NetworkMessage {
	m2 := &messages.NetworkMessage{
		MessageType: m.MessageType,
		Data:        make([]byte, len(m.Data)),
		Signature:   make([]byte, len(m.Signature)),
	}
	copy(m2.Data, m.Data)
	copy(m2.Signature, m.Signature)
	return m2
}

func (m *MockCallStream) Recv() (*messages.NetworkMessage, error) {
	ms, ok := <-m.recv
	if !ok {
		return nil, io.EOF
	}
	return ms, nil
}

func (m *MockCallStream) Send(ms *messages.NetworkMessage) error {
	m.send <- copyMessage(ms)
	return nil
}

// unused methods

func (m *MockCallStream) Context() context.Context {
	return context.Background()
}

func (m *MockCallStream) RecvMsg(interface{}) error {
	return errors.UnimplementedError()
}

func (m *MockCallStream) SendHeader(metadata.MD) error {
	return errors.UnimplementedError()
}

func (m *MockCallStream) SendMsg(interface{}) error {
	return errors.UnimplementedError()
}

func (m *MockCallStream) SetHeader(metadata.MD) error {
	return errors.UnimplementedError()
}

func (m *MockCallStream) SetTrailer(metadata.MD) {
}

func (m *MockCallStream) CloseSend() error {
	close(m.send)
	return nil
}

func (m *MockCallStream) Header() (metadata.MD, error) {
	return metadata.MD{}, nil
}

func (m *MockCallStream) Trailer() metadata.MD {
	return metadata.MD{}
}
