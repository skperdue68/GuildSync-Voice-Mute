package main

import (
	"sync"
	"time"
)

// Native reads run on one goroutine. The callback must not perform network I/O.
func pollShortcut(read func() (bool, error), closeNative func(), edge func(bool), failed func(error)) func() {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		defer closeNative()
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				down, err := read()
				if err != nil {
					edge(false)
					failed(err)
					return
				}
				edge(down)
			}
		}
	}()
	return func() { once.Do(func() { close(done) }) }
}
