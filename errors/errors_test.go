package errors

import (
	"log"
	"testing"
)

func TestErrorPrinting(t *testing.T) {
	log.Print(UnimplementedError())
}
