package waitable

import "sync"

type Waitable interface {
	Done() <-chan struct{}
}

func WaitAll(targets ...Waitable) <-chan struct{} {
	var wg sync.WaitGroup
	merged := make(chan struct{})

	for _, target := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-target.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}
