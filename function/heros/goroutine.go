package heros

import (
	"sync"
)

var wg = sync.WaitGroup{}

type Glimit struct {
	n int
	c chan struct{}
}

// initialization Glimit struct
func goroutineNew(n int) *Glimit {
	return &Glimit{
		n: n,
		c: make(chan struct{}, n),
	}
}

// Run f in a new goroutine but with limit.
func (g *Glimit) goroutineRun(f func()) {
	g.c <- struct{}{}
	go func() {
		f()
		<-g.c
	}()
	wg.Add(1)
}
