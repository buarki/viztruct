package samples

import (
	"crypto"
	"sync"
	"sync/atomic"
)

type Name string

type SomeStruct struct {
	count     atomic.Int64
	y         bool
	wg        sync.WaitGroup
	z         float64
	mutex     sync.Mutex
	x         bool
	hash      crypto.Hash
	names     []string
	FirstName Name
}

func (s *SomeStruct) DoSomething() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.count.Add(1)
}
