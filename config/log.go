package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	sync "sync"
)

var logger *log.Logger = nil
var mu sync.Mutex
var buf *bufio.Writer

func InitLogger(id int) {
	mu.Lock()
	defer mu.Unlock()
	if logger != nil {
		return
	}
	f, err := os.Create(fmt.Sprintf("log%d.log", id))
	if err != nil {
		panic(err)
	}
	buf = bufio.NewWriter(f)
	logger = log.New(buf, fmt.Sprintf("%d: ", id), log.Lmicroseconds|log.Ltime)
}

func LogTime(m string, details ...interface{}) {
	if LogTimes && logger != nil {
		logger.Printf(m, details...)
	}
}

func Flush() {
	buf.Flush()
}
