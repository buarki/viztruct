package samples

import (
	"sync/atomic"
)

type BadLayout struct {
	Active    bool
	count     atomic.Int64
	ID        uint64
	Confirmed bool
	Age       uint32
}

/*
type SomeStruct struct {
	count atomic.Int64
	x     bool
	y     bool
	wg    sync.WaitGroup
	z     float64
	mutex sync.Mutex
	hash  crypto.Hash
	names []string
}

/*
type SomeStruct2 struct {
	mutex2 sync.Mutex
	wg     sync.WaitGroup
	value  float64
}*/

/*
type SomeStruct3 struct {
	myChan chan prometheus.Counter
	mutex3 sync.Mutex
	wg     sync.WaitGroup
	next   *SomeStruct3
	sss    structi.Field
	xx     otel.ErrorHandler
}
*/
