package errors

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"sync"
)

var mu sync.Mutex
var Addr = "unset"

// write error to log file
func LogError(e error) {
	log.Print(e)
	// only allow the first error - all other errors will block here forever (assuming a panic is coming!)
	mu.Lock()
	f, err := os.Create(fmt.Sprintf("error%s.log", Addr))
	defer f.Close()
	if err == nil {
		b := bufio.NewWriter(f)
		pprof.Lookup("goroutine").WriteTo(b, 1)
		b.WriteString(fmt.Sprintf("%v", e))
		b.Flush()
	}
}

// wrap errors from other sources with stack
func NetworkError(err error) error {
	LogError(err)
	return err
}

func err(reason string) error {
	err := errors.New(reason)
	LogError(err)
	return err
}

func UnimplementedError() error   { return err("Function not implemented") }
func UnrecognizedError() error    { return err("Message type not recognized") }
func SignatureError() error       { return err("Invalid message signature") }
func LengthInvalidError() error   { return err("Message length invalid") }
func BadElementError() error      { return err("Element is not on curve") }
func GroupAgreementError() error  { return err("Anytrust group disagrees") }
func ClientNotFoundError() error  { return err("Client not found") }
func KeyNotFound() error          { return err("Key not found") }
func Duplicate() error            { return err("Duplicate message") }
func MissingMessages() error      { return err("Messages are missing") }
func BadMetadataError() error     { return err("Metadata does not match") }
func WrongServerError() error     { return err("Message sent to wrong server") }
func DecryptionFailure() error    { return err("Unable to decrypt message") }
func ProofFailure() error         { return err("Proof failure") }
func TokenInvalid() error         { return err("Token invalid") }
func CommitFailure() error        { return err("Commitment invalid") }
func WrongReceipt() error         { return err("Receipt incorrect") }
func LinkOverflow() error         { return err("Link overflow") }
func SynchronizationError() error { return err("Multiple messages from same server") }
