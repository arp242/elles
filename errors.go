package main

import (
	"sync"
)

type errGroup struct {
	MaxSize int
	mu      sync.Mutex
	errs    []error
}

func (g *errGroup) Len() int { return len(g.errs) }

// List all the errors; returns nil if there are no errors.
func (g *errGroup) List() []error {
	if g.Len() == 0 {
		return nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	e := make([]error, len(g.errs))
	copy(e, g.errs)
	return e
}

// Append a new error to the list; this is thread-safe.
//
// It won't do anything if the error is nil, in which case it will return false.
// This makes appending errors in a loop slightly nicer:
//
//	for {
//	    err := do()
//	    if errors.Append(err) {
//	        continue
//	    }
//	}
func (g *errGroup) Append(err error) bool {
	if err == nil {
		return false
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.MaxSize == 0 || len(g.errs) < g.MaxSize {
		g.errs = append(g.errs, err)
	}
	return true
}
