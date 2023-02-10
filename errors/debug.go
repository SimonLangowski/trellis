package errors

import "log"

const debug = false

func DebugPrint(fmt string, args ...interface{}) {
	if debug {
		log.Printf(fmt, args...)
	}
}
