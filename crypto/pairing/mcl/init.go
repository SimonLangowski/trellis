package mcl

// from https://github.com/herumi/mcl/blob/master/ffi/go/mcl/init.go

/*
#cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6 -DMCLBN_FR_UNIT_SIZE=4
#cgo LDFLAGS:-lmclbn384_256 -lmcl
#include <mcl/bn.h>
*/
import "C"
import "fmt"

// Init --
// call this function before calling all the other operations
// this function is not thread safe
func Init(curve int) error {
	err := C.mclBn_init(C.int(curve), C.MCLBN_COMPILED_TIME_VAR)
	if err != 0 {
		return fmt.Errorf("ERR mclBn_init curve=%d", curve)
	}
	return nil
}

// called by golang before using this package
// https://golang.org/doc/effective_go#init
func init() {
	err := Init(BLS12_381)
	if err != nil {
		panic(err)
	}
	initSizes()
}
