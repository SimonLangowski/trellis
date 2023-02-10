package errors

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

var running = false

func MonitorMemory(name string, id int, interval int64) {
	mu.Lock()
	defer mu.Unlock()
	if !running {
		running = true
		go func() {
			for i := 0; i < 100; i++ {
				time.Sleep(time.Second * time.Duration(interval))
				f, err := os.Create(fmt.Sprintf("%s%d-%d.pprof", name, id, i))
				if err != nil {
					log.Fatal(err)
				}
				runtime.GC() // get up-to-date statistics
				if err := pprof.WriteHeapProfile(f); err != nil {
					log.Fatal("could not write memory profile: ", err)
				}
				f.Close()
			}
		}()
	}
}
