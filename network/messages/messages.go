package messages

// Message things to be imported by things implementing RPCs
// Common tasks like checking message signatures and synchronizing

type RPCMessage interface {
	Len() int
	InterpretFrom([]byte) error
	PackTo([]byte)
}
