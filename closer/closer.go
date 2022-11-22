package closer

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
)

var (
	mu      sync.Mutex
	lastId  int
	closers map[int]Closer
)

type Closer func() error

func init() {
	closers = make(map[int]Closer)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go handle(c)
}

func Add(closer Closer) int {
	mu.Lock()
	lastId += 1
	id := lastId

	closers[id] = closer
	mu.Unlock()
	return id

}

func Remove(ids ...int) {
	mu.Lock()
	for _, id := range ids {
		delete(closers, id)
	}
	mu.Unlock()
}

func Close(ids ...int) error {
	var retErr error
	mu.Lock()
	for _, id := range ids {
		if err := close(id); err != nil {
			retErr = err
		}
	}
	mu.Unlock()

	return retErr
}

func close(id int) error {
	closer, ok := closers[id]
	if !ok {
		return fmt.Errorf("closer %d: not found", id)
	}

	delete(closers, id)
	return closer()
}

func handle(c chan os.Signal) {
	<-c
	mu.Lock()
	for i := 0; i <= lastId; i++ {
		if closer, ok := closers[i]; ok {
			if err := closer(); err != nil {
				log.Println("closer.handle:", err)
			}
		}
	}
	mu.Unlock()
}
