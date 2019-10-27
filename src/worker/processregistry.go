package worker

import "sync"

// Structure used to prevent the worker's goroutines from trying to fetch the response data if another goroutine is
// already processing it. The worker "ticks" take one second, so for example if for some reason fetching of the response
// takes 2 seconds, another one could start doing it, not knowing about the ongoing operation.
// This structure is synchronized between the goroutines and used to "lock" or "unlock" specified objects from being
// processed by goroutines.
type ProgressRegistry struct {
	Ids map[int64]int
	mux sync.Mutex
}

// Locks the object with specified id from processing
func (p *ProgressRegistry) lock(id int64) {
	p.mux.Lock()
	p.Ids[id] = 1
	p.mux.Unlock()
}

// Checks if the object with specified id is locked
func (p *ProgressRegistry) isLocked(id int64) bool {
	p.mux.Lock()
	_, ok := p.Ids[id]
	defer p.mux.Unlock()

	return ok
}

// Unlocks the object for processing
func (p *ProgressRegistry) unlock(id int64) {
	p.mux.Lock()
	delete(p.Ids, id)
	p.mux.Unlock()
}
